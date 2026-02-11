package analyzer

import (
	"context"
	"fmt"
	"iter"
	"strings"
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
	context.Context,
	context.CancelCauseFunc,
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
	bootstrapCtx, bootstrapCancel := context.WithCancelCause(context.Background())
	appDetector := detect.NewAppDetector()
	graphBuilder := graph.NewDependencyGraphBuilder()
	sorter := graph.NewTopologicalSorter()
	bootstrapGenerator := generate.NewBootstrapGenerator()
	suggestedFixBuilder := report.NewSuggestedFixBuilder()
	diagnosticEmitter := report.NewDiagnosticEmitter()

	return providerRegistry, injectorRegistry, packageTracker, bootstrapCtx, bootstrapCancel, appDetector,
		graphBuilder, sorter, bootstrapGenerator, suggestedFixBuilder, diagnosticEmitter
}

func TestAppAnalyzer_ContextCancellation(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, bootstrapCtx, bootstrapCancel, appDetector, graphBuilder, sorter,
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
	bootstrapCancel(fmt.Errorf("simulated validation error"))

	packageLoader := &mockPackageLoader{}
	analyzer := AppAnalyzer(
		appDetector, injectorRegistry, providerRegistry, packageLoader,
		packageTracker, bootstrapCtx,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)

	// Should not emit any diagnostics when context is cancelled
	analysistest.Run(t, "testdata/bootstrapgen/contextcancel", analyzer, ".")
}

func TestAppAnalyzer_MissingConstructor(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, bootstrapCtx, _, appDetector, graphBuilder, sorter,
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
		packageTracker, bootstrapCtx,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/bootstrapgen/missingctor", analyzer, ".")
	analysistest.RunWithSuggestedFixes(t, "testdata/bootstrapgen/missingctor", analyzer, ".")
}

func TestAppAnalyzer_MultipleEntryPoints(t *testing.T) {
	providerRegistry, injectorRegistry, packageTracker, bootstrapCtx, _, appDetector, graphBuilder, sorter,
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
		packageTracker, bootstrapCtx,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
	)
	analysistest.Run(t, "testdata/bootstrapgen/multipleapp", analyzer, "./...")
}

// TestAppAnalyzer_CorrelationErrorNonFatal tests that duplicate (TypeName, Name) registration
// returns an error from Registry.Register() but does NOT cancel the ValidationContext,
// so AppAnalyzer continues to generate bootstrap code.
// This scenario cannot be triggered via analysistest because Go TypeNames are unique per package.
func TestAppAnalyzer_CorrelationErrorNonFatal(t *testing.T) {
	injectorReg := registry.NewInjectorRegistry()
	bootstrapCtx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	// First registration succeeds
	err := injectorReg.Register(
		&registry.InjectorInfo{
			TypeName:        "example.com/repo.Repository",
			PackagePath:     "example.com/repo",
			PackageName:     "repo",
			LocalName:       "Repository",
			ConstructorName: "NewRepository",
			Dependencies:    []string{},
			Name:            "primary",
			OptionMetadata:  detect.OptionMetadata{Name: "primary"},
		},
	)
	if err != nil {
		t.Fatalf("First registration should succeed, got: %v", err)
	}

	// Duplicate registration returns error
	err = injectorReg.Register(
		&registry.InjectorInfo{
			TypeName:        "example.com/repo.Repository",
			PackagePath:     "example.com/repo2",
			PackageName:     "repo2",
			LocalName:       "Repository",
			ConstructorName: "NewRepository",
			Dependencies:    []string{},
			Name:            "primary",
			OptionMetadata:  detect.OptionMetadata{Name: "primary"},
		},
	)
	if err == nil {
		t.Fatal("Duplicate registration should return error")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("Error should mention 'duplicate', got: %v", err)
	}

	// Context should NOT be cancelled (correlation errors are non-fatal)
	if bootstrapCtx.Err() != nil {
		t.Error("ValidationContext should NOT be cancelled for correlation errors")
	}
}
