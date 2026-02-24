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
	"golang.org/x/tools/go/packages"
)

// mockPackageLoader is a test implementation that returns an empty package list.
type mockPackageLoader struct{}

func (m *mockPackageLoader) LoadModulePackageNames(dir string) ([]string, error) {
	return []string{}, nil
}

func (m *mockPackageLoader) FindModuleRoot(dir string) (string, error) {
	return dir, nil
}

func (m *mockPackageLoader) LoadPackage(pkgPath string) (*packages.Package, error) {
	return nil, nil
}

// setupTestDependencies creates all required dependencies for AppAnalyzer-only tests.
func setupTestDependencies(t *testing.T) (
	*registry.ProviderRegistry,
	*registry.InjectorRegistry,
	detect.AppDetector,
	*graph.DependencyGraphBuilder,
	*graph.TopologicalSorter,
	generate.BootstrapGenerator,
	report.SuggestedFixBuilder,
	report.DiagnosticEmitter,
	detect.AppOptionExtractor,
	graph.ContainerValidator,
	graph.ContainerResolver,
) {
	t.Helper()
	markers, err := detect.ResolveMarkers()
	if err != nil {
		t.Fatal(err)
	}
	providerRegistry := registry.NewProviderRegistry()
	injectorRegistry := registry.NewInjectorRegistry()
	appDetector := detect.NewAppDetector(markers)
	interfaceRegistry := graph.NewInterfaceRegistry()
	graphBuilder := graph.NewDependencyGraphBuilder(interfaceRegistry)
	sorter := graph.NewTopologicalSorter()
	bootstrapGenerator := generate.NewBootstrapGenerator()
	suggestedFixBuilder := report.NewSuggestedFixBuilder()
	diagnosticEmitter := report.NewDiagnosticEmitter()
	appOptionExtractor := detect.NewAppOptionExtractorImpl(markers)
	containerValidator := graph.NewContainerValidatorImpl(interfaceRegistry)
	containerResolver := graph.NewContainerResolverImpl(interfaceRegistry)

	return providerRegistry, injectorRegistry, appDetector,
		graphBuilder, sorter, bootstrapGenerator, suggestedFixBuilder, diagnosticEmitter,
		appOptionExtractor, containerValidator, containerResolver
}

func TestAppAnalyzer_MissingConstructor(t *testing.T) {
	providerRegistry, injectorRegistry, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter, appOptionExtractor, containerValidator, containerResolver := setupTestDependencies(t)

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

	variableReg := registry.NewVariableRegistry()
	runner := NewAppAnalyzeRunner(
		appDetector, injectorRegistry, providerRegistry,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
		variableReg, appOptionExtractor, containerValidator, containerResolver,
	)
	appAnalyzer := (*analysis.Analyzer)(NewAppAnalyzer(runner))

	cfg := phasedchecker.Config{
		Pipeline: phasedchecker.Pipeline{
			Phases: []phasedchecker.Phase{
				{Name: "app", Analyzers: []*analysis.Analyzer{appAnalyzer}},
			},
		},
	}
	checkertest.Run(t, "testdata/bootstrapgen/missingctor", cfg, ".")
	checkertest.RunWithSuggestedFixes(t, "testdata/bootstrapgen/missingctor", cfg, ".")
}

func TestAppAnalyzer_MultipleEntryPoints(t *testing.T) {
	_, injectorRegistry, appDetector, graphBuilder, sorter,
		bootstrapGen, fixBuilder, diagnosticEmitter, appOptionExtractor, containerValidator, containerResolver := setupTestDependencies(t)

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

	variableReg := registry.NewVariableRegistry()
	providerRegistry := registry.NewProviderRegistry()
	runner := NewAppAnalyzeRunner(
		appDetector, injectorRegistry, providerRegistry,
		graphBuilder, sorter, bootstrapGen, fixBuilder, diagnosticEmitter,
		variableReg, appOptionExtractor, containerValidator, containerResolver,
	)
	appAnalyzer := (*analysis.Analyzer)(NewAppAnalyzer(runner))

	cfg := phasedchecker.Config{
		Pipeline: phasedchecker.Pipeline{
			Phases: []phasedchecker.Phase{
				{Name: "app", Analyzers: []*analysis.Analyzer{appAnalyzer}},
			},
		},
	}
	checkertest.Run(t, "testdata/bootstrapgen/multipleapp", cfg, "./...")
}

// TestAppAnalyzer_CorrelationErrorNonFatal tests that duplicate (TypeName, Name) registration
// returns an error from Registry.Register() but does NOT abort the pipeline,
// so AppAnalyzer continues to generate bootstrap code.
func TestAppAnalyzer_CorrelationErrorNonFatal(t *testing.T) {
	injectorReg := registry.NewInjectorRegistry()

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
}
