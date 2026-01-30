package analyzer

import (
	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

// DependencyAnalyzer detects annotation.Provide and annotation.Inject structs
// across all packages and registers them to global registries.
func DependencyAnalyzer(
	provideRegistry *registry.ProviderRegistry,
	injectRegistry *registry.InjectorRegistry,
	packageTracker *registry.PackageTracker,
	provideDetector detect.ProvideDetector,
	provideStructDetector detect.ProvideStructDetector,
	injectDetector detect.InjectDetector,
	structDetector detect.StructDetector,
	fieldAnalyzer detect.FieldAnalyzer,
	constructorAnalyzer detect.ConstructorAnalyzer,
	constructorGenerator generate.ConstructorGenerator,
	suggestedFixBuilder report.SuggestedFixBuilder,
	diagnosticEmitter report.DiagnosticEmitter,
) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "braider_dependency",
		Doc:  "detects Provide and Inject annotated structs and registers to global registry",
		Run: NewDependencyAnalyzeRunner(
			provideRegistry,
			injectRegistry,
			packageTracker,
			provideDetector,
			provideStructDetector,
			injectDetector,
			structDetector,
			fieldAnalyzer,
			constructorAnalyzer,
			constructorGenerator,
			suggestedFixBuilder,
			diagnosticEmitter,
		).Run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

type DependencyAnalyzeRunner struct {
	provideRegistry       *registry.ProviderRegistry
	injectRegistry        *registry.InjectorRegistry
	packageTracker        *registry.PackageTracker
	provideDetector       detect.ProvideDetector
	provideStructDetector detect.ProvideStructDetector
	injectDetector        detect.InjectDetector
	structDetector        detect.StructDetector
	fieldAnalyzer         detect.FieldAnalyzer
	constructorAnalyzer   detect.ConstructorAnalyzer
	constructorGenerator  generate.ConstructorGenerator
	suggestedFixBuilder   report.SuggestedFixBuilder
	diagnosticEmitter     report.DiagnosticEmitter
}

func NewDependencyAnalyzeRunner(
	provideRegistry *registry.ProviderRegistry,
	injectRegistry *registry.InjectorRegistry,
	packageTracker *registry.PackageTracker,
	provideDetector detect.ProvideDetector,
	provideStructDetector detect.ProvideStructDetector,
	injectDetector detect.InjectDetector,
	structDetector detect.StructDetector,
	fieldAnalyzer detect.FieldAnalyzer,
	constructorAnalyzer detect.ConstructorAnalyzer,
	constructorGenerator generate.ConstructorGenerator,
	suggestedFixBuilder report.SuggestedFixBuilder,
	diagnosticEmitter report.DiagnosticEmitter,
) *DependencyAnalyzeRunner {
	return &DependencyAnalyzeRunner{
		provideRegistry:       provideRegistry,
		injectRegistry:        injectRegistry,
		packageTracker:        packageTracker,
		provideDetector:       provideDetector,
		provideStructDetector: provideStructDetector,
		injectDetector:        injectDetector,
		structDetector:        structDetector,
		fieldAnalyzer:         fieldAnalyzer,
		constructorAnalyzer:   constructorAnalyzer,
		constructorGenerator:  constructorGenerator,
		suggestedFixBuilder:   suggestedFixBuilder,
		diagnosticEmitter:     diagnosticEmitter,
	}
}

func (r *DependencyAnalyzeRunner) Run(pass *analysis.Pass) (interface{}, error) {
	reporter := &passReporter{pass: pass}

	// Phase 1: Constructor Generation for Inject structs
	// Detect Inject structs that need constructors and generate them via SuggestedFix
	injectCandidates := r.structDetector.DetectCandidates(pass)
	for _, candidate := range injectCandidates {
		// Analyze fields (excluding annotation.Inject)
		fields := r.fieldAnalyzer.AnalyzeFields(pass, candidate.StructType, candidate.InjectField)

		// Skip if no injectable fields
		if !r.fieldAnalyzer.HasInjectableFields(fields) {
			continue
		}

		// Check if existing constructor is up-to-date
		if candidate.ExistingConstructor != nil {
			// Extract expected dependencies from struct fields
			var expectedDeps []string
			for _, field := range fields {
				if field.Type != nil {
					expectedDeps = append(expectedDeps, field.Type.String())
				}
			}

			// Extract actual dependencies from existing constructor
			actualDeps := r.constructorAnalyzer.ExtractDependencies(pass, candidate.ExistingConstructor)

			// If dependencies match, skip (constructor is up-to-date)
			if dependenciesMatch(expectedDeps, actualDeps) {
				continue
			}
		}

		// Generate constructor code
		constructor, err := r.constructorGenerator.GenerateConstructor(candidate, fields)
		if err != nil {
			r.diagnosticEmitter.EmitGenerationError(
				reporter,
				candidate.TypeSpec.Pos(),
				candidate.TypeSpec.Name.Name,
				err.Error(),
			)
			continue
		}

		// Build suggested fix
		fix := r.suggestedFixBuilder.BuildConstructorFix(pass, candidate, constructor)

		// Emit diagnostic with suggested fix
		if candidate.ExistingConstructor != nil {
			// Existing constructor needs to be updated
			r.diagnosticEmitter.EmitExistingConstructorFix(
				reporter,
				candidate.ExistingConstructor.Pos(),
				constructor.StructName,
				fix,
			)
		} else {
			// New constructor needs to be created
			r.diagnosticEmitter.EmitConstructorFix(
				reporter,
				candidate.TypeSpec.Pos(),
				constructor.StructName,
				fix,
			)
		}
	}

	// Phase 2: Detect and register Provide structs
	providers := r.provideStructDetector.DetectProviders(pass)
	for _, provider := range providers {
		// Task 3.2: Validate constructor existence
		if provider.ExistingConstructor == nil {
			r.diagnosticEmitter.EmitMissingConstructorError(
				reporter,
				provider.TypeSpec.Pos(),
				provider.TypeSpec.Name.Name,
			)
			continue
		}

		// Extract dependencies from constructor parameters
		dependencies := r.constructorAnalyzer.ExtractDependencies(pass, provider.ExistingConstructor)

		// Register to GlobalProviderRegistry
		r.provideRegistry.Register(
			&registry.ProviderInfo{
				TypeName:        pass.Pkg.Path() + "." + provider.TypeSpec.Name.Name,
				PackagePath:     pass.Pkg.Path(),
				LocalName:       provider.TypeSpec.Name.Name,
				ConstructorName: provider.ExistingConstructor.Name.Name,
				Dependencies:    dependencies,
				Implements:      provider.Implements,
				IsPending:       false, // Provide structs must have existing constructors
			},
		)
	}

	// Phase 3: Detect and register Inject structs with IsPending flag
	// Re-detect injectors to include state after constructor generation
	injectors := r.structDetector.DetectCandidates(pass)
	for _, injector := range injectors {
		var dependencies []string
		var isPending bool

		// Determine IsPending flag and extract dependencies
		if injector.ExistingConstructor != nil {
			// Constructor exists on disk
			dependencies = r.constructorAnalyzer.ExtractDependencies(pass, injector.ExistingConstructor)
			isPending = false
		} else {
			// Constructor generated in this pass (pending)
			fields := r.fieldAnalyzer.AnalyzeFields(pass, injector.StructType, injector.InjectField)
			for _, field := range fields {
				if field.Type != nil {
					dependencies = append(dependencies, field.Type.String())
				}
			}
			isPending = true
		}

		// Detect implemented interfaces
		var implements []string
		if injector.TypeSpec != nil {
			implements = r.provideStructDetector.DetectImplementedInterfaces(pass, injector.TypeSpec)
		}

		// Register to GlobalInjectorRegistry with IsPending flag
		r.injectRegistry.Register(
			&registry.InjectorInfo{
				TypeName:        pass.Pkg.Path() + "." + injector.TypeSpec.Name.Name,
				PackagePath:     pass.Pkg.Path(),
				LocalName:       injector.TypeSpec.Name.Name,
				ConstructorName: getConstructorName(injector),
				Dependencies:    dependencies,
				Implements:      implements,
				IsPending:       isPending,
			},
		)
	}

	// Phase 4: Mark package as scanned
	r.packageTracker.MarkPackageScanned(pass.Pkg.Path())

	return nil, nil
}

// getConstructorName returns the constructor name for an injector candidate.
// If ExistingConstructor exists, returns its name; otherwise returns expected name.
func getConstructorName(injector detect.ConstructorCandidate) string {
	if injector.ExistingConstructor != nil {
		return injector.ExistingConstructor.Name.Name
	}
	return "New" + injector.TypeSpec.Name.Name
}

// dependenciesMatch checks if two dependency lists are equivalent.
// Returns true if both lists contain the same dependencies (order-independent).
func dependenciesMatch(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}

	// Create a map for O(n) lookup
	depMap := make(map[string]bool)
	for _, dep := range expected {
		depMap[dep] = true
	}

	// Check if all actual dependencies are in expected
	for _, dep := range actual {
		if !depMap[dep] {
			return false
		}
	}

	return true
}

// passReporter adapts analysis.Pass to report.Reporter interface.
type passReporter struct {
	pass *analysis.Pass
}

func (r *passReporter) Report(d analysis.Diagnostic) {
	r.pass.Report(d)
}
