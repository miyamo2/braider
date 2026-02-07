package registry

import (
	"sync"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestPackageCache_GetAndSet(t *testing.T) {
	t.Run("Get returns nil and false for non-existent package", func(t *testing.T) {
		cache := NewPackageCache()

		pkg, ok := cache.Get("example.com/nonexistent")
		if pkg != nil {
			t.Errorf("Get(nonexistent) returned package, want nil")
		}
		if ok {
			t.Errorf("Get(nonexistent) returned ok=true, want false")
		}
	})

	t.Run("Set and Get stores and retrieves package", func(t *testing.T) {
		cache := NewPackageCache()

		testPkg := &packages.Package{
			ID:   "example.com/test",
			Name: "test",
		}

		cache.Set("example.com/test", testPkg)

		got, ok := cache.Get("example.com/test")
		if !ok {
			t.Fatal("Get() returned ok=false, want true")
		}
		if got == nil {
			t.Fatal("Get() returned nil, want package")
		}
		if got.ID != testPkg.ID {
			t.Errorf("Get().ID = %q, want %q", got.ID, testPkg.ID)
		}
		if got.Name != testPkg.Name {
			t.Errorf("Get().Name = %q, want %q", got.Name, testPkg.Name)
		}
	})

	t.Run("Set overwrites existing package", func(t *testing.T) {
		cache := NewPackageCache()

		pkg1 := &packages.Package{
			ID:   "example.com/pkg",
			Name: "pkg1",
		}
		pkg2 := &packages.Package{
			ID:   "example.com/pkg",
			Name: "pkg2",
		}

		cache.Set("example.com/pkg", pkg1)
		cache.Set("example.com/pkg", pkg2)

		got, ok := cache.Get("example.com/pkg")
		if !ok {
			t.Fatal("Get() returned ok=false, want true")
		}
		if got.Name != "pkg2" {
			t.Errorf("Get().Name = %q, want %q", got.Name, "pkg2")
		}
	})

	t.Run("multiple packages can be stored independently", func(t *testing.T) {
		cache := NewPackageCache()

		pkg1 := &packages.Package{ID: "example.com/pkg1", Name: "pkg1"}
		pkg2 := &packages.Package{ID: "example.com/pkg2", Name: "pkg2"}
		pkg3 := &packages.Package{ID: "example.com/pkg3", Name: "pkg3"}

		cache.Set("example.com/pkg1", pkg1)
		cache.Set("example.com/pkg2", pkg2)
		cache.Set("example.com/pkg3", pkg3)

		got1, ok1 := cache.Get("example.com/pkg1")
		got2, ok2 := cache.Get("example.com/pkg2")
		got3, ok3 := cache.Get("example.com/pkg3")

		if !ok1 || !ok2 || !ok3 {
			t.Fatal("Get() returned ok=false for one or more packages")
		}
		if got1.Name != "pkg1" {
			t.Errorf("pkg1.Name = %q, want %q", got1.Name, "pkg1")
		}
		if got2.Name != "pkg2" {
			t.Errorf("pkg2.Name = %q, want %q", got2.Name, "pkg2")
		}
		if got3.Name != "pkg3" {
			t.Errorf("pkg3.Name = %q, want %q", got3.Name, "pkg3")
		}
	})
}

func TestPackageCache_ThreadSafety(t *testing.T) {
	cache := NewPackageCache()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Concurrent writes
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for j := range numOperations {
				pkgPath := "example.com/pkg" + string(rune('A'+id%26))
				pkg := &packages.Package{
					ID:   pkgPath,
					Name: "pkg" + string(rune('A'+id%26)) + string(rune('0'+j%10)),
				}
				cache.Set(pkgPath, pkg)
			}
		}(i)
	}

	// Concurrent reads
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for range numOperations {
				pkgPath := "example.com/pkg" + string(rune('A'+id%26))
				_, _ = cache.Get(pkgPath)
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is in consistent state
	pkg, ok := cache.Get("example.com/pkgA")
	if ok && pkg == nil {
		t.Error("Cache returned ok=true but nil package after concurrent operations")
	}
}

func TestPackageCache_NilPackage(t *testing.T) {
	t.Run("Set and Get handles nil package", func(t *testing.T) {
		cache := NewPackageCache()

		cache.Set("example.com/nil", nil)

		got, ok := cache.Get("example.com/nil")
		if !ok {
			t.Error("Get() returned ok=false for nil package, want true")
		}
		if got != nil {
			t.Errorf("Get() returned non-nil package, want nil")
		}
	})
}
