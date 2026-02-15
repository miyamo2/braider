// Package loader provides utilities for loading Go packages within a module.
package loader

import (
	"iter"
	"os"
	"path/filepath"
	"sync"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
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

var _ = annotation.Provide[provide.Default](NewPackageLoader)

// packageLoader is the default implementation of PackageLoader.
type packageLoader struct {
	pkgCache       sync.Map // key: string (pkgPath), value: *packages.Package
	modulePkgPaths sync.Map // key: string (moduleRoot), value: []string
}

// NewPackageLoader creates a new PackageLoader instance.
func NewPackageLoader() PackageLoader {
	return &packageLoader{}
}

// LoadModulePackageNames loads all packages in the module.
func (l *packageLoader) LoadModulePackageNames(dir string) ([]string, error) {
	moduleRoot, err := l.FindModuleRoot(dir)
	if err != nil {
		return nil, err
	}

	// Return cached modulePkgPaths if available
	if v, ok := l.modulePkgPaths.Load(moduleRoot); ok {
		return v.([]string), nil
	}

	// Lightweight load with NeedName only
	cfg := &packages.Config{
		Mode: packages.NeedName,
		Dir:  moduleRoot,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, pkg := range pkgs {
		if pkg.PkgPath != "" {
			paths = append(paths, pkg.PkgPath)
		}
	}

	return paths, nil
}

// LoadModulePackageAST loads all packages in the module with full AST.
// Returns only packages belonging to the module, not external packages loaded via LoadPackage.
func (l *packageLoader) LoadModulePackageAST(dir string) (iter.Seq[*packages.Package], error) {
	moduleRoot, err := l.FindModuleRoot(dir)
	if err != nil {
		return nil, err
	}

	if v, ok := l.modulePkgPaths.Load(moduleRoot); ok {
		paths := v.([]string)
		// Check if pkgCache has data (pkgCache may be empty after lightweight load)
		if len(paths) > 0 {
			if _, cached := l.pkgCache.Load(paths[0]); cached {
				return l.packagesByPaths(paths), nil
			}
		} else {
			// Return empty iter.Seq for empty modules (no packages)
			return l.packagesByPaths(paths), nil
		}
		// Fall through to full load if pkgCache is empty
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
			l.pkgCache.Store(pkg.PkgPath, pkg)
			paths = append(paths, pkg.PkgPath)
		}
	}

	l.modulePkgPaths.Store(moduleRoot, paths)

	return l.packagesByPaths(paths), nil
}

// packagesByPaths returns cached packages matching the given paths.
func (l *packageLoader) packagesByPaths(paths []string) iter.Seq[*packages.Package] {
	return func(yield func(*packages.Package) bool) {
		for _, path := range paths {
			if v, ok := l.pkgCache.Load(path); ok {
				if !yield(v.(*packages.Package)) {
					return
				}
			}
		}
	}
}

// LoadPackage loads a single package by its import path.
// Uses the cache to avoid redundant loading.
func (l *packageLoader) LoadPackage(pkgPath string) (*packages.Package, error) {
	if v, ok := l.pkgCache.Load(pkgPath); ok {
		return v.(*packages.Package), nil
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

	l.pkgCache.Store(pkgPath, pkgs[0])

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
