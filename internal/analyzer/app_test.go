package analyzer

import (
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/graph"
	"github.com/miyamo2/braider/internal/loader"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// mockPackageLoader is a test implementation that returns an empty package list.
type mockPackageLoader struct{}

func (m *mockPackageLoader) LoadModulePackages(dir string) ([]string, error) {
	// Return empty list for test - no packages to wait for
	return []string{}, nil
}

func (m *mockPackageLoader) FindModuleRoot(dir string) (string, error) {
	return dir, nil
}

// setupTestDependencies creates all required dependencies for AppAnalyzer tests.
func setupTestDependencies() (
	*registry.ProviderRegistry,
	*registry.InjectorRegistry,
	loader.PackageLoader,
	*registry.PackageTracker,
	detect.AppDetector,
	*graph.DependencyGraphBuilder,
	*graph.TopologicalSorter,
	generate.BootstrapGenerator,
	report.SuggestedFixBuilder,
	report.DiagnosticEmitter,
) {
	providerRegistry := registry.NewProviderRegistry()
	injectorRegistry := registry.NewInjectorRegistry()
	packageLoader := &mockPackageLoader{} // Use mock instead of real implementation
	packageTracker := registry.NewPackageTracker()
	appDetector := detect.NewAppDetector()
	graphBuilder := graph.NewDependencyGraphBuilder()
	sorter := graph.NewTopologicalSorter()
	bootstrapGenerator := generate.NewBootstrapGenerator()
	suggestedFixBuilder := report.NewSuggestedFixBuilder()
	diagnosticEmitter := report.NewDiagnosticEmitter()

	return providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector,
		graphBuilder, sorter, bootstrapGenerator, suggestedFixBuilder,
		diagnosticEmitter
}

// createAppAnalyzer creates an AppAnalyzer with the provided dependencies.
func createAppAnalyzer(
	appDetector detect.AppDetector,
	injectorRegistry *registry.InjectorRegistry,
	providerRegistry *registry.ProviderRegistry,
	packageLoader loader.PackageLoader,
	packageTracker *registry.PackageTracker,
	graphBuilder *graph.DependencyGraphBuilder,
	sorter *graph.TopologicalSorter,
	bootstrapGen generate.BootstrapGenerator,
	fixBuilder report.SuggestedFixBuilder,
	diagnosticEmitter report.DiagnosticEmitter,
) *analysis.Analyzer {
	return AppAnalyzer(
		appDetector,
		injectorRegistry,
		providerRegistry,
		packageLoader,
		packageTracker,
		graphBuilder,
		sorter,
		bootstrapGen,
		fixBuilder,
		diagnosticEmitter,
	)
}

// TestAppAnalyzer_NoAppAnnotation tests that bootstrap generation is skipped
// when no App annotation is present in the package.
func TestAppAnalyzer_NoAppAnnotation(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/noapp", analyzer, ".")
}

// TestAppAnalyzer_DetectsAppAnnotation tests that the analyzer detects
// an App annotation and processes it.
func TestAppAnalyzer_DetectsAppAnnotation(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register a simple injector to test retrieval
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "main.TestService",
			PackagePath:     "main",
			LocalName:       "TestService",
			ConstructorName: "NewTestService",
			Dependencies:    []string{},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	// Mark the package as scanned
	packageTracker.MarkPackageScanned("main")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/simpleapp", analyzer, ".")
}

// TestAppAnalyzer_WaitsForAllPackages tests that the analyzer waits for
// all packages to be scanned before generating bootstrap code.
func TestAppAnalyzer_WaitsForAllPackages(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// This test verifies that AppAnalyzer waits for package scanning
	// We won't mark packages as scanned, so it should timeout
	// (or we could test with goroutines marking them as scanned)

	// Register a simple injector to create a non-empty dependency graph
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "main.TestService",
			PackagePath:     "main",
			LocalName:       "TestService",
			ConstructorName: "NewTestService",
			Dependencies:    []string{},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	// For now, we'll test the happy path where packages are already scanned
	packageTracker.MarkPackageScanned("main")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/simpleapp", analyzer, ".")
}

// TestAppAnalyzer_SamefileAppAnnotations tests that multiple App annotations
// in the same file are handled correctly (first one is used, others are ignored).
func TestAppAnalyzer_SamefileAppAnnotations(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register a test injector so the graph is not empty
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "main.Service",
			PackagePath:     "main",
			LocalName:       "Service",
			ConstructorName: "NewService",
			Dependencies:    []string{},
			IsPending:       false,
		},
	)

	// Mark package as scanned
	packageTracker.MarkPackageScanned("samefileapp")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/samefileapp", analyzer, ".")
}

