// Package loader provides utilities for loading Go packages within a module.
package loader

import (
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
	LoadModulePackageAST(dir string) ([]*packages.Package, error)

	// FindModuleRoot finds the module root directory from a given path.
	FindModuleRoot(dir string) (string, error)
}

// packageLoader is the default implementation of PackageLoader.
type packageLoader struct {
	once sync.Once
	mu   sync.Mutex
	pkgs []*packages.Package
}

// NewPackageLoader creates a new PackageLoader instance.
func NewPackageLoader() PackageLoader {
	return &packageLoader{}
}

// LoadModulePackageNames loads all packages in the module.
func (l *packageLoader) LoadModulePackageNames(dir string) ([]string, error) {
	pkgs, err := l.LoadModulePackageAST(dir)
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
// Returns all packages without filtering errors - caller should handle errors.
func (l *packageLoader) LoadModulePackageAST(dir string) ([]*packages.Package, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var err error
	l.once.Do(
		func() {
			var moduleRoot string
			moduleRoot, err = l.FindModuleRoot(dir)
			if err != nil {
				return
			}

			cfg := &packages.Config{
				Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedName | packages.NeedFiles,
				Dir:  moduleRoot,
			}

			// Load all packages recursively with full AST
			l.pkgs, err = packages.Load(cfg, "./...")
		},
	)
	return l.pkgs, err
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
