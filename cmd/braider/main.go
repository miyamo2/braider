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
)

var (
	bootstrapCtx, bootstrapCancel = context.WithCancelCause(context.Background())
	_                             = annotation.Variable[variable.Default](bootstrapCtx)
	_                             = annotation.Variable[variable.Default](bootstrapCancel)
	_                             = annotation.App(main)
)

func main() {
	_ = dependency

	// multichecker.Main(dependencyAnalyzer, appAnalyzer)
}

// braider:hash:a1d754d90158c4d5
var dependency = func() struct {
	appDetector             detect.AppDetector
	constructorAnalyzer     detect.ConstructorAnalyzer
	fieldAnalyzer           detect.FieldAnalyzer
	injectDetector          detect.InjectDetector
	provideCallDetector     detect.ProvideCallDetector
	structDetector          detect.StructDetector
	variableCallDetector    detect.VariableCallDetector
	codeFormatter           generate.CodeFormatter
	bootstrapGenerator      *generate.bootstrapGenerator
	constructorGenerator    generate.ConstructorGenerator
	dependencyGraphBuilder  *graph.DependencyGraphBuilder
	topologicalSorter       *graph.TopologicalSorter
	namerValidatorImpl      detect.NamerValidator
	optionExtractorImpl     detect.OptionExtractor
	diagnosticEmitter       report.DiagnosticEmitter
	suggestedFixBuilder     report.SuggestedFixBuilder
	appAnalyzeRunner        *analyzer.AppAnalyzeRunner
	dependencyAnalyzeRunner *analyzer.DependencyAnalyzeRunner
} {
	cancelCauseFunc := bootstrapCancel
	context := bootstrapCtx
	appDetector := detect.NewAppDetector()
	constructorAnalyzer := detect.NewconstructorAnalyzer()
	fieldAnalyzer := detect.NewfieldAnalyzer()
	injectDetector := detect.NewinjectDetector()
	provideCallDetector := detect.NewprovideCallDetector()
	structDetector := detect.NewstructDetector(injectDetector)
	variableCallDetector := detect.NewvariableCallDetector()
	codeFormatter := generate.NewcodeFormatter()
	bootstrapGenerator := generate.NewbootstrapGenerator(codeFormatter)
	constructorGenerator := generate.NewconstructorGenerator()
	interfaceRegistry := graph.NewInterfaceRegistry()
	dependencyGraphBuilder := graph.NewDependencyGraphBuilder(interfaceRegistry)
	topologicalSorter := graph.NewTopologicalSorter()
	packageLoader := loader.NewPackageLoader()
	namerValidatorImpl := detect.NewnamerValidatorImpl(packageLoader)
	optionExtractorImpl := detect.NewoptionExtractorImpl(namerValidatorImpl)
	injectorRegistry := registry.NewInjectorRegistry()
	packageTracker := registry.NewPackageTracker()
	providerRegistry := registry.NewProviderRegistry()
	variableRegistry := registry.NewVariableRegistry()
	diagnosticEmitter := report.NewdiagnosticEmitter()
	suggestedFixBuilder := report.NewsuggestedFixBuilder()
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
		bootstrapGenerator      *generate.bootstrapGenerator
		constructorGenerator    generate.ConstructorGenerator
		dependencyGraphBuilder  *graph.DependencyGraphBuilder
		topologicalSorter       *graph.TopologicalSorter
		namerValidatorImpl      detect.NamerValidator
		optionExtractorImpl     detect.OptionExtractor
		diagnosticEmitter       report.DiagnosticEmitter
		suggestedFixBuilder     report.SuggestedFixBuilder
		appAnalyzeRunner        *analyzer.AppAnalyzeRunner
		dependencyAnalyzeRunner *analyzer.DependencyAnalyzeRunner
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
		dependencyGraphBuilder:  dependencyGraphBuilder,
		topologicalSorter:       topologicalSorter,
		namerValidatorImpl:      namerValidatorImpl,
		optionExtractorImpl:     optionExtractorImpl,
		diagnosticEmitter:       diagnosticEmitter,
		suggestedFixBuilder:     suggestedFixBuilder,
		appAnalyzeRunner:        appAnalyzeRunner,
		dependencyAnalyzeRunner: dependencyAnalyzeRunner,
	}
}()
