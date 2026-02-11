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

// setupDependencyAnalyzerDeps creates all required dependencies for DependencyAnalyzer tests.
func setupDependencyAnalyzerDeps() (
	*registry.ProviderRegistry,
	*registry.InjectorRegistry,
	*registry.PackageTracker,
	context.CancelCauseFunc,
	detect.ProvideCallDetector,
	detect.InjectDetector,
	detect.StructDetector,
	detect.FieldAnalyzer,
	detect.ConstructorAnalyzer,
	detect.OptionExtractor,
	generate.ConstructorGenerator,
	report.SuggestedFixBuilder,
	report.DiagnosticEmitter,
	detect.VariableCallDetector,
	*registry.VariableRegistry,
) {
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

	// Use nil option extractor for tests that don't need it
	var optionExtractor detect.OptionExtractor

	constructorGenerator := generate.NewConstructorGenerator()
	suggestedFixBuilder := report.NewSuggestedFixBuilder()
	diagnosticEmitter := report.NewDiagnosticEmitter()

	variableRegistry := registry.NewVariableRegistry()

	return providerRegistry, injectorRegistry, packageTracker, cancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableRegistry
}

// createDependencyAnalyzer creates a DependencyAnalyzer with the provided dependencies.
func createDependencyAnalyzer(
	providerRegistry *registry.ProviderRegistry,
	injectorRegistry *registry.InjectorRegistry,
	packageTracker *registry.PackageTracker,
	bootstrapCancel context.CancelCauseFunc,
	provideCallDetector detect.ProvideCallDetector,
	injectDetector detect.InjectDetector,
	structDetector detect.StructDetector,
	fieldAnalyzer detect.FieldAnalyzer,
	constructorAnalyzer detect.ConstructorAnalyzer,
	optionExtractor detect.OptionExtractor,
	constructorGenerator generate.ConstructorGenerator,
	suggestedFixBuilder report.SuggestedFixBuilder,
	diagnosticEmitter report.DiagnosticEmitter,
	variableCallDetector detect.VariableCallDetector,
	variableRegistry *registry.VariableRegistry,
) *analysis.Analyzer {
	return DependencyAnalyzer(
		providerRegistry, injectorRegistry, packageTracker, bootstrapCancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableRegistry,
	)
}

func TestDependencyAnalyzer(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, bootstrapCancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableRegistry := setupDependencyAnalyzerDeps()

	analyzer := createDependencyAnalyzer(
		providerRegistry, injectorRegistry, packageTracker, bootstrapCancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableRegistry,
	)

	analysistest.Run(t, "testdata/dependency/basic", analyzer, ".")

	// Verify providers were registered
	providers := providerRegistry.GetAll()
	if len(providers) == 0 {
		t.Error("expected providers to be registered, got none")
	}

	// Verify injectors were registered
	injectors := injectorRegistry.GetAll()
	if len(injectors) == 0 {
		t.Error("expected injectors to be registered, got none")
	}

	// Verify package was marked as scanned
	if !packageTracker.IsPackageScanned("example.com/dependency/basic") {
		t.Error("expected package to be marked as scanned")
	}
}

func TestDependencyAnalyzer_SuggestedFixes(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, bootstrapCancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableRegistry := setupDependencyAnalyzerDeps()

	analyzer := createDependencyAnalyzer(
		providerRegistry, injectorRegistry, packageTracker, bootstrapCancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableRegistry,
	)

	// Run with suggested fixes to verify code generation
	// Now using DependencyAnalyzer which includes constructor generation (Phase 1)
	analysistest.RunWithSuggestedFixes(t, "testdata/constructorgen", analyzer, ".")
}

func TestDependencyAnalyzer_MissingProvideConstructor(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, bootstrapCancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableRegistry := setupDependencyAnalyzerDeps()

	analyzer := createDependencyAnalyzer(
		providerRegistry, injectorRegistry, packageTracker, bootstrapCancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableRegistry,
	)

	analysistest.Run(t, "testdata/dependency/missing_constructor", analyzer, ".")

	// Provider should not be registered when constructor is missing
	providers := providerRegistry.GetAll()
	if len(providers) != 0 {
		t.Errorf("expected no providers to be registered when constructor missing, got %d", len(providers))
	}
}

func TestDependencyAnalyzer_CrossPackage(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, bootstrapCancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableRegistry := setupDependencyAnalyzerDeps()

	analyzer := createDependencyAnalyzer(
		providerRegistry, injectorRegistry, packageTracker, bootstrapCancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableRegistry,
	)

	// Analyze multiple packages
	analysistest.Run(t, "testdata/dependency/cross_package", analyzer, "./...")

	// Verify both packages registered their structs
	providers := providerRegistry.GetAll()
	injectors := injectorRegistry.GetAll()

	totalStructs := len(providers) + len(injectors)
	if totalStructs < 2 {
		t.Errorf("expected at least 2 structs from cross-package test, got %d", totalStructs)
	}

	// Verify both packages were marked as scanned
	if !packageTracker.IsPackageScanned("example.com/dependency/cross_package/repo") {
		t.Error("expected repo package to be marked as scanned")
	}
	if !packageTracker.IsPackageScanned("example.com/dependency/cross_package/service") {
		t.Error("expected service package to be marked as scanned")
	}
}

func TestDependencyAnalyzer_InterfaceImplementation(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, bootstrapCancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableRegistry := setupDependencyAnalyzerDeps()

	analyzer := createDependencyAnalyzer(
		providerRegistry, injectorRegistry, packageTracker, bootstrapCancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableRegistry,
	)

	analysistest.Run(t, "testdata/dependency/abstrct", analyzer, "./...")

	// Verify Implements field is populated
	providers := providerRegistry.GetAll()
	injectors := injectorRegistry.GetAll()

	hasImplements := false
	for _, p := range providers {
		if len(p.Implements) > 0 {
			hasImplements = true
			break
		}
	}
	for _, i := range injectors {
		if len(i.Implements) > 0 {
			hasImplements = true
			break
		}
	}

	if !hasImplements {
		t.Error("expected at least one struct to have Implements populated")
	}
}
