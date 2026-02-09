package analyzer

import (
	"context"
	"fmt"
	"go/ast"
	"time"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/graph"
	"github.com/miyamo2/braider/internal/loader"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

// DefaultPackageWaitTimeout is the default timeout for waiting for all packages to be scanned.
const DefaultPackageWaitTimeout = 10 * time.Second

// AppAnalyzer detects annotation.App(main) and generates bootstrap code.
func AppAnalyzer(
	appDetector detect.AppDetector,
	injectRegistry *registry.InjectorRegistry,
	provideRegistry *registry.ProviderRegistry,
	packageLoader loader.PackageLoader,
	packageTracker *registry.PackageTracker,
	bootstrapCtx context.Context,
	graphBuilder *graph.DependencyGraphBuilder,
	sorter *graph.TopologicalSorter,
	bootstrapGen generate.BootstrapGenerator,
	fixBuilder report.SuggestedFixBuilder,
	diagnosticEmitter report.DiagnosticEmitter,
) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "braider_app",
		Doc:  "detects App annotation and generates IIFE bootstrap code",
		Run: NewAppAnalyzeRunner(
			appDetector,
			injectRegistry,
			provideRegistry,
			packageLoader,
			packageTracker,
			bootstrapCtx,
			graphBuilder,
			sorter,
			bootstrapGen,
			fixBuilder,
			diagnosticEmitter,
		).Run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

type AppAnalyzeRunner struct {
	appDetector       detect.AppDetector
	injectRegistry    *registry.InjectorRegistry
	provideRegistry   *registry.ProviderRegistry
	packageLoader     loader.PackageLoader
	packageTracker    *registry.PackageTracker
	bootstrapCtx      context.Context
	graphBuilder      *graph.DependencyGraphBuilder
	sorter            *graph.TopologicalSorter
	bootstrapGen      generate.BootstrapGenerator
	fixBuilder        report.SuggestedFixBuilder
	diagnosticEmitter report.DiagnosticEmitter
}

func NewAppAnalyzeRunner(
	appDetector detect.AppDetector,
	injectRegistry *registry.InjectorRegistry,
	provideRegistry *registry.ProviderRegistry,
	packageLoader loader.PackageLoader,
	packageTracker *registry.PackageTracker,
	bootstrapCtx context.Context,
	graphBuilder *graph.DependencyGraphBuilder,
	sorter *graph.TopologicalSorter,
	bootstrapGen generate.BootstrapGenerator,
	fixBuilder report.SuggestedFixBuilder,
	diagnosticEmitter report.DiagnosticEmitter,
) *AppAnalyzeRunner {
	return &AppAnalyzeRunner{
		appDetector:       appDetector,
		injectRegistry:    injectRegistry,
		provideRegistry:   provideRegistry,
		packageLoader:     packageLoader,
		packageTracker:    packageTracker,
		bootstrapCtx:      bootstrapCtx,
		graphBuilder:      graphBuilder,
		sorter:            sorter,
		bootstrapGen:      bootstrapGen,
		fixBuilder:        fixBuilder,
		diagnosticEmitter: diagnosticEmitter,
	}
}

func (r *AppAnalyzeRunner) Run(pass *analysis.Pass) (interface{}, error) {
	resultCh := make(chan runResult, 1)
	defer close(resultCh)

	go func() {
		resultCh <- r.run(pass)
	}()

	select {
	case <-r.bootstrapCtx.Done():
		return nil, r.bootstrapCtx.Err()
	case result := <-resultCh:
		return result.value, result.err
	}
}

type runResult struct {
	value interface{}
	err   error
}

func (r *AppAnalyzeRunner) run(pass *analysis.Pass) runResult {
	reporter := &passReporter{pass: pass}
	// Phase 1: Detect App annotations
	apps := r.appDetector.DetectAppAnnotations(pass)

	// Skip if no App annotation present
	if len(apps) == 0 {
		return runResult{}
	}

	// Phase 1.5: Deduplicate by file (same file → first only)
	allApps := apps
	apps = r.appDetector.DeduplicateAppsByFile(apps)

	// Report warnings for duplicates (apps that were removed)
	// Use map for O(n) lookup instead of O(n²) with slices.Contains
	dedupedSet := make(map[*detect.AppAnnotation]bool)
	for _, app := range apps {
		dedupedSet[app] = true
	}

	for _, app := range allApps {
		if !dedupedSet[app] {
			r.diagnosticEmitter.EmitDuplicateAppWarning(reporter, app.Pos)
		}
	}

	// Phase 2: Validate App annotations
	if err := r.appDetector.ValidateAppAnnotations(pass, apps); err != nil {
		// Report validation error
		if appErr, ok := err.(*detect.AppValidationError); ok {
			switch appErr.Type {
			case detect.NonMainReference:
				r.diagnosticEmitter.EmitNonMainAppError(reporter, appErr.Positions[0], appErr.FuncName)
			}
			// Skip bootstrap generation after validation error
			return runResult{}
		}
		// Unknown error type, skip bootstrap
		return runResult{}
	}

	// Phase 2: Wait for all packages to be scanned
	// Defensive programming: ensure pass.Files is not empty
	if len(pass.Files) == 0 {
		// This should not happen in normal analyzer execution, but handle it defensively
		return runResult{err: fmt.Errorf("no files in pass")}
	}

	allPkgPaths, err := r.packageLoader.LoadModulePackageNames(pass.Fset.File(pass.Files[0].Pos()).Name())
	if err != nil {
		// Emit warning diagnostic (non-critical)
		// The registry may be incomplete, but we proceed with what we have
		r.diagnosticEmitter.EmitPackageLoadWarning(reporter, apps[0].Pos, err.Error())
	}

	// Wait for all packages with timeout
	if len(allPkgPaths) > 0 {
		ctx := context.Background()

		if err := r.packageTracker.WaitForAllPackagesWithContext(ctx, allPkgPaths); err != nil {
			// Emit timeout warning diagnostic but continue
			// The registry may be incomplete, but we proceed with what we have
			r.diagnosticEmitter.EmitPackageWaitWarning(reporter, apps[0].Pos, err.Error())
		}
	}

	// Phase 3: Retrieve all providers and injectors from global registries
	providers := r.provideRegistry.GetAll()
	injectors := r.injectRegistry.GetAll()

	// Phase 4: Build dependency graph
	depGraph, err := r.graphBuilder.BuildGraph(pass, providers, injectors)
	if err != nil {
		// Report graph construction errors to the user
		r.diagnosticEmitter.EmitGraphBuildError(reporter, apps[0].Pos, err.Error())
		return runResult{}
	}

	// Execute topological sort
	sortedTypes, err := r.sorter.Sort(depGraph)
	if err != nil {
		if cycleErr, ok := err.(*graph.CycleError); ok {
			r.diagnosticEmitter.EmitCircularDependency(reporter, apps[0].Pos, cycleErr.Cycle)
			return runResult{}
		}
		// Unknown topological sort error - report for debugging
		r.diagnosticEmitter.EmitGraphBuildError(reporter, apps[0].Pos, err.Error())
		return runResult{}
	}

	// Phase 5: Generate bootstrap code
	// Check if bootstrap exists and is current
	existingBootstrap := r.bootstrapGen.DetectExistingBootstrap(pass)
	if existingBootstrap != nil {
		if r.bootstrapGen.CheckBootstrapCurrent(pass, existingBootstrap, depGraph) {
			// Bootstrap is up-to-date (idempotent)
			return runResult{}
		}
	}

	// Generate bootstrap
	bootstrap, err := r.bootstrapGen.GenerateBootstrap(pass, depGraph, sortedTypes)
	if err != nil {
		r.diagnosticEmitter.EmitGenerationError(reporter, apps[0].Pos, "bootstrap", err.Error())
		return runResult{}
	}

	// Find main function
	mainFunc := findMainFunction(pass)
	if mainFunc == nil {
		return runResult{}
	}

	// Build and emit fix
	var fix analysis.SuggestedFix

	if existingBootstrap != nil {
		fix = r.fixBuilder.BuildBootstrapReplacementFix(pass, existingBootstrap, bootstrap, mainFunc)
		r.diagnosticEmitter.EmitBootstrapUpdateFix(reporter, apps[0].Pos, fix)
	} else {
		fix = r.fixBuilder.BuildBootstrapFix(pass, apps[0], bootstrap, mainFunc)
		r.diagnosticEmitter.EmitBootstrapFix(reporter, apps[0].Pos, fix)
	}

	return runResult{}
}

// findMainFunction finds the main function declaration in the package.
func findMainFunction(pass *analysis.Pass) *ast.FuncDecl {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if fn.Name.Name == "main" {
					return fn
				}
			}
		}
	}
	return nil
}
