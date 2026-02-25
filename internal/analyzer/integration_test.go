// Package analyzer provides integration tests for complete DependencyAnalyzer + AppAnalyzer pipeline.
//
// These tests verify end-to-end behavior using checkertest, running actual analyzers
// against testdata with real annotation types via phasedchecker's two-phase pipeline.
package analyzer

import (
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/graph"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"github.com/miyamo2/phasedchecker"
	"github.com/miyamo2/phasedchecker/checkertest"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

// mockPackageLoader is a test implementation that returns an empty package list.
type mockPackageLoader struct{}

func (m *mockPackageLoader) LoadPackage(pkgPath string) (*packages.Package, error) {
	return nil, nil
}

// setupIntegrationDeps creates shared registries and real components for integration tests.
// Returns both DependencyAnalyzer and AppAnalyzer configured with the same shared state,
// plus the Aggregator for wiring the AfterPhase callback.
func setupIntegrationDeps(t *testing.T) (*analysis.Analyzer, *analysis.Analyzer, *Aggregator) {
	t.Helper()
	depAnalyzer, appAnalyzer, _, _, _, agg := buildIntegrationDeps(t)
	return depAnalyzer, appAnalyzer, agg
}

// buildIntegrationDeps creates all shared components and returns analyzers plus raw registries.
func buildIntegrationDeps(t *testing.T) (
	*analysis.Analyzer, *analysis.Analyzer,
	*registry.InjectorRegistry, *registry.ProviderRegistry,
	*registry.VariableRegistry,
	*Aggregator,
) {
	t.Helper()
	// Shared registries
	providerReg := registry.NewProviderRegistry()
	injectorReg := registry.NewInjectorRegistry()
	variableReg := registry.NewVariableRegistry()

	// Detection components (all real)
	markers, err := detect.ResolveMarkers()
	if err != nil {
		t.Fatal(err)
	}
	packageLoader := &mockPackageLoader{}
	namerValidator := detect.NewNamerValidatorImpl(packageLoader)
	optionExtractor := detect.NewOptionExtractorImpl(markers, namerValidator)
	injectDetector := detect.NewInjectDetector(markers)
	fieldAnalyzer := detect.NewFieldAnalyzer()
	constructorAnalyzer := detect.NewConstructorAnalyzer()
	provideCallDetector := detect.NewProvideCallDetector(markers)
	structDetector := detect.NewStructDetector(injectDetector)

	// Generation components
	constructorGenerator := generate.NewConstructorGenerator()
	bootstrapGenerator := generate.NewBootstrapGenerator()

	// Report components
	suggestedFixBuilder := report.NewSuggestedFixBuilder()
	diagnosticEmitter := report.NewDiagnosticEmitter()

	// Graph components
	interfaceRegistry := graph.NewInterfaceRegistry()
	graphBuilder := graph.NewDependencyGraphBuilder(interfaceRegistry)
	sorter := graph.NewTopologicalSorter()

	// App detection
	appDetector := detect.NewAppDetector(markers)

	// Container components
	appOptionExtractor := detect.NewAppOptionExtractorImpl(markers)
	containerValidator := graph.NewContainerValidatorImpl(interfaceRegistry)
	containerResolver := graph.NewContainerResolverImpl(interfaceRegistry)

	// Variable components
	variableCallDetector := detect.NewVariableCallDetector(markers)

	// Aggregator (shared registries)
	duplicateReg := registry.NewDuplicateRegistry()
	agg := NewAggregator(providerReg, injectorReg, variableReg, duplicateReg)

	depRunner := NewDependencyAnalyzeRunner(
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector,
	)
	depAnalyzer := (*analysis.Analyzer)(NewDependencyAnalyzer(depRunner))

	appRunner := NewAppAnalyzeRunner(
		appDetector, injectorReg, providerReg,
		graphBuilder, sorter, bootstrapGenerator,
		suggestedFixBuilder, diagnosticEmitter,
		variableReg,
		appOptionExtractor, containerValidator, containerResolver, agg.DuplicateRegistry,
	)
	appAnalyzer := (*analysis.Analyzer)(NewAppAnalyzer(appRunner))

	return depAnalyzer, appAnalyzer, injectorReg, providerReg, variableReg, agg
}

// TestIntegration runs all standard integration tests as table-driven subtests.
// Each test case creates fresh analyzers with isolated registries, runs the two-phase
// pipeline via checkertest, and verifies diagnostics and suggested fixes.
func TestIntegration(t *testing.T) {
	tests := []struct {
		name    string
		testdir string
	}{
		// --- Core scenarios ---
		{name: "BasicSinglePackage", testdir: "basic"},
		{name: "CrossPackageImports", testdir: "crosspackage"},
		{name: "MultiTypeCrossPackage", testdir: "multitype"},
		{name: "SimpleApp", testdir: "simpleapp"},
		{name: "SameFileApp", testdir: "samefileapp"},
		{name: "EmptyGraph", testdir: "emptygraph"},
		{name: "DependencyAlreadyUsed", testdir: "depinuse"},
		{name: "PackageNameCollision", testdir: "pkgcollision"},
		{name: "ModuleWideDiscovery", testdir: "modulewide"},
		{name: "WithoutConstructor", testdir: "without_constructor"},

		// --- Interface resolution ---
		{name: "InterfaceResolution", testdir: "iface"},
		{name: "InterfaceDependency", testdir: "ifacedep"},
		{name: "CrossPackageInterface", testdir: "crossiface"},

		// --- Typed/Named inject ---
		{name: "TypedInject", testdir: "typed_inject"},
		{name: "NamedInject", testdir: "named_inject"},

		// --- Provide variations ---
		{name: "ProvideTyped", testdir: "provide_typed"},
		{name: "ProvideNamed", testdir: "provide_named"},
		{name: "ProvideCrossType", testdir: "provide_cross_type"},

		// --- Variable annotation ---
		{name: "VariableBasic", testdir: "variable_basic"},
		{name: "VariableTyped", testdir: "variable_typed"},
		{name: "VariableNamed", testdir: "variable_named"},
		{name: "VariableMixed", testdir: "variable_mixed"},
		{name: "VariableTypedNamed", testdir: "variable_typed_named"},
		{name: "VariableCrossPackage", testdir: "variable_cross_package"},
		{name: "VariableAliasImport", testdir: "variable_alias_import"},
		{name: "VariablePkgCollision", testdir: "variable_pkg_collision"},
		{name: "VariableIdentExternalType", testdir: "variable_ident_ext_type"},

		// --- Idempotent/outdated ---
		{name: "IdempotentBehavior", testdir: "idempotent"},
		{name: "BootstrapUpdate", testdir: "outdated"},
		{name: "VariableIdempotent", testdir: "variable_idempotent"},
		{name: "VariableOutdated", testdir: "variable_outdated"},

		// --- Struct tag ---
		{name: "StructTagNamed", testdir: "struct_tag_named"},
		{name: "StructTagExclude", testdir: "struct_tag_exclude"},
		{name: "StructTagMixed", testdir: "struct_tag_mixed"},
		{name: "StructTagAllExcluded", testdir: "struct_tag_all_excluded"},
		{name: "StructTagIdempotent", testdir: "struct_tag_idempotent"},
		{name: "StructTagOutdated", testdir: "struct_tag_outdated"},
		{name: "StructTagTypedFields", testdir: "struct_tag_typed_fields"},

		// --- Container mode ---
		{name: "ContainerBasic", testdir: "container_basic"},
		{name: "ContainerNamed", testdir: "container_named"},
		{name: "ContainerIdempotent", testdir: "container_idempotent"},
		{name: "ContainerOutdated", testdir: "container_outdated"},
		{name: "ContainerAnonymous", testdir: "container_anonymous"},
		{name: "ContainerNamedField", testdir: "container_named_field"},
		{name: "ContainerIfaceField", testdir: "container_iface_field"},
		{name: "ContainerCrossPackage", testdir: "container_cross_package"},
		{name: "ContainerTransitive", testdir: "container_transitive"},
		{name: "ContainerVariable", testdir: "container_variable"},
		{name: "ContainerMixedOption", testdir: "container_mixed_option"},
		{name: "ContainerProvideCrossType", testdir: "container_provide_cross_type"},

		// --- Container error cases ---
		{name: "ErrorContainerUnresolved", testdir: "error_container_unresolved"},
		{name: "ErrorContainerTagExclude", testdir: "error_container_tag_exclude"},
		{name: "ErrorContainerTagEmpty", testdir: "error_container_tag_empty"},
		{name: "ErrorContainerNonStruct", testdir: "error_container_non_struct"},
		{name: "ErrorContainerAmbiguous", testdir: "error_container_ambiguous"},

		// --- App-only (no DependencyAnalyzer) ---
		{name: "NonMainReference", testdir: "nonmainapp"},
		{name: "NoAppAnnotation", testdir: "noapp"},
		{name: "MultipleEntryPoints", testdir: "multipleapp"},

		// --- Error cases ---
		{name: "ErrorCases", testdir: "error_cases"},
		{name: "ErrorNonLiteralNamer", testdir: "error_nonliteral"},
		{name: "ErrorProvideTyped", testdir: "error_provide_typed"},
		{name: "ErrorVariableTyped", testdir: "error_variable_typed"},
		{name: "ErrorVariableNamer", testdir: "error_variable_namer"},
		{name: "ErrorVariableNameMismatch", testdir: "error_variable_name_mismatch"},
		{name: "ErrorVariableUnresolvableExpression", testdir: "error_variable_unresolvable"},
		{name: "ErrorDuplicateName", testdir: "error_duplicate_name"},
		{name: "ErrorVariableDuplicateName", testdir: "error_variable_duplicate_name"},

		// --- Error cases (non-fatal, AppAnalyzer still generates bootstrap) ---
		{name: "ErrorUnresolvedParam", testdir: "unresolvedparam"},
		{name: "ErrorUnresolvedParamDetail", testdir: "unresparam"},
		{name: "CircularDependency", testdir: "circular"},
		{name: "AmbiguousInterface", testdir: "ambiguous"},
		{name: "AmbiguousInterfaceProvide", testdir: "ambiguousiface"},
		{name: "UnresolvedInterface", testdir: "unresiface"},
		{name: "UnresolvedInterfaceDependency", testdir: "unresolvedif"},
		{name: "ErrorStructTagEmpty", testdir: "error_struct_tag_empty"},
		{name: "ErrorStructTagConflict", testdir: "error_struct_tag_conflict"},

		// --- Constructor generation ---
		{name: "ConstructorGeneration", testdir: "constructorgen"},

		// --- Dependency-only smoke tests ---
		// These test cases have NO annotation.App, NO // want directives, and NO .golden files.
		// They verify that the dependency phase processes valid annotation patterns
		// without crashing and without emitting unexpected diagnostics.
		// (checkertest reports unexpected diagnostics as test failures via checkDiagnostics)
		{name: "DepBasicRegistration", testdir: "dep_basic"},              // Injectable + Provide in same package
		{name: "DepMissingConstructor", testdir: "dep_missing_constructor"}, // plain struct without any annotation
		{name: "DepCrossPackage", testdir: "dep_cross_package"},            // Injectable depending on cross-package Provide
		{name: "DepInterfaceImplementation", testdir: "dep_interface_impl"}, // interface + Provide + Injectable
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				depAnalyzer, appAnalyzer, agg := setupIntegrationDeps(t)
				testdir := "testdata/e2e/" + tt.testdir

				cfg := phasedchecker.Config{
					Pipeline: phasedchecker.Pipeline{
						Phases: []phasedchecker.Phase{
							{
								Name: "dependency", Analyzers: []*analysis.Analyzer{depAnalyzer},
								AfterPhase: agg.AfterDependencyPhase,
							},
							{Name: "app", Analyzers: []*analysis.Analyzer{appAnalyzer}},
						},
					},
					DiagnosticPolicy: phasedchecker.DiagnosticPolicy{
						Rules: []phasedchecker.CategoryRule{
							{Category: report.CategoryOptionValidation, Severity: phasedchecker.SeverityCritical},
							{Category: report.CategoryExpressionValidation, Severity: phasedchecker.SeverityCritical},
						},
					},
				}

				checkertest.RunWithSuggestedFixes(t, testdir, cfg, "./...")
			},
		)
	}
}
