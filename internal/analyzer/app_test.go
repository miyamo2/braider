package analyzer

import (
	"iter"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/graph"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/packages"
)

// mockPackageLoader is a test implementation that returns an empty package list.
type mockPackageLoader struct{}

func (m *mockPackageLoader) LoadModulePackageNames(dir string) ([]string, error) {
	return []string{}, nil
}

func (m *mockPackageLoader) LoadModulePackageAST(dir string) (iter.Seq[*packages.Package], error) {
	return func(yield func(*packages.Package) bool) {}, nil
}

func (m *mockPackageLoader) FindModuleRoot(dir string) (string, error) {
	return dir, nil
}

func (m *mockPackageLoader) LoadPackage(pkgPath string) (*packages.Package, error) {
	return nil, nil
}

// setupTestDependencies creates all required dependencies for AppAnalyzer-only tests (Group E).
func setupTestDependencies() (
	*registry.ProviderRegistry,
	*registry.InjectorRegistry,
	*registry.PackageTracker,
	*registry.ValidationContext,
	detect.AppDetector,
	*graph.DependencyGraphBuilder,
	*graph.TopologicalSorter,
	generate.BootstrapGenerator,
	report.SuggestedFixBuilder,
	report.DiagnosticEmitter,
) {
	providerRegistry := registry.NewProviderRegistry()
	injectorRegistry := registry.NewInjectorRegistry()
	packageTracker := registry.NewPackageTracker()
	validationContext := registry.NewValidationContext()
	appDetector := detect.NewAppDetector()
	graphBuilder := graph.NewDependencyGraphBuilder()
	sorter := graph.NewTopologicalSorter()
	bootstrapGenerator := generate.NewBootstrapGenerator()
	suggestedFixBuilder := report.NewSuggestedFixBuilder()
	diagnosticEmitter := report.NewDiagnosticEmitter()

	return providerRegistry, injectorRegistry, packageTracker, validationContext, appDetector,
		graphBuilder, sorter, bootstrapGenerator, suggestedFixBuilder, diagnosticEmitter
}

// --- Group A: Sub-package Injectable/Provide → Two-phase pipeline (11 tests) ---