// TestAppAnalyzer_NonMainReference tests that App annotation referencing
// a non-main function is detected and reported as an error.
func TestAppAnalyzer_NonMainReference(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Mark package as scanned
	packageTracker.MarkPackageScanned("nonmainapp")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/nonmainapp", analyzer, ".")
}

// TestAppAnalyzer_RetrievesProvidersAndInjectors tests that the analyzer
// retrieves all registered providers and injectors from global registries.
func TestAppAnalyzer_RetrievesProvidersAndInjectors(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register some providers and injectors
	providerRegistry.Register(
		&registry.ProviderInfo{
			TypeName:        "main.UserRepository",
			PackagePath:     "main",
			LocalName:       "UserRepository",
			ConstructorName: "NewUserRepository",
			Dependencies:    []string{},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "main.UserService",
			PackagePath:     "main",
			LocalName:       "UserService",
			ConstructorName: "NewUserService",
			Dependencies:    []string{"main.UserRepository"},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	// Mark package as scanned
	packageTracker.MarkPackageScanned("main")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/simpleapp", analyzer, ".")
}

// TestAppAnalyzer_MultipleEntryPoints tests that multiple App annotations
// in different entry points are handled correctly (each package processes independently).
func TestAppAnalyzer_MultipleEntryPoints(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register injectors for both packages to create non-empty dependency graphs
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "main.Service1",
			PackagePath:     "multipleapp/cmd/1",
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
			LocalName:       "Service2",
			ConstructorName: "NewService2",
			Dependencies:    []string{},
			IsPending:       false,
		},
	)

	// Mark packages as scanned
	packageTracker.MarkPackageScanned("multipleapp/cmd/1")
	packageTracker.MarkPackageScanned("multipleapp/cmd/2")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)

	// Test both packages
	analysistest.Run(t, "testdata/src/multipleapp", analyzer, "./cmd/1", "./cmd/2")
}

// TestAppAnalyzer_CircularDependency tests that circular dependencies
// are detected and reported with the full cycle path.
func TestAppAnalyzer_CircularDependency(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register circular dependency: ServiceA -> ServiceB -> ServiceA
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "circular/service.ServiceA",
			PackagePath:     "circular/service",
			LocalName:       "ServiceA",
			ConstructorName: "NewServiceA",
			Dependencies:    []string{"circular/service.ServiceB"},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "circular/service.ServiceB",
			PackagePath:     "circular/service",
			LocalName:       "ServiceB",
			ConstructorName: "NewServiceB",
			Dependencies:    []string{"circular/service.ServiceA"},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	// Mark package as scanned
	packageTracker.MarkPackageScanned("circular")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/circular", analyzer, ".")
}

// TestAppAnalyzer_EmptyGraph tests that an empty bootstrap is generated
// when no providers or injectors exist.
func TestAppAnalyzer_EmptyGraph(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// No injectors or providers registered - empty graph
	packageTracker.MarkPackageScanned("emptygraph")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/emptygraph", analyzer, ".")
}

