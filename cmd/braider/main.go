package main

import (
	"github.com/miyamo2/braider/internal/analyzer"
	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/graph"
	"github.com/miyamo2/braider/internal/loader"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
	"github.com/miyamo2/phasedchecker"
	"golang.org/x/tools/go/analysis"
)

var _ = annotation.App[app.Container[struct {
	dependencyAnalyzer *analysis.Analyzer `braider:"dependencyAnalyzer"`
	appAnalyzer        *analysis.Analyzer `braider:"appAnalyzer"`
	aggregator         *analyzer.Aggregator
}]](main)

func main() {
	phasedchecker.Main(
		phasedchecker.Config{
			Pipeline: phasedchecker.Pipeline{
				Phases: []phasedchecker.Phase{
					{
						Name:       "dependency",
						Analyzers:  []*analysis.Analyzer{dependency.dependencyAnalyzer},
						AfterPhase: dependency.aggregator.AfterDependencyPhase,
					},
					{
						Name:      "app",
						Analyzers: []*analysis.Analyzer{dependency.appAnalyzer},
					},
				},
			},
			DiagnosticPolicy: phasedchecker.DiagnosticPolicy{
				Rules: []phasedchecker.CategoryRule{
					{Category: report.CategoryOptionValidation, Severity: phasedchecker.SeverityCritical},
					{Category: report.CategoryExpressionValidation, Severity: phasedchecker.SeverityCritical},
				},
			},
		},
	)
}

// braider:hash:ebe186a2490ad887
var dependency = func() struct {
	dependencyAnalyzer *analysis.Analyzer
	appAnalyzer        *analysis.Analyzer
	aggregator         *analyzer.Aggregator
} {
	markerInterfaces := detect.MustResolveMarkers()
	appDetector := detect.NewAppDetector(markerInterfaces)
	appOptionExtractorImpl := detect.NewAppOptionExtractorImpl(markerInterfaces)
	constructorAnalyzer := detect.NewConstructorAnalyzer()
	fieldAnalyzer := detect.NewFieldAnalyzer()
	injectDetector := detect.NewInjectDetector(markerInterfaces)
	provideCallDetector := detect.NewProvideCallDetector(markerInterfaces)
	structDetector := detect.NewStructDetector(injectDetector)
	variableCallDetector := detect.NewVariableCallDetector(markerInterfaces)
	bootstrapGenerator := generate.NewBootstrapGenerator()
	constructorGenerator := generate.NewConstructorGenerator()
	interfaceRegistry := graph.NewInterfaceRegistry()
	dependencyGraphBuilder := graph.NewDependencyGraphBuilder(interfaceRegistry)
	topologicalSorter := graph.NewTopologicalSorter()
	containerResolverImpl := graph.NewContainerResolverImpl(interfaceRegistry)
	containerValidatorImpl := graph.NewContainerValidatorImpl(interfaceRegistry)
	packageLoader := loader.NewPackageLoader()
	namerValidatorImpl := detect.NewNamerValidatorImpl(packageLoader)
	optionExtractorImpl := detect.NewOptionExtractorImpl(markerInterfaces, namerValidatorImpl)
	injectorRegistry := registry.NewInjectorRegistry()
	providerRegistry := registry.NewProviderRegistry()
	variableRegistry := registry.NewVariableRegistry()
	aggregator := analyzer.NewAggregator(providerRegistry, injectorRegistry, variableRegistry)
	diagnosticEmitter := report.NewDiagnosticEmitter()
	suggestedFixBuilder := report.NewSuggestedFixBuilder()
	appAnalyzeRunner := analyzer.NewAppAnalyzeRunner(
		appDetector,
		injectorRegistry,
		providerRegistry,
		dependencyGraphBuilder,
		topologicalSorter,
		bootstrapGenerator,
		suggestedFixBuilder,
		diagnosticEmitter,
		variableRegistry,
		appOptionExtractorImpl,
		containerValidatorImpl,
		containerResolverImpl,
	)
	dependencyAnalyzeRunner := analyzer.NewDependencyAnalyzeRunner(
		provideCallDetector,
		injectDetector,
		structDetector,
		fieldAnalyzer,
		constructorAnalyzer,
		optionExtractorImpl,
		constructorGenerator,
		suggestedFixBuilder,
		diagnosticEmitter,
		variableCallDetector,
	)
	appAnalyzer := analyzer.NewAppAnalyzer(appAnalyzeRunner)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(dependencyAnalyzeRunner)
	return struct {
		dependencyAnalyzer *analysis.Analyzer
		appAnalyzer        *analysis.Analyzer
		aggregator         *analyzer.Aggregator
	}{
		dependencyAnalyzer: dependencyAnalyzer,
		appAnalyzer:        appAnalyzer,
		aggregator:         aggregator,
	}
}()
