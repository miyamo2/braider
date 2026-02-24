package analyzer

import (
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"github.com/miyamo2/phasedchecker"
	"github.com/miyamo2/phasedchecker/checkertest"
	"golang.org/x/tools/go/analysis"
)

// depAnalyzerTestEnv holds the DependencyAnalyzer and its registries for test assertions.
type depAnalyzerTestEnv struct {
	analyzer   *analysis.Analyzer
	aggregator *Aggregator
}

// newDepAnalyzerTestEnv creates a DependencyAnalyzer with all real components.
// Returns a struct with the analyzer and aggregator needed for test assertions.
func newDepAnalyzerTestEnv(t *testing.T) *depAnalyzerTestEnv {
	t.Helper()
	providerRegistry := registry.NewProviderRegistry()
	injectorRegistry := registry.NewInjectorRegistry()
	variableRegistry := registry.NewVariableRegistry()

	markers, err := detect.ResolveMarkers()
	if err != nil {
		t.Fatal(err)
	}
	injectDetector := detect.NewInjectDetector(markers)
	fieldAnalyzer := detect.NewFieldAnalyzer()
	constructorAnalyzer := detect.NewConstructorAnalyzer()
	provideCallDetector := detect.NewProvideCallDetector(markers)
	structDetector := detect.NewStructDetector(injectDetector)
	variableCallDetector := detect.NewVariableCallDetector(markers)
	var optionExtractor detect.OptionExtractor
	constructorGenerator := generate.NewConstructorGenerator()
	suggestedFixBuilder := report.NewSuggestedFixBuilder()
	diagnosticEmitter := report.NewDiagnosticEmitter()

	agg := NewAggregator(providerRegistry, injectorRegistry, variableRegistry)

	runner := NewDependencyAnalyzeRunner(
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter, variableCallDetector,
	)
	analyzer := (*analysis.Analyzer)(NewDependencyAnalyzer(runner))

	return &depAnalyzerTestEnv{
		analyzer:   analyzer,
		aggregator: agg,
	}
}

func TestDependencyAnalyzer(t *testing.T) {
	env := newDepAnalyzerTestEnv(t)

	cfg := phasedchecker.Config{
		Pipeline: phasedchecker.Pipeline{
			Phases: []phasedchecker.Phase{
				{Name: "dependency", Analyzers: []*analysis.Analyzer{env.analyzer}, AfterPhase: env.aggregator.AfterDependencyPhase},
			},
		},
	}
	checkertest.Run(t, "testdata/dependency/basic", cfg, ".")

	// Verify providers were registered (via aggregator)
	if len(env.aggregator.ProviderRegistry.GetAll()) == 0 {
		t.Error("expected providers to be registered, got none")
	}

	// Verify injectors were registered (via aggregator)
	if len(env.aggregator.InjectorRegistry.GetAll()) == 0 {
		t.Error("expected injectors to be registered, got none")
	}
}

func TestDependencyAnalyzer_SuggestedFixes(t *testing.T) {
	env := newDepAnalyzerTestEnv(t)

	cfg := phasedchecker.Config{
		Pipeline: phasedchecker.Pipeline{
			Phases: []phasedchecker.Phase{
				{Name: "dependency", Analyzers: []*analysis.Analyzer{env.analyzer}, AfterPhase: env.aggregator.AfterDependencyPhase},
			},
		},
	}
	checkertest.RunWithSuggestedFixes(t, "testdata/constructorgen", cfg, ".")
}

func TestDependencyAnalyzer_MissingProvideConstructor(t *testing.T) {
	env := newDepAnalyzerTestEnv(t)

	cfg := phasedchecker.Config{
		Pipeline: phasedchecker.Pipeline{
			Phases: []phasedchecker.Phase{
				{Name: "dependency", Analyzers: []*analysis.Analyzer{env.analyzer}, AfterPhase: env.aggregator.AfterDependencyPhase},
			},
		},
	}
	checkertest.Run(t, "testdata/dependency/missing_constructor", cfg, ".")

	// Provider should not be registered when constructor is missing
	if n := len(env.aggregator.ProviderRegistry.GetAll()); n != 0 {
		t.Errorf("expected no providers to be registered when constructor missing, got %d", n)
	}
}

func TestDependencyAnalyzer_CrossPackage(t *testing.T) {
	env := newDepAnalyzerTestEnv(t)

	cfg := phasedchecker.Config{
		Pipeline: phasedchecker.Pipeline{
			Phases: []phasedchecker.Phase{
				{Name: "dependency", Analyzers: []*analysis.Analyzer{env.analyzer}, AfterPhase: env.aggregator.AfterDependencyPhase},
			},
		},
	}
	checkertest.Run(t, "testdata/dependency/cross_package", cfg, "./...")

	// Verify both packages registered their structs
	totalStructs := len(env.aggregator.ProviderRegistry.GetAll()) + len(env.aggregator.InjectorRegistry.GetAll())
	if totalStructs < 2 {
		t.Errorf("expected at least 2 structs from cross-package test, got %d", totalStructs)
	}
}

func TestDependencyAnalyzer_InterfaceImplementation(t *testing.T) {
	env := newDepAnalyzerTestEnv(t)

	cfg := phasedchecker.Config{
		Pipeline: phasedchecker.Pipeline{
			Phases: []phasedchecker.Phase{
				{Name: "dependency", Analyzers: []*analysis.Analyzer{env.analyzer}, AfterPhase: env.aggregator.AfterDependencyPhase},
			},
		},
	}
	checkertest.Run(t, "testdata/dependency/abstrct", cfg, "./...")

	// Verify Implements field is populated
	hasImplements := false
	for _, p := range env.aggregator.ProviderRegistry.GetAll() {
		if len(p.Implements) > 0 {
			hasImplements = true
			break
		}
	}
	if !hasImplements {
		for _, i := range env.aggregator.InjectorRegistry.GetAll() {
			if len(i.Implements) > 0 {
				hasImplements = true
				break
			}
		}
	}

	if !hasImplements {
		t.Error("expected at least one struct to have Implements populated")
	}
}
