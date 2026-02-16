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
	"github.com/miyamo2/braider/pkg/annotation/variable"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
)

var (
	bootstrapCtx, bootstrapCancel = context.WithCancelCause(context.Background())
	_                             = annotation.Variable[variable.Default](bootstrapCtx)
	_                             = annotation.Variable[variable.Default](bootstrapCancel)
	_                             = annotation.App(main)
)

func main() {
	multichecker.Main((*analysis.Analyzer)(dependency.dependencyAnalyzer), (*analysis.Analyzer)(dependency.appAnalyzer))
}

// braider:hash:5a2cbcb944ab64e7
var dependency = func() struct {
	appDetector             detect.AppDetector
	constructorAnalyzer     detect.ConstructorAnalyzer
	fieldAnalyzer           detect.FieldAnalyzer
	injectDetector          detect.InjectDetector
	provideCallDetector     detect.ProvideCallDetector
	structDetector          detect.StructDetector
	variableCallDetector    detect.VariableCallDetector
	codeFormatter           generate.CodeFormatter
	bootstrapGenerator      generate.BootstrapGenerator
	constructorGenerator    generate.ConstructorGenerator
	interfaceRegistry       *graph.InterfaceRegistry
	dependencyGraphBuilder  *graph.DependencyGraphBuilder
	topologicalSorter       *graph.TopologicalSorter
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
	markerInterfaces := detect.ResolveMarkers()
	appDetector := detect.NewAppDetector()
	constructorAnalyzer := detect.NewConstructorAnalyzer()
	fieldAnalyzer := detect.NewFieldAnalyzer()
	injectDetector := detect.NewInjectDetector(markerInterfaces)
	provideCallDetector := detect.NewProvideCallDetector(markerInterfaces)
	structDetector := detect.NewStructDetector(injectDetector)
	variableCallDetector := detect.NewVariableCallDetector(markerInterfaces)
	codeFormatter := generate.NewCodeFormatter()
	bootstrapGenerator := generate.NewBootstrapGenerator(codeFormatter)
	constructorGenerator := generate.NewConstructorGenerator()
	interfaceRegistry := graph.NewInterfaceRegistry()
	dependencyGraphBuilder := graph.NewDependencyGraphBuilder(interfaceRegistry)
	topologicalSorter := graph.NewTopologicalSorter()
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
	)
	appAnalyzer := analyzer.NewAppAnalyzer(appAnalyzeRunner)
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
	dependencyAnalyzer := analyzer.NewDependencyAnalyzer(dependencyAnalyzeRunner)
	return struct {
		appDetector             detect.AppDetector
		constructorAnalyzer     detect.ConstructorAnalyzer
		fieldAnalyzer           detect.FieldAnalyzer
		injectDetector          detect.InjectDetector
		provideCallDetector     detect.ProvideCallDetector
		structDetector          detect.StructDetector
		variableCallDetector    detect.VariableCallDetector
		codeFormatter           generate.CodeFormatter
		bootstrapGenerator      generate.BootstrapGenerator
		constructorGenerator    generate.ConstructorGenerator
		interfaceRegistry       *graph.InterfaceRegistry
		dependencyGraphBuilder  *graph.DependencyGraphBuilder
		topologicalSorter       *graph.TopologicalSorter
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
		appDetector:             appDetector,
		constructorAnalyzer:     constructorAnalyzer,
		fieldAnalyzer:           fieldAnalyzer,
		injectDetector:          injectDetector,
		provideCallDetector:     provideCallDetector,
		structDetector:          structDetector,
		variableCallDetector:    variableCallDetector,
		codeFormatter:           codeFormatter,
		bootstrapGenerator:      bootstrapGenerator,
		constructorGenerator:    constructorGenerator,
		interfaceRegistry:       interfaceRegistry,
		dependencyGraphBuilder:  dependencyGraphBuilder,
		topologicalSorter:       topologicalSorter,
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
