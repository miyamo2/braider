package analyzer

import (
	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

// AppAnalyzer detects annotation.App(main) and generates bootstrap code.
func AppAnalyzer(
	appDetector detect.AppDetector,
	injectRegistry *registry.InjectorRegistry,
	provideRegistry *registry.ProviderRegistry,
	diagnosticEmitter report.DiagnosticEmitter,
) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "braider_app",
		Doc:      "detects App annotation and generates IIFE bootstrap code",
		Run:      NewAppAnalyzeRunner(appDetector, injectRegistry, provideRegistry, diagnosticEmitter).Run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

type AppAnalyzeRunner struct {
	appDetector       detect.AppDetector
	injectRegistry    *registry.InjectorRegistry
	provideRegistry   *registry.ProviderRegistry
	diagnosticEmitter report.DiagnosticEmitter
}

func NewAppAnalyzeRunner(
	appDetector detect.AppDetector,
	injectRegistry *registry.InjectorRegistry,
	provideRegistry *registry.ProviderRegistry,
	diagnosticEmitter report.DiagnosticEmitter,
) *AppAnalyzeRunner {
	return &AppAnalyzeRunner{
		appDetector:       appDetector,
		injectRegistry:    injectRegistry,
		provideRegistry:   provideRegistry,
		diagnosticEmitter: diagnosticEmitter,
	}
}

func (r *AppAnalyzeRunner) Run(pass *analysis.Pass) (interface{}, error) {
	reporter := &passReporter{pass: pass}

	// Phase 1: Detect and validate App annotation
	apps := r.appDetector.DetectAppAnnotations(pass)

	// Skip if no App annotation present
	if len(apps) == 0 {
		return nil, nil
	}

	// Validate App annotations
	if err := r.appDetector.ValidateAppAnnotations(pass, apps); err != nil {
		// Report validation error
		if appErr, ok := err.(*detect.AppValidationError); ok {
			switch appErr.Type {
			case detect.MultipleAppAnnotations:
				r.diagnosticEmitter.EmitMultipleAppError(reporter, appErr.Positions)
			case detect.NonMainReference:
				r.diagnosticEmitter.EmitNonMainAppError(reporter, appErr.Positions[0], appErr.FuncName)
			}
			// Skip bootstrap generation after validation error
			return nil, nil
		}
		// Unknown error type, skip bootstrap
		return nil, nil
	}

	// Phase 2: Wait for all packages to be scanned
	// TODO: This will be implemented when PackageLoader is available
	// For now, we skip the waiting logic and proceed directly to registry retrieval

	// Phase 3: Retrieve all providers and injectors from global registries
	providers := r.provideRegistry.GetAll()
	injectors := r.injectRegistry.GetAll()

	// TODO: Phase 4: Build dependency graph
	//   - Build dependency graph using DependencyGraph.BuildGraph()
	//   - Execute topological sort using TopologicalSort.Sort()
	//
	// TODO: Phase 5: Generate bootstrap code
	//   - Use BootstrapGenerator.GenerateBootstrap()
	//   - Check if current: BootstrapGenerator.CheckBootstrapCurrent()
	//   - Build SuggestedFix: SuggestedFixBuilder.BuildBootstrapFix()
	//   - Emit diagnostic: DiagnosticEmitter.EmitBootstrapFix()

	// Placeholder to prevent unused variable errors
	_ = providers
	_ = injectors

	return nil, nil
}