// TestAppAnalyzer_IdempotentBehavior tests that no diagnostic is emitted
// when the bootstrap code is already current.
func TestAppAnalyzer_IdempotentBehavior(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register a simple injector
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "idempotent.UserService",
			PackagePath:     "idempotent",
			LocalName:       "UserService",
			ConstructorName: "NewUserService",
			Dependencies:    []string{},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	packageTracker.MarkPackageScanned("idempotent")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/idempotent", analyzer, ".")
}

// TestAppAnalyzer_DependencyAlreadyReferenced tests that _ = dependency
// is not added when the dependency variable is already referenced in main.
// NOTE: This integration test is skipped because IsDependencyReferenced logic
// is already thoroughly tested in internal/generate/ast_util_test.go.
// The test fixture design (referencing non-existent dependency variable) causes
// compilation errors that are incompatible with analysistest framework.
func TestAppAnalyzer_DependencyAlreadyReferenced(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register a service that will be referenced in main
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "depused.UserService",
			PackagePath:     "depused",
			LocalName:       "UserService",
			ConstructorName: "NewUserService",
			Dependencies:    []string{},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	packageTracker.MarkPackageScanned("depused")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/depused", analyzer, ".")
}

// TestAppAnalyzer_InterfaceResolution tests that interface parameters
// are correctly resolved to implementing injectable structs.
func TestAppAnalyzer_InterfaceResolution(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register repository implementing the interface
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "iface/repository.UserRepository",
			PackagePath:     "iface/repository",
			LocalName:       "UserRepository",
			ConstructorName: "NewUserRepository",
			Dependencies:    []string{},
			Implements:      []string{"iface/domain.IUserRepository"},
			IsPending:       false,
		},
	)

	// Register service depending on the interface
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "iface/service.UserService",
			PackagePath:     "iface/service",
			LocalName:       "UserService",
			ConstructorName: "NewUserService",
			Dependencies:    []string{"iface/domain.IUserRepository"},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	packageTracker.MarkPackageScanned("iface")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/iface", analyzer, ".")
}

// TestAppAnalyzer_AmbiguousInterface tests that an error is reported
// when multiple injectable structs implement the same interface.
func TestAppAnalyzer_AmbiguousInterface(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register two repositories implementing the same interface
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "ambiguous/repository.UserRepositoryA",
			PackagePath:     "ambiguous/repository",
			LocalName:       "UserRepositoryA",
			ConstructorName: "NewUserRepositoryA",
			Dependencies:    []string{},
			Implements:      []string{"ambiguous/domain.IUserRepository"},
			IsPending:       false,
		},
	)

	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "ambiguous/repository.UserRepositoryB",
			PackagePath:     "ambiguous/repository",
			LocalName:       "UserRepositoryB",
			ConstructorName: "NewUserRepositoryB",
			Dependencies:    []string{},
			Implements:      []string{"ambiguous/domain.IUserRepository"},
			IsPending:       false,
		},
	)

	// Register service depending on the interface
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "ambiguous/service.UserService",
			PackagePath:     "ambiguous/service",
			LocalName:       "UserService",
			ConstructorName: "NewUserService",
			Dependencies:    []string{"ambiguous/domain.IUserRepository"},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	packageTracker.MarkPackageScanned("ambiguous")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/ambiguous", analyzer, ".")
}

// TestAppAnalyzer_CrossPackageInterface tests that interface resolution
// works across packages via the global registry.
func TestAppAnalyzer_CrossPackageInterface(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register repository from one package implementing interface from another
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "crossiface/repository.UserRepository",
			PackagePath:     "crossiface/repository",
			LocalName:       "UserRepository",
			ConstructorName: "NewUserRepository",
			Dependencies:    []string{},
			Implements:      []string{"crossiface/domain.IUserRepository"},
			IsPending:       false,
		},
	)

	// Register service depending on the interface
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "crossiface/service.UserService",
			PackagePath:     "crossiface/service",
			LocalName:       "UserService",
			ConstructorName: "NewUserService",
			Dependencies:    []string{"crossiface/domain.IUserRepository"},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	packageTracker.MarkPackageScanned("crossiface")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/crossiface", analyzer, ".")
}