func TestBootstrap_BasicSinglePackage(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/basic"
	analysistest.Run(t, testdir, depAnalyzer, "example.com/basic/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_CrossPackageImports(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/crosspackage"
	analysistest.Run(t, testdir, depAnalyzer, "crosspackage/repository")
	analysistest.Run(t, testdir, depAnalyzer, "crosspackage/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_MultiTypeCrossPackage(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/multitype"
	analysistest.Run(t, testdir, depAnalyzer, "example.com/multitype/repository")
	analysistest.Run(t, testdir, depAnalyzer, "example.com/multitype/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_InterfaceDependency(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/ifacedep"
	analysistest.Run(t, testdir, depAnalyzer, "example.com/ifacedep/domain")
	analysistest.Run(t, testdir, depAnalyzer, "example.com/ifacedep/repository")
	analysistest.Run(t, testdir, depAnalyzer, "example.com/ifacedep/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_InterfaceResolution(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/iface"
	analysistest.Run(t, testdir, depAnalyzer, "iface/domain")
	analysistest.Run(t, testdir, depAnalyzer, "iface/repository")
	analysistest.Run(t, testdir, depAnalyzer, "iface/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_CrossPackageInterface(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/crossiface"
	analysistest.Run(t, testdir, depAnalyzer, "crossiface/domain")
	analysistest.Run(t, testdir, depAnalyzer, "crossiface/repository")
	analysistest.Run(t, testdir, depAnalyzer, "crossiface/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_ModuleWideDiscovery(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/modulewide"
	analysistest.Run(t, testdir, depAnalyzer, "modulewide/repository")
	analysistest.Run(t, testdir, depAnalyzer, "modulewide/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_IdempotentImport(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/idempotent_import"
	analysistest.Run(t, testdir, depAnalyzer, "idempotent_import/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_CircularDependency(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/circular"
	analysistest.Run(t, testdir, depAnalyzer, "circular/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_AmbiguousInterface(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/ambiguous"
	analysistest.Run(t, testdir, depAnalyzer, "ambiguous/domain")
	analysistest.Run(t, testdir, depAnalyzer, "ambiguous/repository")
	analysistest.Run(t, testdir, depAnalyzer, "ambiguous/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_UnresolvedInterface(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/unresiface"
	analysistest.Run(t, testdir, depAnalyzer, "unresiface/writer")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// --- Group B: Root package Injectable, no // want on App → Two-phase (2 tests) ---

func TestBootstrap_DependencyAlreadyUsed(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/depinuse"
	analysistest.Run(t, testdir, depAnalyzer, ".")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_IdempotentBehavior(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/idempotent"
	analysistest.Run(t, testdir, depAnalyzer, ".")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// --- Group C: Root Injectable moved to sub-package → Two-phase (3 tests) ---

func TestBootstrap_DependencyBlankIdentifier(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/depblank"
	analysistest.Run(t, testdir, depAnalyzer, "example.com/depblank/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_BootstrapUpdate(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/outdated"
	analysistest.Run(t, testdir, depAnalyzer, "outdated/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_SameFileApp(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/samefileapp"
	analysistest.Run(t, testdir, depAnalyzer, "samefileapp/service")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// --- Group D: No Injectable/Provide → Simple Two-phase (3 tests) ---

func TestBootstrap_EmptyGraph(t *testing.T) {
	_, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/emptygraph"
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_NonMainReference(t *testing.T) {
	_, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/nonmainapp"
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

func TestBootstrap_NoAppAnnotation(t *testing.T) {
	_, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/noapp"
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}

// --- Group E: AppAnalyzer-only (3 tests) ---

func TestAppAnalyzer_ContextCancellation(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, validationContext, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "contextcancel.TestService",
			PackagePath:     "contextcancel",
			PackageName:     "main",
			LocalName:       "TestService",
			ConstructorName: "NewTestService",
			Dependencies:    []string{},
			Implements:      []string{},
			IsPending:       false,
		},
	)
	packageTracker.MarkPackageScanned("contextcancel")

	// Cancel the validation context to simulate fatal validation error
	validationContext.Cancel()

	packageLoader := &mockPackageLoader{}
	analyzer := AppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader,
		packageTracker, validationContext,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)

	// Should not emit any diagnostics when context is cancelled
	analysistest.Run(t, "testdata/bootstrapgen/contextcancel", analyzer, ".")
}

func TestAppAnalyzer_MissingConstructor(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, validationContext, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	providerRegistry.Register(
		&registry.ProviderInfo{
			TypeName:        "missingctor/repository.UserRepository",
			PackagePath:     "missingctor/repository",
			PackageName:     "repository",
			LocalName:       "UserRepository",
			ConstructorName: "", // No constructor
			Dependencies:    []string{},
			Implements:      []string{},
			IsPending:       false,
		},
	)
	packageTracker.MarkPackageScanned("missingctor")

	packageLoader := &mockPackageLoader{}
	analyzer := AppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader,
		packageTracker, validationContext,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/bootstrapgen/missingctor", analyzer, ".")
	analysistest.RunWithSuggestedFixes(t, "testdata/bootstrapgen/missingctor", analyzer, ".")
}

func TestAppAnalyzer_MultipleEntryPoints(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, validationContext, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "main.Service1",
			PackagePath:     "multipleapp/cmd/1",
			PackageName:     "main",
			LocalName:       "Service1",
			ConstructorName: "NewService1",
			Dependencies:    []string{},
			IsPending:       false,
		},
	)
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "main.Service2",
			PackagePath:     "multipleapp/cmd/2",
			PackageName:     "main",
			LocalName:       "Service2",
			ConstructorName: "NewService2",
			Dependencies:    []string{},
			IsPending:       false,
		},
	)
	packageTracker.MarkPackageScanned("multipleapp/cmd/1")
	packageTracker.MarkPackageScanned("multipleapp/cmd/2")

	packageLoader := &mockPackageLoader{}
	analyzer := AppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader,
		packageTracker, validationContext,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/bootstrapgen/multipleapp", analyzer, "./...")
}
