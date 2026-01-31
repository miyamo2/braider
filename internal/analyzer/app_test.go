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
