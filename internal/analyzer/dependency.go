package analyzer

import (
	"github.com/miyamo2/braider/internal/detect"
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

	// Phase 1: Detect and register Provide structs
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
		})
	}

	// Phase 2: Detect and register Inject structs
	injectors := structDetector.DetectCandidates(pass)
	for _, injector := range injectors {
		var dependencies []string

		// Extract dependencies based on constructor existence
		if injector.ExistingConstructor != nil {
			// Extract from existing constructor parameters
			dependencies = constructorAnalyzer.ExtractDependencies(pass, injector.ExistingConstructor)
		} else {
			// Derive from struct fields (excluding annotation.Inject)
			fields := fieldAnalyzer.AnalyzeFields(pass, injector.StructType, injector.InjectField)
			for _, field := range fields {
				if field.Type != nil {
					dependencies = append(dependencies, field.Type.String())
				}
			}
		}

		// Detect implemented interfaces
		var implements []string
		if injector.TypeSpec != nil {
			implements = provideStructDetector.DetectImplementedInterfaces(pass, injector.TypeSpec)
		}

		// Register to GlobalInjectorRegistry
		registry.GlobalInjectorRegistry.Register(&registry.InjectorInfo{
			TypeName:        pass.Pkg.Path() + "." + injector.TypeSpec.Name.Name,
			PackagePath:     pass.Pkg.Path(),
			LocalName:       injector.TypeSpec.Name.Name,
			ConstructorName: getConstructorName(injector),
			Dependencies:    dependencies,
			Implements:      implements,
		})
	}

	// Phase 3: Mark package as scanned
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
