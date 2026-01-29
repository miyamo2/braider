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
var DependencyAnalyzer = &analysis.Analyzer{
	Name:     "braider_dependency",
	Doc:      "detects Provide and Inject annotated structs and registers to global registry",
	Run:      runDependency,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

// Components used by the dependency analyzer
var (
	provideDetector       = detect.NewProvideDetector()
	provideStructDetector = detect.NewProvideStructDetector(provideDetector)
	injectDetector        = detect.NewInjectDetector()
	structDetector        = detect.NewStructDetector(injectDetector)
	fieldAnalyzer         = detect.NewFieldAnalyzer()
	constructorAnalyzer   = detect.NewConstructorAnalyzer()
	constructorGenerator  = generate.NewConstructorGenerator()
	suggestedFixBuilder   = report.NewSuggestedFixBuilder()
	diagnosticEmitter     = report.NewDiagnosticEmitter()
)

// passReporter adapts analysis.Pass to report.Reporter interface.
type passReporter struct {
	pass *analysis.Pass
}

func (r *passReporter) Report(d analysis.Diagnostic) {
	r.pass.Report(d)
}

func runDependency(pass *analysis.Pass) (interface{}, error) {
	reporter := &passReporter{pass: pass}

	// Phase 1: Constructor Generation for Inject structs
	// Detect Inject structs that need constructors and generate them via SuggestedFix
	injectCandidates := structDetector.DetectCandidates(pass)
	for _, candidate := range injectCandidates {
		// Analyze fields (excluding annotation.Inject)
		fields := fieldAnalyzer.AnalyzeFields(pass, candidate.StructType, candidate.InjectField)

		// Skip if no injectable fields
		if !fieldAnalyzer.HasInjectableFields(fields) {
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
			actualDeps := constructorAnalyzer.ExtractDependencies(pass, candidate.ExistingConstructor)

			// If dependencies match, skip (constructor is up-to-date)
			if dependenciesMatch(expectedDeps, actualDeps) {
				continue
			}
		}

		// Generate constructor code
		constructor, err := constructorGenerator.GenerateConstructor(candidate, fields)
		if err != nil {
			diagnosticEmitter.EmitGenerationError(
				reporter,
				candidate.TypeSpec.Pos(),
				candidate.TypeSpec.Name.Name,
				err.Error(),
			)
			continue
		}

		// Build suggested fix
		fix := suggestedFixBuilder.BuildConstructorFix(pass, candidate, constructor)

		// Emit diagnostic with suggested fix
		if candidate.ExistingConstructor != nil {
			// Existing constructor needs to be updated
			diagnosticEmitter.EmitExistingConstructorFix(
				reporter,
				candidate.ExistingConstructor.Pos(),
				constructor.StructName,
				fix,
			)
		} else {
			// New constructor needs to be created
			diagnosticEmitter.EmitConstructorFix(
				reporter,
				candidate.TypeSpec.Pos(),
				constructor.StructName,
				fix,
			)
		}
	}

	// Phase 2: Detect and register Provide structs
	providers := provideStructDetector.DetectProviders(pass)
	for _, provider := range providers {
		// Task 3.2: Validate constructor existence
		if provider.ExistingConstructor == nil {
			diagnosticEmitter.EmitMissingConstructorError(
				reporter,
				provider.TypeSpec.Pos(),
				provider.TypeSpec.Name.Name,
			)
			continue
		}

		// Extract dependencies from constructor parameters
		dependencies := constructorAnalyzer.ExtractDependencies(pass, provider.ExistingConstructor)

		// Register to GlobalProviderRegistry
		registry.GlobalProviderRegistry.Register(&registry.ProviderInfo{
			TypeName:        pass.Pkg.Path() + "." + provider.TypeSpec.Name.Name,
			PackagePath:     pass.Pkg.Path(),
			LocalName:       provider.TypeSpec.Name.Name,
			ConstructorName: provider.ExistingConstructor.Name.Name,
			Dependencies:    dependencies,
			Implements:      provider.Implements,
			IsPending:       false, // Provide structs must have existing constructors
		})
	}

	// Phase 3: Detect and register Inject structs with IsPending flag
	// Re-detect injectors to include state after constructor generation
	injectors := structDetector.DetectCandidates(pass)
	for _, injector := range injectors {
		var dependencies []string
		var isPending bool

		// Determine IsPending flag and extract dependencies
		if injector.ExistingConstructor != nil {
			// Constructor exists on disk
			dependencies = constructorAnalyzer.ExtractDependencies(pass, injector.ExistingConstructor)
			isPending = false
		} else {
			// Constructor generated in this pass (pending)
			fields := fieldAnalyzer.AnalyzeFields(pass, injector.StructType, injector.InjectField)
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
			implements = provideStructDetector.DetectImplementedInterfaces(pass, injector.TypeSpec)
		}

		// Register to GlobalInjectorRegistry with IsPending flag
		registry.GlobalInjectorRegistry.Register(&registry.InjectorInfo{
			TypeName:        pass.Pkg.Path() + "." + injector.TypeSpec.Name.Name,
			PackagePath:     pass.Pkg.Path(),
			LocalName:       injector.TypeSpec.Name.Name,
			ConstructorName: getConstructorName(injector),
			Dependencies:    dependencies,
			Implements:      implements,
			IsPending:       isPending,
		})
	}

	// Phase 4: Mark package as scanned
	registry.GlobalPackageTracker.MarkPackageScanned(pass.Pkg.Path())

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
