package analyzer

import "github.com/miyamo2/braider/internal/registry"

// DependencyResult is the per-package result returned by DependencyAnalyzer.Run().
// It collects all DI registration info discovered in a single package, to be
// aggregated by Aggregator.AfterDependencyPhase after all packages are analyzed.
//
// The PackagePath / IsMainPackage / HasExplicitApp fields feed entry-point
// resolution: Aggregator pushes them into EntryPointRegistry so AppAnalyzeRunner
// can decide whether to infer an annotation.App for a single-main-package project,
// suppress inference because an explicit App exists somewhere, or emit an
// ambiguous-entry-point diagnostic when multiple main packages are in scope.
type DependencyResult struct {
	Providers []*registry.ProviderInfo
	Injectors []*registry.InjectorInfo
	Variables []*registry.VariableInfo

	// PackagePath is the import path of the analyzed package (pass.Pkg.Path()).
	PackagePath string
	// IsMainPackage is true iff the package's declared name is "main" AND a
	// top-level "func main" declaration is present in the package.
	IsMainPackage bool
	// HasExplicitApp is true iff the package contains at least one explicit
	// annotation.App declaration discovered by detect.AppDetector.
	HasExplicitApp bool
}
