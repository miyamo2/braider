package main

import (
	"fmt"
	"os"

	"github.com/miyamo2/phasedchecker"
	"golang.org/x/tools/go/analysis"

	"github.com/miyamo2/braider/internal/analyzer"
	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/graph"
	"github.com/miyamo2/braider/internal/loader"
	"github.com/miyamo2/braider/internal/lsp"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[struct {
	dependencyAnalyzer *analysis.Analyzer `braider:"dependencyAnalyzer"`
	appAnalyzer        *analysis.Analyzer `braider:"appAnalyzer"`
	aggregator         *analyzer.Aggregator
}]](main)

const lspHelp = `lsp subcommand:

  braider lsp

  Start an LSP server (JSON-RPC 2.0 over stdio) for editor integration.

  Capabilities:
    textDocument/completion   Surface exported type candidates for DI annotation type arguments.
    textDocument/hover        Show which provider/injector/variable binding wins for the type under the cursor.
    textDocument/codeAction   Offer "Register with annotation.Provide" quick-fix on exported constructors.

  The server performs a best-effort go/packages load per request, using open-file
  overlays for unsaved edits, and runs a background workspace scan on startup to
  populate in-memory provider/injector/variable caches.
`

func main() {
	if len(os.Args) > 1 && os.Args[1] == "lsp" {
		if len(os.Args) > 2 && os.Args[2] == "--help" {
			fmt.Print(lspHelp)
			return
		}
		server := lsp.NewServer(os.Stdin, os.Stdout)
		if err := server.Run(); err != nil {
			os.Exit(1)
		}
		return
	}

	if len(os.Args) == 3 && os.Args[1] == "help" && os.Args[2] == "lsp" {
		fmt.Print(lspHelp)
		return
	}

	// phasedchecker.Main calls os.Exit after printing help, so we print the subcommands
	// section first when a general help invocation is detected.
	isGeneralHelp := len(os.Args) == 1 || (len(os.Args) == 2 && os.Args[1] == "help")
	if isGeneralHelp {
		fmt.Print("Subcommands:\n\n  lsp\n    Start an LSP server (JSON-RPC 2.0 over stdio) for editor integration.\n    Run 'braider help lsp' for details.\n\n")
	}

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
					{Category: report.CategoryDependencyRegistration, Severity: phasedchecker.SeverityCritical},
				},
				DefaultSeverity: phasedchecker.SeverityWarn,
			},
		},
	)
}

// braider:hash:f1c474d4ddb64d45
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
	duplicateRegistry := registry.NewDuplicateRegistry()
	injectorRegistry := registry.NewInjectorRegistry()
	providerRegistry := registry.NewProviderRegistry()
	variableRegistry := registry.NewVariableRegistry()
	aggregator := analyzer.NewAggregator(providerRegistry, injectorRegistry, variableRegistry, duplicateRegistry)
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
		duplicateRegistry,
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
