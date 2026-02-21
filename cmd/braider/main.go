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
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
	"github.com/miyamo2/braider/pkg/annotation/variable"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
)

var (
	bootstrapCtx, bootstrapCancel = context.WithCancelCause(context.Background())
	_                             = annotation.Variable[variable.Default](bootstrapCtx)
	_                             = annotation.Variable[variable.Default](bootstrapCancel)
	_                             = annotation.App[app.Container[struct {
		dependencyAnalyzer *analysis.Analyzer `braider:"dependencyAnalyzer"`
		appAnalyzer        *analysis.Analyzer `braider:"appAnalyzer"`
	}]](main)
)

func main() {
	multichecker.Main(dependency.dependencyAnalyzer, dependency.appAnalyzer)
}

// braider:hash:0c9d34c6262111be
var dependency = func() struct {
	dependencyAnalyzer *analysis.Analyzer
	appAnalyzer        *analysis.Analyzer
} {
	cancelCauseFunc := bootstrapCancel
	context := bootstrapCtx
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
	packageTracker := registry.NewPackageTracker()
	providerRegistry := registry.NewProviderRegistry()
	variableRegistry := registry.NewVariableRegistry()
	diagnosticEmitter := report.NewDiagnosticEmitter()
	suggestedFixBuilder := report.NewSuggestedFixBuilder()
	appAnalyzeRunner := analyzer.NewAppAnalyzeRunner(
		appDetector,
		injectorRegistry,
		providerRegistry,
		packageLoader,
		packageTracker,
		context,
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
		providerRegistry,
		injectorRegistry,
		packageTracker,
		cancelCauseFunc,
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
		variableRegistry,
	)
	appAnalyzer := analyzer.NewAppAnalyzer(appAnalyzeRunner)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(dependencyAnalyzeRunner)
	return struct {
		dependencyAnalyzer *analysis.Analyzer
		appAnalyzer        *analysis.Analyzer
	}{
		dependencyAnalyzer: dependencyAnalyzer,
		appAnalyzer:        appAnalyzer,
	}
}()
