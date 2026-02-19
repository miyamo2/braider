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
	_                             = annotation.App[app.Default](main)
)

func main() {
	multichecker.Main((*analysis.Analyzer)(dependency.dependencyAnalyzer), (*analysis.Analyzer)(dependency.appAnalyzer))
}

// braider:hash:0665f306df89c872
var dependency = func() struct {
	markerInterfaces        *detect.MarkerInterfaces
	appDetector             detect.AppDetector
	appOptionExtractorImpl  detect.AppOptionExtractor
	constructorAnalyzer     detect.ConstructorAnalyzer
	fieldAnalyzer           detect.FieldAnalyzer
	injectDetector          detect.InjectDetector
	provideCallDetector     detect.ProvideCallDetector
	structDetector          detect.StructDetector
	variableCallDetector    detect.VariableCallDetector
	bootstrapGenerator      generate.BootstrapGenerator
	constructorGenerator    generate.ConstructorGenerator
	interfaceRegistry       *graph.InterfaceRegistry
	dependencyGraphBuilder  *graph.DependencyGraphBuilder
	topologicalSorter       *graph.TopologicalSorter
	containerResolverImpl   graph.ContainerResolver
	containerValidatorImpl  graph.ContainerValidator
	packageLoader           loader.PackageLoader
	namerValidatorImpl      detect.NamerValidator
	optionExtractorImpl     detect.OptionExtractor
	injectorRegistry        *registry.InjectorRegistry
	packageTracker          *registry.PackageTracker
	providerRegistry        *registry.ProviderRegistry
	variableRegistry        *registry.VariableRegistry
	diagnosticEmitter       report.DiagnosticEmitter
	suggestedFixBuilder     report.SuggestedFixBuilder
	appAnalyzeRunner        *analyzer.AppAnalyzeRunner
	appAnalyzer             *analyzer.AppAnalyzer
	dependencyAnalyzeRunner *analyzer.DependencyAnalyzeRunner
	dependencyAnalyzer      *analyzer.DependencyAnalyzer
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
	appAnalyzeRunner := analyzer.NewAppAnalyzeRunner(appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker, context, dependencyGraphBuilder, topologicalSorter, bootstrapGenerator, suggestedFixBuilder, diagnosticEmitter, variableRegistry, appOptionExtractorImpl, containerValidatorImpl, containerResolverImpl)
	appAnalyzer := analyzer.NewAppAnalyzer(appAnalyzeRunner)
	dependencyAnalyzeRunner := analyzer.NewDependencyAnalyzeRunner(providerRegistry, injectorRegistry, packageTracker, cancelCauseFunc, provideCallDetector, injectDetector, structDetector, fieldAnalyzer, constructorAnalyzer, optionExtractorImpl, constructorGenerator, suggestedFixBuilder, diagnosticEmitter, variableCallDetector, variableRegistry)
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(dependencyAnalyzeRunner)
	return struct {
		markerInterfaces        *detect.MarkerInterfaces
		appDetector             detect.AppDetector
		appOptionExtractorImpl  detect.AppOptionExtractor
		constructorAnalyzer     detect.ConstructorAnalyzer
		fieldAnalyzer           detect.FieldAnalyzer
		injectDetector          detect.InjectDetector
		provideCallDetector     detect.ProvideCallDetector
		structDetector          detect.StructDetector
		variableCallDetector    detect.VariableCallDetector
		bootstrapGenerator      generate.BootstrapGenerator
		constructorGenerator    generate.ConstructorGenerator
		interfaceRegistry       *graph.InterfaceRegistry
		dependencyGraphBuilder  *graph.DependencyGraphBuilder
		topologicalSorter       *graph.TopologicalSorter
		containerResolverImpl   graph.ContainerResolver
		containerValidatorImpl  graph.ContainerValidator
		packageLoader           loader.PackageLoader
		namerValidatorImpl      detect.NamerValidator
		optionExtractorImpl     detect.OptionExtractor
		injectorRegistry        *registry.InjectorRegistry
		packageTracker          *registry.PackageTracker
		providerRegistry        *registry.ProviderRegistry
		variableRegistry        *registry.VariableRegistry
		diagnosticEmitter       report.DiagnosticEmitter
		suggestedFixBuilder     report.SuggestedFixBuilder
		appAnalyzeRunner        *analyzer.AppAnalyzeRunner
		appAnalyzer             *analyzer.AppAnalyzer
		dependencyAnalyzeRunner *analyzer.DependencyAnalyzeRunner
		dependencyAnalyzer      *analyzer.DependencyAnalyzer
	}{
		markerInterfaces:        markerInterfaces,
		appDetector:             appDetector,
		appOptionExtractorImpl:  appOptionExtractorImpl,
		constructorAnalyzer:     constructorAnalyzer,
		fieldAnalyzer:           fieldAnalyzer,
		injectDetector:          injectDetector,
		provideCallDetector:     provideCallDetector,
		structDetector:          structDetector,
		variableCallDetector:    variableCallDetector,
		bootstrapGenerator:      bootstrapGenerator,
		constructorGenerator:    constructorGenerator,
		interfaceRegistry:       interfaceRegistry,
		dependencyGraphBuilder:  dependencyGraphBuilder,
		topologicalSorter:       topologicalSorter,
		containerResolverImpl:   containerResolverImpl,
		containerValidatorImpl:  containerValidatorImpl,
		packageLoader:           packageLoader,
		namerValidatorImpl:      namerValidatorImpl,
		optionExtractorImpl:     optionExtractorImpl,
		injectorRegistry:        injectorRegistry,
		packageTracker:          packageTracker,
		providerRegistry:        providerRegistry,
		variableRegistry:        variableRegistry,
		diagnosticEmitter:       diagnosticEmitter,
		suggestedFixBuilder:     suggestedFixBuilder,
		appAnalyzeRunner:        appAnalyzeRunner,
		appAnalyzer:             appAnalyzer,
		dependencyAnalyzeRunner: dependencyAnalyzeRunner,
		dependencyAnalyzer:      dependencyAnalyzer,
	}
}()
