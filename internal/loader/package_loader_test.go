package loader_test

import (
	"testing"

	"github.com/miyamo2/braider/internal/loader"
)

func TestPackageLoader_LoadPackage(t *testing.T) {
	l := loader.NewPackageLoader()

	t.Run("loads a standard library package", func(t *testing.T) {
		pkg, err := l.LoadPackage("fmt")
		if err != nil {
			t.Fatalf("LoadPackage(\"fmt\") error = %v", err)
		}
		if pkg == nil {
			t.Fatal("LoadPackage(\"fmt\") returned nil")
		}
		if pkg.Name != "fmt" {
			t.Errorf("pkg.Name = %q, want %q", pkg.Name, "fmt")
		}
	})

	t.Run("cache hit on second load", func(t *testing.T) {
		pkg1, err := l.LoadPackage("fmt")
		if err != nil {
			t.Fatalf("first LoadPackage(\"fmt\") error = %v", err)
		}

		pkg2, err := l.LoadPackage("fmt")
		if err != nil {
			t.Fatalf("second LoadPackage(\"fmt\") error = %v", err)
		}

		// Same pointer from cache
		if pkg1 != pkg2 {
			t.Error("expected same package pointer from cache on second call")
		}
	})

	t.Run("non-existent package returns package with errors", func(t *testing.T) {
		pkg, err := l.LoadPackage("this/package/does/not/exist/at/all")
		if err != nil {
			// packages.Load returned an error directly
			return
		}
		// packages.Load may return a package with Errors populated
		if pkg != nil && len(pkg.Errors) == 0 {
			t.Error("expected package errors for non-existent package, got none")
		}
	})
}
