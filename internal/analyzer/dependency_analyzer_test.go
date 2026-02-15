package analyzer

import (
	"context"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// depAnalyzerTestEnv holds the DependencyAnalyzer and its registries for test assertions.
type depAnalyzerTestEnv struct {
	analyzer         *analysis.Analyzer
	providerRegistry *registry.ProviderRegistry
	injectorRegistry *registry.InjectorRegistry
	packageTracker   *registry.PackageTracker
}

// newDepAnalyzerTestEnv creates a DependencyAnalyzer with all real components.
// Returns a struct with the analyzer and registries needed for test assertions.
func newDepAnalyzerTestEnv() *depAnalyzerTestEnv {
	providerRegistry := registry.NewProviderRegistry()
	injectorRegistry := registry.NewInjectorRegistry()
	packageTracker := registry.NewPackageTracker()
	_, cancel := context.WithCancelCause(context.Background())

	injectDetector := detect.NewInjectDetector()
	fieldAnalyzer := detect.NewFieldAnalyzer()
	constructorAnalyzer := detect.NewConstructorAnalyzer()
	provideCallDetector := detect.NewProvideCallDetector()
	structDetector := detect.NewStructDetector(injectDetector)
	variableCallDetector := detect.NewVariableCallDetector()
	var optionExtractor detect.OptionExtractor
	constructorGenerator := generate.NewConstructorGenerator()
	suggestedFixBuilder := report.NewSuggestedFixBuilder()
	diagnosticEmitter := report.NewDiagnosticEmitter()
	variableRegistry := registry.NewVariableRegistry()

	runner := NewDependencyAnalyzeRunner(
		providerRegistry, injectorRegistry, packageTracker, cancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableRegistry,
	)
	analyzer := (*analysis.Analyzer)(NewDependencyAnalyzer(runner))

	return &depAnalyzerTestEnv{
		analyzer:         analyzer,
		providerRegistry: providerRegistry,
		injectorRegistry: injectorRegistry,
		packageTracker:   packageTracker,
	}
}

func TestDependencyAnalyzer(t *testing.T) {
	env := newDepAnalyzerTestEnv()
	analysistest.Run(t, "testdata/dependency/basic", env.analyzer, ".")

	// Verify providers were registered
	if len(env.providerRegistry.GetAll()) == 0 {
		t.Error("expected providers to be registered, got none")
	}

	// Verify injectors were registered
	if len(env.injectorRegistry.GetAll()) == 0 {
		t.Error("expected injectors to be registered, got none")
	}

	// Verify package was marked as scanned
	if !env.packageTracker.IsPackageScanned("example.com/dependency/basic") {
		t.Error("expected package to be marked as scanned")
	}
}

func TestDependencyAnalyzer_SuggestedFixes(t *testing.T) {
	env := newDepAnalyzerTestEnv()
	analysistest.RunWithSuggestedFixes(t, "testdata/constructorgen", env.analyzer, ".")
}

func TestDependencyAnalyzer_MissingProvideConstructor(t *testing.T) {
	env := newDepAnalyzerTestEnv()
	analysistest.Run(t, "testdata/dependency/missing_constructor", env.analyzer, ".")

	// Provider should not be registered when constructor is missing
	if n := len(env.providerRegistry.GetAll()); n != 0 {
		t.Errorf("expected no providers to be registered when constructor missing, got %d", n)
	}
}

func TestDependencyAnalyzer_CrossPackage(t *testing.T) {
	env := newDepAnalyzerTestEnv()
	analysistest.Run(t, "testdata/dependency/cross_package", env.analyzer, "./...")

	// Verify both packages registered their structs
	totalStructs := len(env.providerRegistry.GetAll()) + len(env.injectorRegistry.GetAll())
	if totalStructs < 2 {
		t.Errorf("expected at least 2 structs from cross-package test, got %d", totalStructs)
	}

	// Verify both packages were marked as scanned
	if !env.packageTracker.IsPackageScanned("example.com/dependency/cross_package/repo") {
		t.Error("expected repo package to be marked as scanned")
	}
	if !env.packageTracker.IsPackageScanned("example.com/dependency/cross_package/service") {
		t.Error("expected service package to be marked as scanned")
	}
}

func TestDependencyAnalyzer_InterfaceImplementation(t *testing.T) {
	env := newDepAnalyzerTestEnv()
	analysistest.Run(t, "testdata/dependency/abstrct", env.analyzer, "./...")

	// Verify Implements field is populated
	hasImplements := false
	for _, p := range env.providerRegistry.GetAll() {
		if len(p.Implements) > 0 {
			hasImplements = true
			break
		}
	}
	if !hasImplements {
		for _, i := range env.injectorRegistry.GetAll() {
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