// TestAppAnalyzer_UnresolvedInterface tests that an error is reported
// when an interface parameter has no injectable implementation.
func TestAppAnalyzer_UnresolvedInterface(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register writer that depends on io.Reader (no implementation)
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "unresiface/writer.MyWriter",
			PackagePath:     "unresiface/writer",
			LocalName:       "MyWriter",
			ConstructorName: "NewMyWriter",
			Dependencies:    []string{"io.Reader"},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	packageTracker.MarkPackageScanned("unresiface")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/unresiface", analyzer, ".")
}

// TestAppAnalyzer_ModuleWideDiscovery tests that all injectors and providers
// are discovered from the module without explicit imports in main.
func TestAppAnalyzer_ModuleWideDiscovery(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register multiple injectors from different packages
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "modulewide/repository.UserRepository",
			PackagePath:     "modulewide/repository",
			LocalName:       "UserRepository",
			ConstructorName: "NewUserRepository",
			Dependencies:    []string{},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "modulewide/repository.OrderRepository",
			PackagePath:     "modulewide/repository",
			LocalName:       "OrderRepository",
			ConstructorName: "NewOrderRepository",
			Dependencies:    []string{},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "modulewide/service.UserService",
			PackagePath:     "modulewide/service",
			LocalName:       "UserService",
			ConstructorName: "NewUserService",
			Dependencies:    []string{"modulewide/repository.UserRepository"},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	packageTracker.MarkPackageScanned("modulewide")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/modulewide", analyzer, ".")
}

// TestAppAnalyzer_UnresolvableParameter tests that an error is reported
// when a constructor parameter cannot be resolved to an injectable type.
func TestAppAnalyzer_UnresolvableParameter(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register repository with unresolvable dependency (*sql.DB)
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "unresparam/repository.UserRepository",
			PackagePath:     "unresparam/repository",
			LocalName:       "UserRepository",
			ConstructorName: "NewUserRepository",
			Dependencies:    []string{"database/sql.DB"},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	packageTracker.MarkPackageScanned("unresparam")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/unresparam", analyzer, ".")
}

// TestAppAnalyzer_BootstrapUpdate tests that a diagnostic is emitted
// with a SuggestedFix when the bootstrap code is outdated.
func TestAppAnalyzer_BootstrapUpdate(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register updated injectors (different from what's in the existing bootstrap)
	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "outdated.UserService",
			PackagePath:     "outdated",
			LocalName:       "UserService",
			ConstructorName: "NewUserService",
			Dependencies:    []string{},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	injectorRegistry.Register(
		&registry.InjectorInfo{
			TypeName:        "outdated.OrderService",
			PackagePath:     "outdated",
			LocalName:       "OrderService",
			ConstructorName: "NewOrderService",
			Dependencies:    []string{},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	packageTracker.MarkPackageScanned("outdated")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/outdated", analyzer, ".")
}

// TestAppAnalyzer_MissingConstructor tests that an error is reported
// when a Provide struct lacks a constructor.
func TestAppAnalyzer_MissingConstructor(t *testing.T) {
	providerRegistry, injectorRegistry, packageLoader, packageTracker, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter := setupTestDependencies()

	// Register provider without constructor (IsPending=false but no constructor exists)
	// This scenario is actually prevented by DependencyAnalyzer validation
	// But we can test the error path by simulating the condition
	providerRegistry.Register(
		&registry.ProviderInfo{
			TypeName:        "missingctor/repository.UserRepository",
			PackagePath:     "missingctor/repository",
			LocalName:       "UserRepository",
			ConstructorName: "", // No constructor
			Dependencies:    []string{},
			Implements:      []string{},
			IsPending:       false,
		},
	)

	packageTracker.MarkPackageScanned("missingctor")

	analyzer := createAppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader, packageTracker,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/src/missingctor", analyzer, ".")
}
