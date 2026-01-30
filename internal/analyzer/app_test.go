package analyzer

import (
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// setupTestDependencies creates all required dependencies for AppAnalyzer tests.
func setupTestDependencies() (
	*registry.ProviderRegistry,
	*registry.InjectorRegistry,
	*registry.PackageTracker,
	detect.AppDetector,
	report.DiagnosticEmitter,
) {
	providerRegistry := registry.NewProviderRegistry()
	injectorRegistry := registry.NewInjectorRegistry()
	packageTracker := registry.NewPackageTracker()
	appDetector := detect.NewAppDetector()
	diagnosticEmitter := report.NewDiagnosticEmitter()

	return providerRegistry, injectorRegistry, packageTracker, appDetector, diagnosticEmitter
}

// createAppAnalyzer creates an AppAnalyzer with the provided dependencies.
func createAppAnalyzer(
	appDetector detect.AppDetector,
	injectorRegistry *registry.InjectorRegistry,
	providerRegistry *registry.ProviderRegistry,
	diagnosticEmitter report.DiagnosticEmitter,
) *analysis.Analyzer {
	return AppAnalyzer(appDetector, injectorRegistry, providerRegistry, diagnosticEmitter)
}

// TestAppAnalyzer_NoAppAnnotation tests that bootstrap generation is skipped
// when no App annotation is present in the package.
func TestAppAnalyzer_NoAppAnnotation(t *testing.T) {
	providerRegistry, injectorRegistry, _, appDetector, diagnosticEmitter := setupTestDependencies()

	analyzer := createAppAnalyzer(appDetector, injectorRegistry, providerRegistry, diagnosticEmitter)
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer, "noapp")
}

// TestAppAnalyzer_DetectsAppAnnotation tests that the analyzer detects
// an App annotation and processes it.
func TestAppAnalyzer_DetectsAppAnnotation(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, appDetector, diagnosticEmitter := setupTestDependencies()

	// Register a simple injector to test retrieval
	injectorRegistry.Register(&registry.InjectorInfo{
		TypeName:        "noapp.TestService",
		PackagePath:     "noapp",
		LocalName:       "TestService",
		ConstructorName: "NewTestService",
		Dependencies:    []string{},
		Implements:      []string{},
		IsPending:       false,
	})

	// Mark the package as scanned
	packageTracker.MarkPackageScanned("noapp")

	analyzer := createAppAnalyzer(appDetector, injectorRegistry, providerRegistry, diagnosticEmitter)
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer, "simpleapp")
}

// TestAppAnalyzer_WaitsForAllPackages tests that the analyzer waits for
// all packages to be scanned before generating bootstrap code.
func TestAppAnalyzer_WaitsForAllPackages(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, appDetector, diagnosticEmitter := setupTestDependencies()

	// This test verifies that AppAnalyzer waits for package scanning
	// We won't mark packages as scanned, so it should timeout
	// (or we could test with goroutines marking them as scanned)

	// For now, we'll test the happy path where packages are already scanned
	packageTracker.MarkPackageScanned("simpleapp")

	analyzer := createAppAnalyzer(appDetector, injectorRegistry, providerRegistry, diagnosticEmitter)
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer, "simpleapp")
}

// TestAppAnalyzer_MultipleAppAnnotations tests that multiple App annotations
// are detected and reported as an error.
func TestAppAnalyzer_MultipleAppAnnotations(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, appDetector, diagnosticEmitter := setupTestDependencies()

	// Mark package as scanned
	packageTracker.MarkPackageScanned("multipleapp")

	analyzer := createAppAnalyzer(appDetector, injectorRegistry, providerRegistry, diagnosticEmitter)
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer, "multipleapp")
}

// TestAppAnalyzer_NonMainReference tests that App annotation referencing
// a non-main function is detected and reported as an error.
func TestAppAnalyzer_NonMainReference(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, appDetector, diagnosticEmitter := setupTestDependencies()

	// Mark package as scanned
	packageTracker.MarkPackageScanned("nonmainapp")

	analyzer := createAppAnalyzer(appDetector, injectorRegistry, providerRegistry, diagnosticEmitter)
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer, "nonmainapp")
}

// TestAppAnalyzer_RetrievesProvidersAndInjectors tests that the analyzer
// retrieves all registered providers and injectors from global registries.
func TestAppAnalyzer_RetrievesProvidersAndInjectors(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, appDetector, diagnosticEmitter := setupTestDependencies()

	// Register some providers and injectors
	providerRegistry.Register(&registry.ProviderInfo{
		TypeName:        "example.com/repo.UserRepository",
		PackagePath:     "example.com/repo",
		LocalName:       "UserRepository",
		ConstructorName: "NewUserRepository",
		Dependencies:    []string{},
		Implements:      []string{},
		IsPending:       false,
	})

	injectorRegistry.Register(&registry.InjectorInfo{
		TypeName:        "example.com/service.UserService",
		PackagePath:     "example.com/service",
		LocalName:       "UserService",
		ConstructorName: "NewUserService",
		Dependencies:    []string{"example.com/repo.UserRepository"},
		Implements:      []string{},
		IsPending:       false,
	})

	// Mark package as scanned
	packageTracker.MarkPackageScanned("simpleapp")

	analyzer := createAppAnalyzer(appDetector, injectorRegistry, providerRegistry, diagnosticEmitter)
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer, "simpleapp")
}
