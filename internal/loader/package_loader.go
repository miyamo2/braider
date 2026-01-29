// Package loader provides utilities for loading Go packages within a module.
package loader

import (
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

// PackageLoader loads all packages in a Go module.
type PackageLoader interface {
	// LoadModulePackages loads all packages in the module.
	// Returns a list of package paths for synchronization with PackageTracker.
	LoadModulePackages(dir string) ([]string, error)

	// FindModuleRoot finds the module root directory from a given path.
	FindModuleRoot(dir string) (string, error)
}

// packageLoader is the default implementation of PackageLoader.
type packageLoader struct{}

// NewPackageLoader creates a new PackageLoader instance.
func NewPackageLoader() PackageLoader {
	return &packageLoader{}
}

// LoadModulePackages loads all packages in the module.
func (l *packageLoader) LoadModulePackages(dir string) ([]string, error) {
	moduleRoot, err := l.FindModuleRoot(dir)
	if err != nil {
		return nil, err
	}

	cfg := &packages.Config{
		Mode: packages.NeedName,
		Dir:  moduleRoot,
	}

	// Load all packages recursively
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
