package main

import (
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
	// Step 1: Registries
	providerRegistry := registry.NewProviderRegistry()
	injectorRegistry := registry.NewInjectorRegistry()
	packageTracker := registry.NewPackageTracker()

	packageLoader := loader.NewPackageLoader()

	// Step 2: Basic detectors (no dependencies)
	provideDetector := detect.NewProvideDetector()
	injectDetector := detect.NewInjectDetector()
	fieldAnalyzer := detect.NewFieldAnalyzer()
	constructorAnalyzer := detect.NewConstructorAnalyzer()
	appDetector := detect.NewAppDetector()

	// Step 3: Complex detectors (with dependencies)
	provideStructDetector := detect.NewProvideStructDetector(provideDetector)
	structDetector := detect.NewStructDetector(injectDetector)

	// Step 4: Graph components
	graphBuilder := graph.NewDependencyGraphBuilder()
	sorter := graph.NewTopologicalSorter()

	// Step 5: Generators and reporters
	constructorGenerator := generate.NewConstructorGenerator()
	bootstrapGenerator := generate.NewBootstrapGenerator()
	suggestedFixBuilder := report.NewSuggestedFixBuilder()
	diagnosticEmitter := report.NewDiagnosticEmitter()

	// Step 6: Instantiate analyzers
	dependencyAnalyzer := analyzer.DependencyAnalyzer(
		providerRegistry,
		injectorRegistry,
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
	)

	appAnalyzer := analyzer.AppAnalyzer(
		appDetector,
		injectorRegistry,
		providerRegistry,
		packageLoader,
		packageTracker,
		graphBuilder,
		sorter,
		bootstrapGenerator,
		suggestedFixBuilder,
		diagnosticEmitter,
	)

	// Step 7: Pass to multichecker
	multichecker.Main(dependencyAnalyzer, appAnalyzer)
}
