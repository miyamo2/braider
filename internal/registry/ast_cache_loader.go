package registry

import (
	"fmt"

	"golang.org/x/tools/go/packages"
)

// ASTCacheLoader loads and caches external package ASTs for validation.
// Provides cached access to external package ASTs via go/packages,
// eliminating redundant I/O for cross-package validation.
type ASTCacheLoader interface {
	// LoadPackage loads the package at the given import path and returns
	// its AST. Uses cached package if previously loaded.
	// Returns error if package cannot be loaded (source unavailable, compilation error).
	LoadPackage(pkgPath string) (*packages.Package, error)
}

// astCacheLoaderImpl implements ASTCacheLoader.
type astCacheLoaderImpl struct {
	cache *PackageCache
}

// NewASTCacheLoader creates a new ASTCacheLoader instance.
func NewASTCacheLoader(cache *PackageCache) ASTCacheLoader {
	return &astCacheLoaderImpl{
		cache: cache,
	}
}

// LoadPackage loads the package at the given import path.
// Checks cache first, then calls packages.Load() on cache miss.
func (l *astCacheLoaderImpl) LoadPackage(pkgPath string) (*packages.Package, error) {
	// Check cache first
	if pkg, ok := l.cache.Get(pkgPath); ok {
		return pkg, nil
	}

	// Load package with syntax and types
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedName | packages.NeedFiles,
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("cannot load package %s: %w", pkgPath, err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("cannot load package %s: no packages found", pkgPath)
	}

	pkg := pkgs[0]

	// Check for package errors
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("cannot load package %s: %v", pkgPath, pkg.Errors[0])
	}

	// Store in cache
	l.cache.Set(pkgPath, pkg)

	return pkg, nil
}
