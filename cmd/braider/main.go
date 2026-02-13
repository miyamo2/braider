package main

import (
	"context"

	"github.com/miyamo2/braider/internal/analyzer"
	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/graph"
	"github.com/miyamo2/braider/internal/loader"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	// Step 1: Registries and shared context
	providerRegistry := registry.NewProviderRegistry()
	injectorRegistry := registry.NewInjectorRegistry()
	packageTracker := registry.NewPackageTracker()
	bootstrapCtx, bootstrapCancel := context.WithCancelCause(context.Background())

	// Step 2: Loaders
	packageLoader := loader.NewPackageLoader()

	// Step 3: Basic detectors (no dependencies)
	injectDetector := detect.NewInjectDetector()
	fieldAnalyzer := detect.NewFieldAnalyzer()
	constructorAnalyzer := detect.NewConstructorAnalyzer()
	appDetector := detect.NewAppDetector()

	// Step 4: Complex detectors (with dependencies)
	provideCallDetector := detect.NewProvideCallDetector()
	structDetector := detect.NewStructDetector(injectDetector)
	namerValidator := detect.NewNamerValidator(packageLoader)
	optionExtractor := detect.NewOptionExtractor(namerValidator)

	// Step 5: Graph components
	graphBuilder := graph.NewDependencyGraphBuilder()
	sorter := graph.NewTopologicalSorter()

	// Step 6: Generators and reporters
	constructorGenerator := generate.NewConstructorGenerator()
	bootstrapGenerator := generate.NewBootstrapGenerator()
	suggestedFixBuilder := report.NewSuggestedFixBuilder()
	diagnosticEmitter := report.NewDiagnosticEmitter()

	// Step 4.5: Variable components
	variableCallDetector := detect.NewVariableCallDetector()
	variableRegistry := registry.NewVariableRegistry()

	// Step 7: Instantiate analyzers
	dependencyAnalyzer := analyzer.DependencyAnalyzer(
		providerRegistry,
		injectorRegistry,
		packageTracker,
		bootstrapCancel,
		provideCallDetector,
		injectDetector,
		structDetector,
		fieldAnalyzer,
		constructorAnalyzer,
		optionExtractor,
		constructorGenerator,
		suggestedFixBuilder,
		diagnosticEmitter,
		variableCallDetector,
		variableRegistry,
	)

	appAnalyzer := analyzer.AppAnalyzer(
		appDetector,
		injectorRegistry,
		providerRegistry,
		packageLoader,
		packageTracker,
		bootstrapCtx,
		graphBuilder,
		sorter,
		bootstrapGenerator,
		suggestedFixBuilder,
		diagnosticEmitter,
		variableRegistry,
	)

	// Step 7: Pass to multichecker
	multichecker.Main(dependencyAnalyzer, appAnalyzer)
}
