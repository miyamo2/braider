// Package analyzer provides integration tests for complete DependencyAnalyzer + AppAnalyzer pipeline.
//
// These tests verify end-to-end behavior using checkertest, running actual analyzers
// against testdata with real annotation types via phasedchecker's two-phase pipeline.
package analyzer

import (
	"strings"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/graph"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"github.com/miyamo2/phasedchecker"
	"github.com/miyamo2/phasedchecker/checkertest"
	"golang.org/x/tools/go/analysis"
)

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
	agg := NewAggregator(providerReg, injectorReg, variableReg)

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
		appOptionExtractor, containerValidator, containerResolver,
	)
	appAnalyzer := (*analysis.Analyzer)(NewAppAnalyzer(appRunner))

	return depAnalyzer, appAnalyzer, injectorReg, providerReg, variableReg, agg
}

// TestIntegration runs all standard integration tests as table-driven subtests.
// Each test case creates fresh analyzers with isolated registries, runs the two-phase
// pipeline via checkertest, and verifies diagnostics and suggested fixes.
func TestIntegration(t *testing.T) {
	tests := []struct {
		name          string
		testdir       string
		appSuggestFix bool
	}{
		// --- Core scenarios ---
		{name: "BasicSinglePackage", testdir: "basic", appSuggestFix: true},
		{name: "CrossPackageImports", testdir: "crosspackage", appSuggestFix: true},
		{name: "MultiTypeCrossPackage", testdir: "multitype", appSuggestFix: true},
		{name: "SimpleApp", testdir: "simpleapp", appSuggestFix: true},
		{name: "SameFileApp", testdir: "samefileapp", appSuggestFix: true},
		{name: "EmptyGraph", testdir: "emptygraph", appSuggestFix: true},
		{name: "DependencyAlreadyUsed", testdir: "depinuse", appSuggestFix: true},
		{name: "DependencyBlankIdentifier", testdir: "depblank", appSuggestFix: true},
		{name: "PackageNameCollision", testdir: "pkgcollision", appSuggestFix: true},
		{name: "ModuleWideDiscovery", testdir: "modulewide", appSuggestFix: true},
		{name: "WithoutConstructor", testdir: "without_constructor", appSuggestFix: true},

		// --- Interface resolution ---
		{name: "InterfaceResolution", testdir: "iface", appSuggestFix: true},
		{name: "InterfaceDependency", testdir: "ifacedep", appSuggestFix: true},
		{name: "CrossPackageInterface", testdir: "crossiface", appSuggestFix: true},

		// --- Typed/Named inject ---
		{name: "TypedInject", testdir: "typed_inject", appSuggestFix: true},
		{name: "NamedInject", testdir: "named_inject", appSuggestFix: true},

		// --- Provide variations ---
		{name: "ProvideTyped", testdir: "provide_typed", appSuggestFix: true},
		{name: "ProvideNamed", testdir: "provide_named", appSuggestFix: true},
		{name: "ProvideCrossType", testdir: "provide_cross_type", appSuggestFix: true},

		// --- Variable annotation ---
		{name: "VariableBasic", testdir: "variable_basic", appSuggestFix: true},
		{name: "VariableTyped", testdir: "variable_typed", appSuggestFix: true},
		{name: "VariableNamed", testdir: "variable_named", appSuggestFix: true},
		{name: "VariableMixed", testdir: "variable_mixed", appSuggestFix: true},
		{name: "VariableTypedNamed", testdir: "variable_typed_named", appSuggestFix: true},
		{name: "VariableCrossPackage", testdir: "variable_cross_package", appSuggestFix: true},
		{name: "VariableAliasImport", testdir: "variable_alias_import", appSuggestFix: true},
		{name: "VariablePkgCollision", testdir: "variable_pkg_collision", appSuggestFix: true},
		{name: "VariableIdentExternalType", testdir: "variable_ident_ext_type", appSuggestFix: true},

		// --- Idempotent/outdated ---
		{name: "IdempotentBehavior", testdir: "idempotent", appSuggestFix: true},
		{name: "IdempotentImport", testdir: "idempotent_import", appSuggestFix: true},
		{name: "BootstrapUpdate", testdir: "outdated", appSuggestFix: true},
		{name: "VariableIdempotent", testdir: "variable_idempotent", appSuggestFix: true},
		{name: "VariableOutdated", testdir: "variable_outdated", appSuggestFix: true},

		// --- Struct tag ---
		{name: "StructTagNamed", testdir: "struct_tag_named", appSuggestFix: true},
		{name: "StructTagExclude", testdir: "struct_tag_exclude", appSuggestFix: true},
		{name: "StructTagMixed", testdir: "struct_tag_mixed", appSuggestFix: true},
		{name: "StructTagAllExcluded", testdir: "struct_tag_all_excluded", appSuggestFix: true},
		{name: "StructTagIdempotent", testdir: "struct_tag_idempotent", appSuggestFix: true},
		{name: "StructTagOutdated", testdir: "struct_tag_outdated", appSuggestFix: true},
		{name: "StructTagTypedFields", testdir: "struct_tag_typed_fields", appSuggestFix: true},

		// --- Container mode ---
		{name: "ContainerBasic", testdir: "container_basic", appSuggestFix: true},
		{name: "ContainerNamed", testdir: "container_named", appSuggestFix: true},
		{name: "ContainerIdempotent", testdir: "container_idempotent", appSuggestFix: true},
		{name: "ContainerOutdated", testdir: "container_outdated", appSuggestFix: true},
		{name: "ContainerAnonymous", testdir: "container_anonymous", appSuggestFix: true},
		{name: "ContainerNamedField", testdir: "container_named_field", appSuggestFix: true},
		{name: "ContainerIfaceField", testdir: "container_iface_field", appSuggestFix: true},
		{name: "ContainerCrossPackage", testdir: "container_cross_package", appSuggestFix: true},
		{name: "ContainerTransitive", testdir: "container_transitive", appSuggestFix: true},
		{name: "ContainerVariable", testdir: "container_variable", appSuggestFix: true},
		{name: "ContainerMixedOption", testdir: "container_mixed_option", appSuggestFix: true},
		{name: "ContainerProvideCrossType", testdir: "container_provide_cross_type", appSuggestFix: true},

		// --- Container error cases ---
		{name: "ErrorContainerUnresolved", testdir: "error_container_unresolved", appSuggestFix: false},
		{name: "ErrorContainerTagExclude", testdir: "error_container_tag_exclude", appSuggestFix: false},
		{name: "ErrorContainerTagEmpty", testdir: "error_container_tag_empty", appSuggestFix: false},
		{name: "ErrorContainerNonStruct", testdir: "error_container_non_struct", appSuggestFix: false},
		{name: "ErrorContainerAmbiguous", testdir: "error_container_ambiguous", appSuggestFix: false},

		// --- App-only (no DependencyAnalyzer) ---
		{name: "NonMainReference", testdir: "nonmainapp", appSuggestFix: true},
		{name: "NoAppAnnotation", testdir: "noapp", appSuggestFix: true},

		// --- Error cases (AppAnalyzer uses Run, not RunWithSuggestedFixes) ---
		{name: "ErrorCases", testdir: "error_cases", appSuggestFix: false},
		{name: "ErrorNonLiteralNamer", testdir: "error_nonliteral", appSuggestFix: false},
		{name: "ErrorProvideTyped", testdir: "error_provide_typed", appSuggestFix: false},
		{name: "ErrorVariableTyped", testdir: "error_variable_typed", appSuggestFix: false},
		{name: "ErrorVariableNamer", testdir: "error_variable_namer", appSuggestFix: false},
		{name: "ErrorVariableNameMismatch", testdir: "error_variable_name_mismatch", appSuggestFix: false},
		{name: "ErrorVariableUnresolvableExpression", testdir: "error_variable_unresolvable", appSuggestFix: false},

		// --- Error cases (non-fatal, AppAnalyzer still generates bootstrap) ---
		{name: "ErrorUnresolvedParam", testdir: "unresolvedparam", appSuggestFix: true},
		{name: "ErrorUnresolvedParamDetail", testdir: "unresparam", appSuggestFix: true},
		{name: "CircularDependency", testdir: "circular", appSuggestFix: true},
		{name: "AmbiguousInterface", testdir: "ambiguous", appSuggestFix: true},
		{name: "AmbiguousInterfaceProvide", testdir: "ambiguousiface", appSuggestFix: true},
		{name: "UnresolvedInterface", testdir: "unresiface", appSuggestFix: true},
		{name: "UnresolvedInterfaceDependency", testdir: "unresolvedif", appSuggestFix: true},
		{name: "ErrorStructTagEmpty", testdir: "error_struct_tag_empty", appSuggestFix: true},
		{name: "ErrorStructTagConflict", testdir: "error_struct_tag_conflict", appSuggestFix: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			depAnalyzer, appAnalyzer, agg := setupIntegrationDeps(t)
			testdir := "testdata/bootstrapgen/" + tt.testdir

			cfg := phasedchecker.Config{
				Pipeline: phasedchecker.Pipeline{
					Phases: []phasedchecker.Phase{
						{Name: "dependency", Analyzers: []*analysis.Analyzer{depAnalyzer}, AfterPhase: agg.AfterDependencyPhase},
						{Name: "app", Analyzers: []*analysis.Analyzer{appAnalyzer}},
					},
				},
				DiagnosticPolicy: phasedchecker.DiagnosticPolicy{
					Rules: []phasedchecker.CategoryRule{
						{Category: "braider:fatal", Severity: phasedchecker.SeverityCritical},
					},
				},
			}

			if tt.appSuggestFix {
				checkertest.RunWithSuggestedFixes(t, testdir, cfg, "./...")
			} else {
				checkertest.Run(t, testdir, cfg, "./...")
			}
		})
	}
}

// TestIntegration_ErrorDuplicateName tests duplicate (TypeName, Name) detection:
// Same named dependency registered twice -> non-fatal warning emitted.
// Uses programmatic registry access since analysistest cannot naturally test duplicate registration
// (each analysistest.Run creates a fresh analysis pass for the same source).
func TestIntegration_ErrorDuplicateName(t *testing.T) {
	depAnalyzer, _, injectorReg, _, _, agg := buildIntegrationDeps(t)
	testdir := "testdata/bootstrapgen/error_duplicate_name"

	// Run single-phase pipeline to register Named service
	cfg := phasedchecker.Config{
		Pipeline: phasedchecker.Pipeline{
			Phases: []phasedchecker.Phase{
				{Name: "dependency", Analyzers: []*analysis.Analyzer{depAnalyzer}, AfterPhase: agg.AfterDependencyPhase},
			},
		},
	}
	checkertest.Run(t, testdir, cfg, "./...")

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
