// Package analyzer provides integration tests for complete DependencyAnalyzer + AppAnalyzer pipeline.
//
// These tests verify end-to-end behavior using analysistest, running actual analyzers
// against testdata with real annotation types. DependencyAnalyzer is run with analysistest.Run
// (no golden file), and AppAnalyzer is run with analysistest.RunWithSuggestedFixes
// (golden file for main.go only).
package analyzer

import (
	"context"
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
	depAnalyzer, appAnalyzer, _, _, _ := buildIntegrationDeps()
	return depAnalyzer, appAnalyzer
}

// buildIntegrationDeps creates all shared components and returns analyzers plus raw registries.
func buildIntegrationDeps() (
	*analysis.Analyzer, *analysis.Analyzer,
	*registry.InjectorRegistry, *registry.ProviderRegistry,
	*registry.VariableRegistry,
) {
	// Shared registries
	providerReg := registry.NewProviderRegistry()
	injectorReg := registry.NewInjectorRegistry()
	packageTracker := registry.NewPackageTracker()
	ctx, bootstrapCancel := context.WithCancelCause(context.Background())

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

	// Variable components
	variableCallDetector := detect.NewVariableCallDetector()
	variableReg := registry.NewVariableRegistry()

	depAnalyzer := DependencyAnalyzer(
		providerReg, injectorReg, packageTracker, bootstrapCancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableReg,
	)

	appAnalyzer := AppAnalyzer(
		appDetector, injectorReg, providerReg, packageLoader,
		packageTracker, ctx,
		graphBuilder, sorter, bootstrapGenerator,
		suggestedFixBuilder, diagnosticEmitter,
		variableReg,
	)

	return depAnalyzer, appAnalyzer, injectorReg, providerReg, variableReg
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

func TestIntegration_PackageNameCollision(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/pkgcollision"
	analysistest.Run(t, testdir, depAnalyzer, "pkgcollision/v1user")
	analysistest.Run(t, testdir, depAnalyzer, "pkgcollision/v2user")
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

func TestIntegration_AmbiguousInterfaceProvide(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/ambiguousiface"
	analysistest.Run(t, testdir, depAnalyzer, "example.com/ambiguousiface/domain")
	analysistest.Run(t, testdir, depAnalyzer, "example.com/ambiguousiface/repository")
	analysistest.Run(t, testdir, depAnalyzer, "example.com/ambiguousiface/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_UnresolvedInterface(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/unresiface"
	analysistest.Run(t, testdir, depAnalyzer, "unresiface/writer")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_UnresolvedInterfaceDependency(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/unresolvedif"
	analysistest.Run(t, testdir, depAnalyzer, "example.com/unresolvedif/writer")
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

func TestIntegration_SimpleApp(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/simpleapp"
	analysistest.Run(t, testdir, depAnalyzer, "simpleapp/service")
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

// TestIntegration_ErrorNonLiteralNamer tests Named[N] with non-literal Name() return:
// Namer returns variable instead of string literal -> fatal validation error -> AppAnalyzer skipped.
func TestIntegration_ErrorNonLiteralNamer(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/error_nonliteral"

	// Phase 1: DependencyAnalyzer detects non-literal Name() return and emits diagnostic
	analysistest.Run(t, testdir, depAnalyzer, "error_nonliteral/service")

	// Phase 2: AppAnalyzer skips due to cancelled validation context (no diagnostics expected)
	analysistest.Run(t, testdir, appAnalyzer, ".")
}

// TestIntegration_ProvideNamed tests Provide[provide.Named[N]] flow:
// provider with named registration -> bootstrap uses custom variable name from Name() method.
func TestIntegration_ProvideNamed(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/provide_named"

	// Phase 1: DependencyAnalyzer scans repository package with Named provider
	analysistest.Run(t, testdir, depAnalyzer, "provide_named/repository")

	// Phase 2: AppAnalyzer generates bootstrap with named variable
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_ErrorProvideTyped tests Provide[provide.Typed[I]] constraint violation:
// concrete type does not implement interface -> fatal validation error -> AppAnalyzer skipped.
func TestIntegration_ErrorProvideTyped(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/error_provide_typed"

	// Phase 1: DependencyAnalyzer scans packages; repository triggers constraint violation
	analysistest.Run(t, testdir, depAnalyzer, "error_provide_typed/domain")
	analysistest.Run(t, testdir, depAnalyzer, "error_provide_typed/repository")

	// Phase 2: AppAnalyzer skips due to cancelled validation context (no diagnostics expected)
	analysistest.Run(t, testdir, appAnalyzer, ".")
}

func TestIntegration_ErrorUnresolvedParam(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/unresolvedparam"
	analysistest.Run(t, testdir, depAnalyzer, "example.com/unresolvedparam/repository")
	analysistest.Run(t, testdir, depAnalyzer, "example.com/unresolvedparam/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestIntegration_ErrorUnresolvedParamDetail(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/unresparam"
	analysistest.Run(t, testdir, depAnalyzer, "unresparam/repository")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_ErrorDuplicateName tests duplicate (TypeName, Name) detection:
// Same named dependency registered twice -> non-fatal warning emitted.
// Uses programmatic registry access since analysistest cannot naturally test duplicate registration
// (each analysistest.Run creates a fresh analysis pass for the same source).
func TestIntegration_ErrorDuplicateName(t *testing.T) {
	depAnalyzer, _, injectorReg, _, _ := buildIntegrationDeps()
	testdir := "testdata/bootstrapgen/error_duplicate_name"

	// First scan: registers Named service via analysistest (succeeds without duplicate)
	analysistest.Run(t, testdir, depAnalyzer, "error_duplicate_name/service")

	// Verify first registration succeeded
	allInjectors := injectorReg.GetAll()
	if len(allInjectors) == 0 {
		t.Fatal("expected at least one injector registered after first scan")
	}

	// Find the registered injector
	var registeredInfo *registry.InjectorInfo
	for _, info := range allInjectors {
		if info.Name == "primary" {
			registeredInfo = info
			break
		}
	}
	if registeredInfo == nil {
		t.Fatal("expected injector with name \"primary\" to be registered")
	}

	// Programmatic duplicate: re-register with same (TypeName, Name) to verify duplicate detection
	err := injectorReg.Register(&registry.InjectorInfo{
		TypeName:        registeredInfo.TypeName,
		PackagePath:     "another/package",
		PackageName:     "another",
		LocalName:       registeredInfo.LocalName,
		ConstructorName: registeredInfo.ConstructorName,
		Name:            "primary",
	})
	if err == nil {
		t.Fatal("expected duplicate registration error, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate named dependency") {
		t.Fatalf("expected error containing \"duplicate named dependency\", got: %s", err.Error())
	}
}

// --- Group F: Variable Annotation Integration Tests ---

// TestIntegration_VariableBasic tests Variable[variable.Default] flow:
// pre-existing value registered under its declared type -> bootstrap assigns as local variable.
func TestIntegration_VariableBasic(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/variable_basic"
	analysistest.Run(t, testdir, depAnalyzer, "variable_basic/config")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_VariableTyped tests Variable[variable.Typed[I]] flow:
// pre-existing value registered as interface type -> bootstrap assigns as local variable.
func TestIntegration_VariableTyped(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/variable_typed"
	analysistest.Run(t, testdir, depAnalyzer, "variable_typed/domain")
	analysistest.Run(t, testdir, depAnalyzer, "variable_typed/config")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_VariableNamed tests Variable[variable.Named[N]] flow:
// pre-existing value registered with custom name -> bootstrap uses named variable.
func TestIntegration_VariableNamed(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/variable_named"
	analysistest.Run(t, testdir, depAnalyzer, "variable_named/config")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_VariableMixed tests Variable + Provide + Injectable coexisting:
// Variable (os.Stdout as Writer) + Injectable UserService (depends on Writer) -> topological order.
func TestIntegration_VariableMixed(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/variable_mixed"
	analysistest.Run(t, testdir, depAnalyzer, "variable_mixed/domain")
	analysistest.Run(t, testdir, depAnalyzer, "variable_mixed/config")
	analysistest.Run(t, testdir, depAnalyzer, "variable_mixed/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_VariableTypedNamed tests Variable with mixed options via anonymous interface:
// Variable[interface{ variable.Typed[I]; variable.Named[N] }] -> typed + named registration.
func TestIntegration_VariableTypedNamed(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/variable_typed_named"
	analysistest.Run(t, testdir, depAnalyzer, "variable_typed_named/domain")
	analysistest.Run(t, testdir, depAnalyzer, "variable_typed_named/config")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_VariableCrossPackage tests Variable in non-main package:
// Variable in config package + Injectable in service package -> cross-package resolution.
func TestIntegration_VariableCrossPackage(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/variable_cross_package"
	analysistest.Run(t, testdir, depAnalyzer, "variable_cross_package/config")
	analysistest.Run(t, testdir, depAnalyzer, "variable_cross_package/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_VariableIdempotent tests Variable bootstrap idempotency:
// pre-existing bootstrap with correct hash -> no regeneration, no diagnostics.
func TestIntegration_VariableIdempotent(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/variable_idempotent"
	analysistest.Run(t, testdir, depAnalyzer, ".")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_VariableOutdated tests Variable bootstrap update detection:
// pre-existing bootstrap with wrong hash -> "bootstrap code is outdated" diagnostic + regeneration.
func TestIntegration_VariableOutdated(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/variable_outdated"
	analysistest.Run(t, testdir, depAnalyzer, "variable_outdated/config")
	analysistest.Run(t, testdir, depAnalyzer, "variable_outdated/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_ErrorVariableTyped tests Variable[variable.Typed[I]] constraint violation:
// argument type does not implement typed interface -> fatal validation error -> AppAnalyzer skipped.
func TestIntegration_ErrorVariableTyped(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/error_variable_typed"

	// Phase 1: DependencyAnalyzer scans domain (interface), then config (Variable call triggers error)
	analysistest.Run(t, testdir, depAnalyzer, "error_variable_typed/domain")
	analysistest.Run(t, testdir, depAnalyzer, "error_variable_typed/config")

	// Phase 2: AppAnalyzer skips due to cancelled validation context (no diagnostics expected)
	analysistest.Run(t, testdir, appAnalyzer, ".")
}

// TestIntegration_ErrorVariableNamer tests Variable[variable.Named[N]] with non-literal Name() return:
// Namer returns variable instead of string literal -> fatal validation error -> AppAnalyzer skipped.
func TestIntegration_ErrorVariableNamer(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/error_variable_namer"

	// Phase 1: DependencyAnalyzer scans config first (Namer type), then main (Variable call triggers error)
	analysistest.Run(t, testdir, depAnalyzer, "error_variable_namer/config")
	analysistest.Run(t, testdir, depAnalyzer, ".")

	// Phase 2: AppAnalyzer skips due to cancelled validation context (no diagnostics expected)
	analysistest.Run(t, testdir, appAnalyzer, ".")
}

// TestIntegration_ErrorVariableDuplicateName tests duplicate (TypeName, Name) detection for Variables:
// Same named Variable registered twice -> non-fatal warning emitted.
// Uses programmatic registry access since analysistest cannot naturally test duplicate registration.
func TestIntegration_ErrorVariableDuplicateName(t *testing.T) {
	variableReg := registry.NewVariableRegistry()

	// First registration succeeds
	err := variableReg.Register(&registry.VariableInfo{
		TypeName:       "os.File",
		PackagePath:    "os",
		PackageName:    "os",
		LocalName:      "File",
		ExpressionText: "os.Stdout",
		Dependencies:   []string{},
		Name:           "stdout",
	})
	if err != nil {
		t.Fatalf("First registration should succeed, got: %v", err)
	}

	// Duplicate registration returns error
	err = variableReg.Register(&registry.VariableInfo{
		TypeName:       "os.File",
		PackagePath:    "another",
		PackageName:    "another",
		LocalName:      "File",
		ExpressionText: "another.Stdout",
		Dependencies:   []string{},
		Name:           "stdout",
	})
	if err == nil {
		t.Fatal("Duplicate registration should return error")
	}
	if !strings.Contains(err.Error(), "duplicate named dependency") {
		t.Fatalf("Error should mention 'duplicate named dependency', got: %v", err)
	}
}

// TestIntegration_ErrorVariableNameMismatch tests unresolvable dependency with named Variable hint:
// Injectable depends on *os.File (unnamed) but only os.File#stdout exists -> hint emitted.
func TestIntegration_ErrorVariableNameMismatch(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/error_variable_name_mismatch"

	// Phase 1: DependencyAnalyzer scans config (Variable) and service (Injectable)
	analysistest.Run(t, testdir, depAnalyzer, "error_variable_name_mismatch/config")
	analysistest.Run(t, testdir, depAnalyzer, "error_variable_name_mismatch/service")

	// Phase 2: AppAnalyzer fails to resolve *os.File (unnamed) and emits hint about os.File#stdout
	analysistest.Run(t, testdir, appAnalyzer, ".")
}

// TestIntegration_VariableAliasImport tests Variable with aliased import normalization (Bug #7):
// import myos "os" + annotation.Variable[variable.Default](myos.Stdout) -> ExpressionText = "os.Stdout".
// Bootstrap should use the declared package name "os", not the user alias "myos".
func TestIntegration_VariableAliasImport(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/variable_alias_import"
	analysistest.Run(t, testdir, depAnalyzer, "variable_alias_import/config")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_VariablePkgCollision tests Variable with package name collision (Bug #8):
// Two packages named "config" (v1/config and v2/config) both referenced by Variable expressions.
// Collision aliases (v1config, v2config) must be reflected in ExpressionText rewriting.
func TestIntegration_VariablePkgCollision(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/variable_pkg_collision"
	analysistest.Run(t, testdir, depAnalyzer, "variable_pkg_collision/v1/config")
	analysistest.Run(t, testdir, depAnalyzer, "variable_pkg_collision/v2/config")
	analysistest.Run(t, testdir, depAnalyzer, "variable_pkg_collision/reg")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_ErrorVariableUnresolvableExpression tests that Variable annotations
// with unsupported expression types emit diagnostic errors and cancel bootstrap generation.
// Using a primitive literal (42) as the Variable argument: the argument is *ast.BasicLit,
// not *ast.Ident or *ast.SelectorExpr, so the detector emits a diagnostic error.
// Since the bootstrap context is cancelled, the App annotation does not emit any diagnostic.
func TestIntegration_ErrorVariableUnresolvableExpression(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/error_variable_unresolvable"

	// Phase 1: DependencyAnalyzer detects unsupported Variable argument and emits error
	analysistest.Run(t, testdir, depAnalyzer, "error_variable_unresolvable/config")

	// Phase 2: AppAnalyzer skips bootstrap due to cancelled context (no diagnostic expected)
	analysistest.Run(t, testdir, appAnalyzer, ".")
}

// --- Group G: Struct Tag Integration Tests ---

// TestIntegration_StructTagNamed tests braider:"name" struct tag with Provide[Named]:
// Injectable struct with braider:"primaryRepo" field -> Provide[Named] with matching name -> named wiring in bootstrap.
func TestIntegration_StructTagNamed(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/struct_tag_named"

	// Phase 1: DependencyAnalyzer scans repository (Named provider) and service (struct tag named dep)
	analysistest.Run(t, testdir, depAnalyzer, "struct_tag_named/repository")
	analysistest.Run(t, testdir, depAnalyzer, "struct_tag_named/service")

	// Phase 2: AppAnalyzer generates bootstrap with named wiring
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_StructTagExclude tests braider:"-" struct tag for field exclusion:
// Injectable struct with one normal field and one braider:"-" field -> bootstrap only wires the non-excluded field.
func TestIntegration_StructTagExclude(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/struct_tag_exclude"

	// Phase 1: DependencyAnalyzer scans service (excluded field + normal field)
	analysistest.Run(t, testdir, depAnalyzer, "struct_tag_exclude/service")

	// Phase 2: AppAnalyzer generates bootstrap without excluded field
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_StructTagMixed tests mixed struct tags in a single struct:
// braider:"name" + braider:"-" + untagged -> bootstrap correctly handles each field type.
func TestIntegration_StructTagMixed(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/struct_tag_mixed"

	// Phase 1: DependencyAnalyzer scans repository (Named provider) and service (mixed tags)
	analysistest.Run(t, testdir, depAnalyzer, "struct_tag_mixed/repository")
	analysistest.Run(t, testdir, depAnalyzer, "struct_tag_mixed/service")

	// Phase 2: AppAnalyzer generates bootstrap with mixed tag handling
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_StructTagAllExcluded tests all non-annotation fields excluded via braider:"-":
// All fields have braider:"-" -> zero-param constructor generated -> bootstrap uses zero-arg constructor.
func TestIntegration_StructTagAllExcluded(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/struct_tag_all_excluded"

	// Phase 1: DependencyAnalyzer scans service (all fields excluded)
	analysistest.Run(t, testdir, depAnalyzer, "struct_tag_all_excluded/service")

	// Phase 2: AppAnalyzer generates bootstrap with zero-arg constructor
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_ErrorStructTagEmpty tests braider:"" invalid tag diagnostic:
// Field with empty braider tag value -> non-fatal diagnostic emitted -> bootstrap still generates.
func TestIntegration_ErrorStructTagEmpty(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/error_struct_tag_empty"

	// Phase 1: DependencyAnalyzer detects invalid empty tag and emits diagnostic (non-fatal)
	analysistest.Run(t, testdir, depAnalyzer, "error_struct_tag_empty/service")

	// Phase 2: AppAnalyzer generates bootstrap (non-fatal error doesn't cancel)
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_ErrorStructTagConflict tests WithoutConstructor + braider:"-" conflict:
// Injectable[WithoutConstructor] with excluded field matching constructor parameter type -> non-fatal diagnostic.
func TestIntegration_ErrorStructTagConflict(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/error_struct_tag_conflict"

	// Phase 1: DependencyAnalyzer detects struct tag conflict and emits diagnostic (non-fatal)
	analysistest.Run(t, testdir, depAnalyzer, "error_struct_tag_conflict/service")

	// Phase 2: AppAnalyzer generates bootstrap (non-fatal error doesn't cancel)
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_StructTagIdempotent tests idempotent behavior with braider struct tags.
// When bootstrap code with correct hash already exists (including braider:"name" tagged fields),
// re-running the analyzer should produce NO diagnostic.
// Verifies hash stability: TypeName, ConstructorName, IsField, Dependencies (with #name composite keys).
func TestIntegration_StructTagIdempotent(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/struct_tag_idempotent"
	analysistest.Run(t, testdir, depAnalyzer, "struct_tag_idempotent/repository")
	analysistest.Run(t, testdir, depAnalyzer, "struct_tag_idempotent/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_StructTagOutdated tests hash mismatch detection with braider struct tags.
// When existing bootstrap has wrong hash (braider:"name" fields changed), re-running should
// detect mismatch and regenerate bootstrap code.
// Verifies that struct tag named dependencies affect hash computation correctly.
func TestIntegration_StructTagOutdated(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/struct_tag_outdated"
	analysistest.Run(t, testdir, depAnalyzer, "struct_tag_outdated/repository")
	analysistest.Run(t, testdir, depAnalyzer, "struct_tag_outdated/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// TestIntegration_StructTagTypedFields tests braider:"name" tags across all supported field types.
// Concrete type, pointer type, and interface type fields each with named tags and matching
// named providers. Verifies constructor generation and bootstrap wiring for all three type variants.
func TestIntegration_StructTagTypedFields(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/struct_tag_typed_fields"
	analysistest.Run(t, testdir, depAnalyzer, "struct_tag_typed_fields/domain")
	analysistest.Run(t, testdir, depAnalyzer, "struct_tag_typed_fields/provider")
	analysistest.Run(t, testdir, depAnalyzer, "struct_tag_typed_fields/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}
