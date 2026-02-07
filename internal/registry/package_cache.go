package registry

import (
	"sync"

	"golang.org/x/tools/go/packages"
)

// PackageCache caches loaded external packages for Namer validation.
// This cache prevents redundant package loading when validating multiple
// Namer implementations from the same external package.
// Thread-safe for concurrent access.
type PackageCache struct {
	mu    sync.RWMutex
	cache map[string]*packages.Package // key: package import path
}

// NewPackageCache creates a new package cache.
func NewPackageCache() *PackageCache {
	return &PackageCache{
		cache: make(map[string]*packages.Package),
	}
}

// Get retrieves a cached package by import path.
// Returns (package, true) if found, (nil, false) if not cached.
func (pc *PackageCache) Get(pkgPath string) (*packages.Package, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	pkg, ok := pc.cache[pkgPath]
	return pkg, ok
}

// Set stores a package in the cache.
func (pc *PackageCache) Set(pkgPath string, pkg *packages.Package) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.cache[pkgPath] = pkg
}
