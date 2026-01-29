package analyzer

import (
	"testing"

	"github.com/miyamo2/braider/internal/registry"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestDependencyAnalyzer(t *testing.T) {
	// Clear registries before test to ensure isolation
	registry.GlobalProviderRegistry.Clear()
	registry.GlobalInjectorRegistry.Clear()
	registry.GlobalPackageTracker.Clear()

	analysistest.Run(t, "testdata/dependency/basic", DependencyAnalyzer, ".")

	// Verify providers were registered
	providers := registry.GlobalProviderRegistry.GetAll()
	if len(providers) == 0 {
		t.Error("expected providers to be registered, got none")
	}

	// Verify injectors were registered
	injectors := registry.GlobalInjectorRegistry.GetAll()
	if len(injectors) == 0 {
		t.Error("expected injectors to be registered, got none")
	}

	// Verify package was marked as scanned
	if !registry.GlobalPackageTracker.IsPackageScanned("example.com/dependency/basic") {
		t.Error("expected package to be marked as scanned")
	}
}

func TestDependencyAnalyzer_SuggestedFixes(t *testing.T) {
	// Run with suggested fixes to verify code generation
	// Now using DependencyAnalyzer which includes constructor generation (Phase 1)
	analysistest.RunWithSuggestedFixes(t, "testdata/constructorgen", DependencyAnalyzer, ".")
}

func TestDependencyAnalyzer_MissingProvideConstructor(t *testing.T) {
	// Clear registries before test
	registry.GlobalProviderRegistry.Clear()
	registry.GlobalInjectorRegistry.Clear()
	registry.GlobalPackageTracker.Clear()

	analysistest.Run(t, "testdata/dependency/missing_constructor", DependencyAnalyzer, ".")

	// Provider should not be registered when constructor is missing
	providers := registry.GlobalProviderRegistry.GetAll()
	if len(providers) != 0 {
		t.Errorf("expected no providers to be registered when constructor missing, got %d", len(providers))
	}
}

func TestDependencyAnalyzer_CrossPackage(t *testing.T) {
	// Clear registries before test
	registry.GlobalProviderRegistry.Clear()
	registry.GlobalInjectorRegistry.Clear()
	registry.GlobalPackageTracker.Clear()

	// Analyze multiple packages
	analysistest.Run(t, "testdata/dependency/cross_package", DependencyAnalyzer, "./...")

	// Verify both packages registered their structs
	providers := registry.GlobalProviderRegistry.GetAll()
	injectors := registry.GlobalInjectorRegistry.GetAll()

	totalStructs := len(providers) + len(injectors)
	if totalStructs < 2 {
		t.Errorf("expected at least 2 structs from cross-package test, got %d", totalStructs)
	}

	// Verify both packages were marked as scanned
	if !registry.GlobalPackageTracker.IsPackageScanned("example.com/dependency/cross_package/repo") {
		t.Error("expected repo package to be marked as scanned")
	}
	if !registry.GlobalPackageTracker.IsPackageScanned("example.com/dependency/cross_package/service") {
		t.Error("expected service package to be marked as scanned")
	}
}

func TestDependencyAnalyzer_InterfaceImplementation(t *testing.T) {
	// Clear registries before test
	registry.GlobalProviderRegistry.Clear()
	registry.GlobalInjectorRegistry.Clear()
	registry.GlobalPackageTracker.Clear()

	analysistest.Run(t, "testdata/dependency/abstrct", DependencyAnalyzer, "./...")

	// Verify Implements field is populated
	providers := registry.GlobalProviderRegistry.GetAll()
	injectors := registry.GlobalInjectorRegistry.GetAll()

	hasImplements := false
	for _, p := range providers {
		if len(p.Implements) > 0 {
			hasImplements = true
			break
		}
	}
	for _, i := range injectors {
		if len(i.Implements) > 0 {
			hasImplements = true
			break
		}
	}

	if !hasImplements {
		t.Error("expected at least one struct to have Implements populated")
	}
}
