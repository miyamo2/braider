// Package loader provides utilities for loading Go packages within a module.
package loader

import (
	"iter"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/tools/go/packages"
)

// PackageLoader loads all packages in a Go module.
type PackageLoader interface {
	// LoadModulePackageNames loads all packages in the module.
	// Returns a list of package paths for synchronization with PackageTracker.
	LoadModulePackageNames(dir string) ([]string, error)

	// LoadModulePackageAST loads all packages in the module with full AST.
	// Returns packages suitable for AST analysis and validation.
	LoadModulePackageAST(dir string) (iter.Seq[*packages.Package], error)

	// LoadPackage loads a single package by its import path.
	// Returns the package with full AST for analysis.
	LoadPackage(pkgPath string) (*packages.Package, error)

	// FindModuleRoot finds the module root directory from a given path.
	FindModuleRoot(dir string) (string, error)
}

// packageLoader is the default implementation of PackageLoader.
type packageLoader struct {
	mu             sync.Mutex
	pkgCache       map[string]*packages.Package // all packages (module + external)
	modulePkgPaths map[string][]string          // moduleRoot → package paths belonging to that module
}

// NewPackageLoader creates a new PackageLoader instance.
func NewPackageLoader() PackageLoader {
	return &packageLoader{
		pkgCache:       make(map[string]*packages.Package),
		modulePkgPaths: make(map[string][]string),
	}
}

// LoadModulePackageNames loads all packages in the module.
func (l *packageLoader) LoadModulePackageNames(dir string) ([]string, error) {
	pkgs, err := l.LoadModulePackageAST(dir)
	if err != nil {
		return nil, err
	}

	var paths []string
	for pkg := range pkgs {
		if pkg.PkgPath != "" {
			paths = append(paths, pkg.PkgPath)
		}
	}

	return paths, nil
}

// LoadModulePackageAST loads all packages in the module with full AST.
// Returns only packages belonging to the module, not external packages loaded via LoadPackage.
func (l *packageLoader) LoadModulePackageAST(dir string) (iter.Seq[*packages.Package], error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	moduleRoot, err := l.FindModuleRoot(dir)
	if err != nil {
		return nil, err
	}

	if paths, ok := l.modulePkgPaths[moduleRoot]; ok {
		return l.packagesByPaths(paths), nil
	}

	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedName | packages.NeedFiles,
		Dir:  moduleRoot,
	}

	// Load all packages recursively with full AST
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, pkg := range pkgs {
		if pkg.PkgPath != "" {
			l.pkgCache[pkg.PkgPath] = pkg
			paths = append(paths, pkg.PkgPath)
		}
	}

	l.modulePkgPaths[moduleRoot] = paths

	return l.packagesByPaths(paths), nil
}

// packagesByPaths returns cached packages matching the given paths.
func (l *packageLoader) packagesByPaths(paths []string) iter.Seq[*packages.Package] {
	return func(yield func(*packages.Package) bool) {
		for _, path := range paths {
			if pkg, ok := l.pkgCache[path]; ok {
				if !yield(pkg) {
					return
				}
			}
		}
	}
}

// LoadPackage loads a single package by its import path.
// Uses the cache to avoid redundant loading.
func (l *packageLoader) LoadPackage(pkgPath string) (*packages.Package, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if pkg, ok := l.pkgCache[pkgPath]; ok {
		return pkg, nil
	}

	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedName | packages.NeedFiles,
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return nil, os.ErrNotExist
	}

	l.pkgCache[pkgPath] = pkgs[0]

	return pkgs[0], nil
}

// FindModuleRoot finds the module root directory from a given path.
func (l *packageLoader) FindModuleRoot(dir string) (string, error) {
	current := dir

	for {
		goModPath := filepath.Join(current, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root without finding go.mod
			return "", os.ErrNotExist
		}
		current = parent
	}
}
