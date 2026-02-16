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
	markers := detect.ResolveMarkers()
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
	bootstrapGenerator := generate.NewBootstrapGenerator(generate.NewCodeFormatter())

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
	variableReg := registry.NewVariableRegistry()

	depRunner := NewDependencyAnalyzeRunner(
		providerReg, injectorReg, packageTracker, bootstrapCancel,
		provideCallDetector, injectDetector, structDetector,
		fieldAnalyzer, constructorAnalyzer, optionExtractor,
		constructorGenerator, suggestedFixBuilder, diagnosticEmitter,
		variableCallDetector, variableReg,
	)
	depAnalyzer := (*analysis.Analyzer)(NewDependencyAnalyzer(depRunner))

	appRunner := NewAppAnalyzeRunner(
		appDetector, injectorReg, providerReg, packageLoader,
		packageTracker, ctx,
		graphBuilder, sorter, bootstrapGenerator,
		suggestedFixBuilder, diagnosticEmitter,
		variableReg,
		appOptionExtractor, containerValidator, containerResolver,
	)
	appAnalyzer := (*analysis.Analyzer)(NewAppAnalyzer(appRunner))

	return depAnalyzer, appAnalyzer, injectorReg, providerReg, variableReg
}

// TestIntegration runs all standard integration tests as table-driven subtests.
// Each test case creates fresh analyzers with isolated registries, scans dependency packages
// with DependencyAnalyzer, then verifies bootstrap generation with AppAnalyzer.
func TestIntegration(t *testing.T) {
	tests := []struct {
		name          string
		testdir       string
		depPackages   []string
		appSuggestFix bool
	}{
		// --- Core scenarios ---
		{
			name:          "BasicSinglePackage",
			testdir:       "basic",
			depPackages:   []string{"example.com/basic/service"},
			appSuggestFix: true,
		},
		{
			name:          "CrossPackageImports",
			testdir:       "crosspackage",
			depPackages:   []string{"crosspackage/repository", "crosspackage/service"},
			appSuggestFix: true,
		},
		{
			name:          "MultiTypeCrossPackage",
			testdir:       "multitype",
			depPackages:   []string{"example.com/multitype/repository", "example.com/multitype/service"},
			appSuggestFix: true,
		},
		{
			name:          "SimpleApp",
			testdir:       "simpleapp",
			depPackages:   []string{"simpleapp/service"},
			appSuggestFix: true,
		},
		{
			name:          "SameFileApp",
			testdir:       "samefileapp",
			depPackages:   []string{"samefileapp/service"},
			appSuggestFix: true,
		},
		{
			name:          "EmptyGraph",
			testdir:       "emptygraph",
			depPackages:   nil,
			appSuggestFix: true,
		},
		{
			name:          "DependencyAlreadyUsed",
			testdir:       "depinuse",
			depPackages:   []string{"."},
			appSuggestFix: true,
		},
		{
			name:          "DependencyBlankIdentifier",
			testdir:       "depblank",
			depPackages:   []string{"example.com/depblank/service"},
			appSuggestFix: true,
		},
		{
			name:          "PackageNameCollision",
			testdir:       "pkgcollision",
			depPackages:   []string{"pkgcollision/v1user", "pkgcollision/v2user"},
			appSuggestFix: true,
		},
		{
			name:          "ModuleWideDiscovery",
			testdir:       "modulewide",
			depPackages:   []string{"modulewide/repository", "modulewide/service"},
			appSuggestFix: true,
		},
		{
			name:          "WithoutConstructor",
			testdir:       "without_constructor",
			depPackages:   []string{"without_constructor/service"},
			appSuggestFix: true,
		},

		// --- Interface resolution ---
		{
			name:          "InterfaceResolution",
			testdir:       "iface",
			depPackages:   []string{"iface/domain", "iface/repository", "iface/service"},
			appSuggestFix: true,
		},
		{
			name:          "InterfaceDependency",
			testdir:       "ifacedep",
			depPackages:   []string{"example.com/ifacedep/domain", "example.com/ifacedep/repository", "example.com/ifacedep/service"},
			appSuggestFix: true,
		},
		{
			name:          "CrossPackageInterface",
			testdir:       "crossiface",
			depPackages:   []string{"crossiface/domain", "crossiface/repository", "crossiface/service"},
			appSuggestFix: true,
		},

		// --- Typed/Named inject ---
		{
			name:          "TypedInject",
			testdir:       "typed_inject",
			depPackages:   []string{"typed_inject/domain", "typed_inject/service"},
			appSuggestFix: true,
		},
		{
			name:          "NamedInject",
			testdir:       "named_inject",
			depPackages:   []string{"named_inject/service"},
			appSuggestFix: true,
		},

		// --- Provide variations ---
		{
			name:          "ProvideTyped",
			testdir:       "provide_typed",
			depPackages:   []string{"provide_typed/domain", "provide_typed/repository", "provide_typed/service"},
			appSuggestFix: true,
		},
		{
			name:          "ProvideNamed",
			testdir:       "provide_named",
			depPackages:   []string{"provide_named/repository"},
			appSuggestFix: true,
		},

		// --- Variable annotation ---
		{
			name:          "VariableBasic",
			testdir:       "variable_basic",
			depPackages:   []string{"variable_basic/config"},
			appSuggestFix: true,
		},
		{
			name:          "VariableTyped",
			testdir:       "variable_typed",
			depPackages:   []string{"variable_typed/domain", "variable_typed/config"},
			appSuggestFix: true,
		},
		{
			name:          "VariableNamed",
			testdir:       "variable_named",
			depPackages:   []string{"variable_named/config"},
			appSuggestFix: true,
		},
		{
			name:          "VariableMixed",
			testdir:       "variable_mixed",
			depPackages:   []string{"variable_mixed/domain", "variable_mixed/config", "variable_mixed/service"},
			appSuggestFix: true,
		},
		{
			name:          "VariableTypedNamed",
			testdir:       "variable_typed_named",
			depPackages:   []string{"variable_typed_named/domain", "variable_typed_named/config"},
			appSuggestFix: true,
		},
		{
			name:          "VariableCrossPackage",
			testdir:       "variable_cross_package",
			depPackages:   []string{"variable_cross_package/config", "variable_cross_package/service"},
			appSuggestFix: true,
		},
		{
			name:          "VariableAliasImport",
			testdir:       "variable_alias_import",
			depPackages:   []string{"variable_alias_import/config"},
			appSuggestFix: true,
		},
		{
			name:          "VariablePkgCollision",
			testdir:       "variable_pkg_collision",
			depPackages:   []string{"variable_pkg_collision/v1/config", "variable_pkg_collision/v2/config", "variable_pkg_collision/reg"},
			appSuggestFix: true,
		},
		{
			name:          "VariableIdentExternalType",
			testdir:       "variable_ident_ext_type",
			depPackages:   []string{"variable_ident_ext_type/config"},
			appSuggestFix: true,
		},

		// --- Idempotent/outdated ---
		{
			name:          "IdempotentBehavior",
			testdir:       "idempotent",
			depPackages:   []string{"."},
			appSuggestFix: true,
		},
		{
			name:          "IdempotentImport",
			testdir:       "idempotent_import",
			depPackages:   []string{"idempotent_import/service"},
			appSuggestFix: true,
		},
		{
			name:          "BootstrapUpdate",
			testdir:       "outdated",
			depPackages:   []string{"outdated/service"},
			appSuggestFix: true,
		},
		{
			name:          "VariableIdempotent",
			testdir:       "variable_idempotent",
			depPackages:   []string{"."},
			appSuggestFix: true,
		},
		{
			name:          "VariableOutdated",
			testdir:       "variable_outdated",
			depPackages:   []string{"variable_outdated/config", "variable_outdated/service"},
			appSuggestFix: true,
		},

		// --- Struct tag ---
		{
			name:          "StructTagNamed",
			testdir:       "struct_tag_named",
			depPackages:   []string{"struct_tag_named/repository", "struct_tag_named/service"},
			appSuggestFix: true,
		},
		{
			name:          "StructTagExclude",
			testdir:       "struct_tag_exclude",
			depPackages:   []string{"struct_tag_exclude/service"},
			appSuggestFix: true,
		},
		{
			name:          "StructTagMixed",
			testdir:       "struct_tag_mixed",
			depPackages:   []string{"struct_tag_mixed/repository", "struct_tag_mixed/service"},
			appSuggestFix: true,
		},
		{
			name:          "StructTagAllExcluded",
			testdir:       "struct_tag_all_excluded",
			depPackages:   []string{"struct_tag_all_excluded/service"},
			appSuggestFix: true,
		},
		{
			name:          "StructTagIdempotent",
			testdir:       "struct_tag_idempotent",
			depPackages:   []string{"struct_tag_idempotent/repository", "struct_tag_idempotent/service"},
			appSuggestFix: true,
		},
		{
			name:          "StructTagOutdated",
			testdir:       "struct_tag_outdated",
			depPackages:   []string{"struct_tag_outdated/repository", "struct_tag_outdated/service"},
			appSuggestFix: true,
		},
		{
			name:          "StructTagTypedFields",
			testdir:       "struct_tag_typed_fields",
			depPackages:   []string{"struct_tag_typed_fields/domain", "struct_tag_typed_fields/provider", "struct_tag_typed_fields/service"},
			appSuggestFix: true,
		},

		// --- Container mode ---
		{
			name:          "ContainerBasic",
			testdir:       "container_basic",
			depPackages:   []string{"container_basic/service"},
			appSuggestFix: true,
		},
		{
			name:          "ContainerNamed",
			testdir:       "container_named",
			depPackages:   []string{"container_named/service"},
			appSuggestFix: true,
		},
		{
			name:          "ContainerIdempotent",
			testdir:       "container_idempotent",
			depPackages:   []string{"container_idempotent/service"},
			appSuggestFix: true,
		},
		{
			name:          "ContainerOutdated",
			testdir:       "container_outdated",
			depPackages:   []string{"container_outdated/service"},
			appSuggestFix: true,
		},
		{
			name:          "ContainerAnonymous",
			testdir:       "container_anonymous",
			depPackages:   []string{"container_anonymous/repository", "container_anonymous/service"},
			appSuggestFix: true,
		},
		{
			name:          "ContainerNamedField",
			testdir:       "container_named_field",
			depPackages:   []string{"container_named_field/service"},
			appSuggestFix: true,
		},
		{
			name:          "ContainerIfaceField",
			testdir:       "container_iface_field",
			depPackages:   []string{"container_iface_field/repository"},
			appSuggestFix: true,
		},
		{
			name:          "ContainerCrossPackage",
			testdir:       "container_cross_package",
			depPackages:   []string{"container_cross_package/service"},
			appSuggestFix: true,
		},
		{
			name:          "ContainerTransitive",
			testdir:       "container_transitive",
			depPackages:   []string{"container_transitive/repository", "container_transitive/service"},
			appSuggestFix: true,
		},
		{
			name:          "ContainerVariable",
			testdir:       "container_variable",
			depPackages:   []string{"container_variable/config"},
			appSuggestFix: true,
		},
		{
			name:          "ContainerMixedOption",
			testdir:       "container_mixed_option",
			depPackages:   []string{"container_mixed_option/config", "container_mixed_option/service"},
			appSuggestFix: true,
		},

		// --- Container error cases ---
		{
			name:          "ErrorContainerUnresolved",
			testdir:       "error_container_unresolved",
			depPackages:   []string{"error_container_unresolved/service"},
			appSuggestFix: false,
		},
		{
			name:          "ErrorContainerTagExclude",
			testdir:       "error_container_tag_exclude",
			depPackages:   []string{"error_container_tag_exclude/service"},
			appSuggestFix: false,
		},
		{
			name:          "ErrorContainerTagEmpty",
			testdir:       "error_container_tag_empty",
			depPackages:   []string{"error_container_tag_empty/service"},
			appSuggestFix: false,
		},
		{
			name:          "ErrorContainerNonStruct",
			testdir:       "error_container_non_struct",
			depPackages:   nil,
			appSuggestFix: false,
		},
		{
			name:          "ErrorContainerAmbiguous",
			testdir:       "error_container_ambiguous",
			depPackages:   []string{"error_container_ambiguous/repository"},
			appSuggestFix: false,
		},

		// --- App-only (no DependencyAnalyzer) ---
		{
			name:          "NonMainReference",
			testdir:       "nonmainapp",
			depPackages:   nil,
			appSuggestFix: true,
		},
		{
			name:          "NoAppAnnotation",
			testdir:       "noapp",
			depPackages:   nil,
			appSuggestFix: true,
		},

		// --- Error cases (AppAnalyzer uses Run, not RunWithSuggestedFixes) ---
		{
			name:          "ErrorCases",
			testdir:       "error_cases",
			depPackages:   []string{"error_cases/service"},
			appSuggestFix: false,
		},
		{
			name:          "ErrorNonLiteralNamer",
			testdir:       "error_nonliteral",
			depPackages:   []string{"error_nonliteral/service"},
			appSuggestFix: false,
		},
		{
			name:          "ErrorProvideTyped",
			testdir:       "error_provide_typed",
			depPackages:   []string{"error_provide_typed/domain", "error_provide_typed/repository"},
			appSuggestFix: false,
		},
		{
			name:          "ErrorVariableTyped",
			testdir:       "error_variable_typed",
			depPackages:   []string{"error_variable_typed/domain", "error_variable_typed/config"},
			appSuggestFix: false,
		},
		{
			name:          "ErrorVariableNamer",
			testdir:       "error_variable_namer",
			depPackages:   []string{"error_variable_namer/config", "."},
			appSuggestFix: false,
		},
		{
			name:          "ErrorVariableNameMismatch",
			testdir:       "error_variable_name_mismatch",
			depPackages:   []string{"error_variable_name_mismatch/config", "error_variable_name_mismatch/service"},
			appSuggestFix: false,
		},
		{
			name:          "ErrorVariableUnresolvableExpression",
			testdir:       "error_variable_unresolvable",
			depPackages:   []string{"error_variable_unresolvable/config"},
			appSuggestFix: false,
		},

		// --- Error cases (non-fatal, AppAnalyzer still generates bootstrap) ---
		{
			name:          "ErrorUnresolvedParam",
			testdir:       "unresolvedparam",
			depPackages:   []string{"example.com/unresolvedparam/repository", "example.com/unresolvedparam/service"},
			appSuggestFix: true,
		},
		{
			name:          "ErrorUnresolvedParamDetail",
			testdir:       "unresparam",
			depPackages:   []string{"unresparam/repository"},
			appSuggestFix: true,
		},
		{
			name:          "CircularDependency",
			testdir:       "circular",
			depPackages:   []string{"circular/service"},
			appSuggestFix: true,
		},
		{
			name:          "AmbiguousInterface",
			testdir:       "ambiguous",
			depPackages:   []string{"ambiguous/domain", "ambiguous/repository", "ambiguous/service"},
			appSuggestFix: true,
		},
		{
			name:          "AmbiguousInterfaceProvide",
			testdir:       "ambiguousiface",
			depPackages:   []string{"example.com/ambiguousiface/domain", "example.com/ambiguousiface/repository", "example.com/ambiguousiface/service"},
			appSuggestFix: true,
		},
		{
			name:          "UnresolvedInterface",
			testdir:       "unresiface",
			depPackages:   []string{"unresiface/writer"},
			appSuggestFix: true,
		},
		{
			name:          "UnresolvedInterfaceDependency",
			testdir:       "unresolvedif",
			depPackages:   []string{"example.com/unresolvedif/writer"},
			appSuggestFix: true,
		},
		{
			name:          "ErrorStructTagEmpty",
			testdir:       "error_struct_tag_empty",
			depPackages:   []string{"error_struct_tag_empty/service"},
			appSuggestFix: true,
		},
		{
			name:          "ErrorStructTagConflict",
			testdir:       "error_struct_tag_conflict",
			depPackages:   []string{"error_struct_tag_conflict/service"},
			appSuggestFix: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			depAnalyzer, appAnalyzer := setupIntegrationDeps()
			testdir := "testdata/bootstrapgen/" + tt.testdir

			for _, pkg := range tt.depPackages {
				analysistest.Run(t, testdir, depAnalyzer, pkg)
			}

			if tt.appSuggestFix {
				analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
			} else {
				analysistest.Run(t, testdir, appAnalyzer, ".")
			}
		})
	}
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
