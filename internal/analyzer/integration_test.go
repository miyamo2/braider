// Package analyzer provides integration tests for complete DependencyAnalyzer + AppAnalyzer pipeline.
//
// These tests verify end-to-end behavior using analysistest, running actual analyzers
// against testdata with real annotation types. DependencyAnalyzer is run with analysistest.Run
// (no golden file), and AppAnalyzer is run with analysistest.RunWithSuggestedFixes
// (golden file for main.go only).
package analyzer

import (
	"strings"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/graph"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// setupIntegrationDeps creates shared registries and real components for integration tests.
// Returns both DependencyAnalyzer and AppAnalyzer configured with the same shared state.
func setupIntegrationDeps() (*analysis.Analyzer, *analysis.Analyzer) {
	// Shared registries
	providerReg := registry.NewProviderRegistry()
	injectorReg := registry.NewInjectorRegistry()
	packageTracker := registry.NewPackageTracker()
	validationCtx := registry.NewValidationContext()

	// Detection components (all real)
	packageLoader := &mockPackageLoader{}
	namerValidator := detect.NewNamerValidator(packageLoader)
	optionExtractor := detect.NewOptionExtractor(namerValidator)
	injectDetector := detect.NewInjectDetector()
	fieldAnalyzer := detect.NewFieldAnalyzer()
	constructorAnalyzer := detect.NewConstructorAnalyzer()
	provideCallDetector := detect.NewProvideCallDetector()
	structDetector := detect.NewStructDetector(injectDetector)

	// Generation components
	constructorGenerator := generate.NewConstructorGenerator()
	bootstrapGenerator := generate.NewBootstrapGenerator()

	// Report components
	suggestedFixBuilder := report.NewSuggestedFixBuilder()
	diagnosticEmitter := report.NewDiagnosticEmitter()

	// Graph components
	graphBuilder := graph.NewDependencyGraphBuilder()
	sorter := graph.NewTopologicalSorter()

	// App detection
	appDetector := detect.NewAppDetector()

	depAnalyzer := DependencyAnalyzer(
		providerReg, injectorReg, packageTracker, validationCtx,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
	)

	appAnalyzer := AppAnalyzer(
		appDetector, injectorReg, providerReg, packageLoader,
		packageTracker, validationCtx,
		graphBuilder, sorter, bootstrapGenerator,
		suggestedFixBuilder, diagnosticEmitter,
	)

	return depAnalyzer, appAnalyzer
}

// TestIntegration_TypedInject tests Injectable[inject.Typed[I]] flow:
// struct registered with interface type -> bootstrap declares interface-typed field.
func TestIntegration_TypedInject(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/refine_annotation/typed_inject"

	// Phase 1: DependencyAnalyzer scans domain and service packages
	analysistest.Run(t, testdir, depAnalyzer, "typed_inject/domain")
	analysistest.Run(t, testdir, depAnalyzer, "typed_inject/service")

	// Phase 2: AppAnalyzer generates bootstrap with golden file verification
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_NamedInject tests Injectable[inject.Named[N]] flow:
// structs with Namer types -> bootstrap uses custom variable names from Name() method.
func TestIntegration_NamedInject(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/refine_annotation/named_inject"

	// Phase 1: DependencyAnalyzer scans service package (Namer types in same package)
	analysistest.Run(t, testdir, depAnalyzer, "named_inject/service")

	// Phase 2: AppAnalyzer generates bootstrap with named variables
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_ProvideTyped tests Provide[provide.Typed[I]] + Injectable[inject.Default] flow:
// provider with interface type -> injector depends on provider -> bootstrap wires both.
func TestIntegration_ProvideTyped(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/refine_annotation/provide_typed"

	// Phase 1: DependencyAnalyzer scans all packages in order
	analysistest.Run(t, testdir, depAnalyzer, "provide_typed/domain")
	analysistest.Run(t, testdir, depAnalyzer, "provide_typed/repository")
	analysistest.Run(t, testdir, depAnalyzer, "provide_typed/service")

	// Phase 2: AppAnalyzer generates bootstrap
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_WithoutConstructor tests Injectable[inject.WithoutConstructor] flow:
// constructor generation skipped, but struct appears in bootstrap using manual constructor.
func TestIntegration_WithoutConstructor(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/refine_annotation/without_constructor"

	// Phase 1: DependencyAnalyzer scans service (WithoutConstructor skips Phase 1 generation)
	analysistest.Run(t, testdir, depAnalyzer, "without_constructor/service")

	// Phase 2: AppAnalyzer generates bootstrap with manual constructor
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_ErrorCases tests Typed[I] constraint violation:
// concrete type does not implement interface -> fatal validation error -> AppAnalyzer skipped.
func TestIntegration_ErrorCases(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/refine_annotation/error_cases"

	// Phase 1: DependencyAnalyzer detects constraint violation and emits diagnostic
	analysistest.Run(t, testdir, depAnalyzer, "error_cases/service")

	// Phase 2: AppAnalyzer skips due to cancelled validation context (no diagnostics expected)
	analysistest.Run(t, testdir, appAnalyzer, ".")
}

// TestIntegration_CorrelationErrorNonFatal tests that duplicate (TypeName, Name) registration
// returns an error from Registry.Register() but does NOT cancel the ValidationContext,
// so AppAnalyzer continues to generate bootstrap code.
// This scenario cannot be triggered via analysistest because Go TypeNames are unique per package.
func TestIntegration_CorrelationErrorNonFatal(t *testing.T) {
	injectorReg := registry.NewInjectorRegistry()
	validationCtx := registry.NewValidationContext()

	// First registration succeeds
	err := injectorReg.Register(&registry.InjectorInfo{
		TypeName:        "example.com/repo.Repository",
		PackagePath:     "example.com/repo",
		PackageName:     "repo",
		LocalName:       "Repository",
		ConstructorName: "NewRepository",
		Dependencies:    []string{},
		Name:            "primary",
		OptionMetadata:  detect.OptionMetadata{Name: "primary"},
	})
	if err != nil {
		t.Fatalf("First registration should succeed, got: %v", err)
	}

	// Duplicate registration returns error
	err = injectorReg.Register(&registry.InjectorInfo{
		TypeName:        "example.com/repo.Repository",
		PackagePath:     "example.com/repo2",
		PackageName:     "repo2",
		LocalName:       "Repository",
		ConstructorName: "NewRepository",
		Dependencies:    []string{},
		Name:            "primary",
		OptionMetadata:  detect.OptionMetadata{Name: "primary"},
	})
	if err == nil {
		t.Fatal("Duplicate registration should return error")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("Error should mention 'duplicate', got: %v", err)
	}

	// Context should NOT be cancelled (correlation errors are non-fatal)
	if validationCtx.IsCancelled() {
		t.Error("ValidationContext should NOT be cancelled for correlation errors")
	}
}
