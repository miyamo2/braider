// Package loader provides utilities for loading Go packages within a module.
package loader

import (
	"os"
	"sync"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
	"golang.org/x/tools/go/packages"
)

// PackageLoader loads all packages in a Go module.
type PackageLoader interface {
	// LoadPackage loads a single package by its import path.
	// Returns the package with full AST for analysis.
	LoadPackage(pkgPath string) (*packages.Package, error)
}

var _ = annotation.Provide[provide.Default](NewPackageLoader)

// packageLoader is the default implementation of PackageLoader.
type packageLoader struct {
	pkgCache sync.Map // key: string (pkgPath), value: *packages.Package
}

// NewPackageLoader creates a new PackageLoader instance.
func NewPackageLoader() PackageLoader {
	return &packageLoader{}
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
