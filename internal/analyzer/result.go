package analyzer

import "github.com/miyamo2/braider/internal/registry"

// DependencyResult is the per-package result returned by DependencyAnalyzer.Run().
// It collects all DI registration info discovered in a single package, to be
// aggregated by Aggregator.AfterDependencyPhase after all packages are analyzed.
type DependencyResult struct {
	Providers []*registry.ProviderInfo
	Injectors []*registry.InjectorInfo
	Variables []*registry.VariableInfo
}
