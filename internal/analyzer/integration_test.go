// Package analyzer provides integration tests for complete DependencyAnalyzer + AppAnalyzer pipeline.
//
// These tests verify end-to-end behavior using analysistest, running actual analyzers
// against testdata with real annotation types. DependencyAnalyzer is run with analysistest.Run
// (no golden file), and AppAnalyzer is run with analysistest.RunWithSuggestedFixes
// (golden file for main.go only).
package analyzer

import (
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

func TestIntegration_BasicSinglePackage(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/basic"
	analysistest.Run(t, testdir, depAnalyzer, "example.com/basic/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_CrossPackageImports(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/crosspackage"
	analysistest.Run(t, testdir, depAnalyzer, "crosspackage/repository")
	analysistest.Run(t, testdir, depAnalyzer, "crosspackage/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_MultiTypeCrossPackage(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/multitype"
	analysistest.Run(t, testdir, depAnalyzer, "example.com/multitype/repository")
	analysistest.Run(t, testdir, depAnalyzer, "example.com/multitype/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_InterfaceDependency(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/ifacedep"
	analysistest.Run(t, testdir, depAnalyzer, "example.com/ifacedep/domain")
	analysistest.Run(t, testdir, depAnalyzer, "example.com/ifacedep/repository")
	analysistest.Run(t, testdir, depAnalyzer, "example.com/ifacedep/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_InterfaceResolution(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/iface"
	analysistest.Run(t, testdir, depAnalyzer, "iface/domain")
	analysistest.Run(t, testdir, depAnalyzer, "iface/repository")
	analysistest.Run(t, testdir, depAnalyzer, "iface/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_CrossPackageInterface(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/crossiface"
	analysistest.Run(t, testdir, depAnalyzer, "crossiface/domain")
	analysistest.Run(t, testdir, depAnalyzer, "crossiface/repository")
	analysistest.Run(t, testdir, depAnalyzer, "crossiface/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_ModuleWideDiscovery(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/modulewide"
	analysistest.Run(t, testdir, depAnalyzer, "modulewide/repository")
	analysistest.Run(t, testdir, depAnalyzer, "modulewide/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_IdempotentImport(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/idempotent_import"
	analysistest.Run(t, testdir, depAnalyzer, "idempotent_import/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_CircularDependency(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/circular"
	analysistest.Run(t, testdir, depAnalyzer, "circular/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_AmbiguousInterface(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/ambiguous"
	analysistest.Run(t, testdir, depAnalyzer, "ambiguous/domain")
	analysistest.Run(t, testdir, depAnalyzer, "ambiguous/repository")
	analysistest.Run(t, testdir, depAnalyzer, "ambiguous/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_UnresolvedInterface(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/unresiface"
	analysistest.Run(t, testdir, depAnalyzer, "unresiface/writer")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_DependencyAlreadyUsed(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/depinuse"
	analysistest.Run(t, testdir, depAnalyzer, ".")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_IdempotentBehavior(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/idempotent"
	analysistest.Run(t, testdir, depAnalyzer, ".")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_DependencyBlankIdentifier(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/depblank"
	analysistest.Run(t, testdir, depAnalyzer, "example.com/depblank/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_BootstrapUpdate(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/outdated"
	analysistest.Run(t, testdir, depAnalyzer, "outdated/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_SameFileApp(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/samefileapp"
	analysistest.Run(t, testdir, depAnalyzer, "samefileapp/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// --- Group D: No Injectable/Provide → Simple Two-phase (3 tests) ---

func TestIntegration_EmptyGraph(t *testing.T) {
	_, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/emptygraph"
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_NonMainReference(t *testing.T) {
	_, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/nonmainapp"
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_NoAppAnnotation(t *testing.T) {
	_, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/noapp"
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_TypedInject tests Injectable[inject.Typed[I]] flow:
// struct registered with interface type -> bootstrap declares interface-typed field.
func TestIntegration_TypedInject(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/typed_inject"

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
	testdir := "testdata/bootstrapgen/named_inject"

	// Phase 1: DependencyAnalyzer scans service package (Namer types in same package)
	analysistest.Run(t, testdir, depAnalyzer, "named_inject/service")

	// Phase 2: AppAnalyzer generates bootstrap with named variables
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_ProvideTyped tests Provide[provide.Typed[I]] + Injectable[inject.Default] flow:
// provider with interface type -> injector depends on provider -> bootstrap wires both.
func TestIntegration_ProvideTyped(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/provide_typed"

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
	testdir := "testdata/bootstrapgen/without_constructor"

	// Phase 1: DependencyAnalyzer scans service (WithoutConstructor skips Phase 1 generation)
	analysistest.Run(t, testdir, depAnalyzer, "without_constructor/service")

	// Phase 2: AppAnalyzer generates bootstrap with manual constructor
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_ErrorCases tests Typed[I] constraint violation:
// concrete type does not implement interface -> fatal validation error -> AppAnalyzer skipped.
func TestIntegration_ErrorCases(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/error_cases"

	// Phase 1: DependencyAnalyzer detects constraint violation and emits diagnostic
	analysistest.Run(t, testdir, depAnalyzer, "error_cases/service")

	// Phase 2: AppAnalyzer skips due to cancelled validation context (no diagnostics expected)
	analysistest.Run(t, testdir, appAnalyzer, ".")
}
