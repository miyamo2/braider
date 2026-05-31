package registry

import (
	"sort"
	"sync"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

var _ = annotation.Provide[provide.Default](NewEntryPointRegistry)

// EntryPointRegistry tracks main packages and explicit annotation.App declarations
// observed across the analyzed scope. It is populated by Aggregator.AfterDependencyPhase
// (which sees per-package DependencyResult flags) and consumed by AppAnalyzeRunner.Run
// to decide whether an App annotation can be inferred for a single-main-package project,
// whether inference must be suppressed because an explicit App exists somewhere, or
// whether an ambiguous-entry-point diagnostic must be emitted because multiple main
// packages exist with no explicit App.
type EntryPointRegistry struct {
	mu                  sync.RWMutex
	mainPackagePaths    map[string]struct{}
	explicitAppPkgPaths map[string]struct{}
}

// NewEntryPointRegistry creates a new empty EntryPointRegistry.
func NewEntryPointRegistry() *EntryPointRegistry {
	return &EntryPointRegistry{
		mainPackagePaths:    make(map[string]struct{}),
		explicitAppPkgPaths: make(map[string]struct{}),
	}
}

// RegisterMainPackage records pkgPath as a main package (package name == "main"
// with a top-level "func main"). Registration is idempotent.
func (r *EntryPointRegistry) RegisterMainPackage(pkgPath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mainPackagePaths[pkgPath] = struct{}{}
}

// RegisterExplicitApp records that pkgPath contains at least one explicit
// annotation.App declaration. Registration is idempotent.
func (r *EntryPointRegistry) RegisterExplicitApp(pkgPath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.explicitAppPkgPaths[pkgPath] = struct{}{}
}

// MainPackagePaths returns a lexicographically sorted snapshot of the registered
// main package paths. The returned slice is safe for the caller to retain.
func (r *EntryPointRegistry) MainPackagePaths() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	paths := make([]string, 0, len(r.mainPackagePaths))
	for p := range r.mainPackagePaths {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// HasExplicitApp reports whether at least one explicit annotation.App declaration
// was registered anywhere in scope.
func (r *EntryPointRegistry) HasExplicitApp() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.explicitAppPkgPaths) > 0
}
