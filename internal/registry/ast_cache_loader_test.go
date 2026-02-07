package registry

import (
	"testing"
)

func TestASTCacheLoader_LoadPackage(t *testing.T) {
	tests := []struct {
		name        string
		pkgPath     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "load standard library package",
			pkgPath: "fmt",
			wantErr: false,
		},
		{
			name:    "load go types package",
			pkgPath: "go/types",
			wantErr: false,
		},
		{
			name:        "load non-existent package",
			pkgPath:     "github.com/nonexistent/fake/package",
			wantErr:     true,
			errContains: "cannot load package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewPackageCache()
			loader := NewASTCacheLoader(cache)

			pkg, err := loader.LoadPackage(tt.pkgPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadPackage() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("LoadPackage() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("LoadPackage() unexpected error: %v", err)
				return
			}

			if pkg == nil {
				t.Errorf("LoadPackage() returned nil package")
				return
			}

			// Verify package has syntax (AST)
			if len(pkg.Syntax) == 0 {
				t.Errorf("LoadPackage() returned package without syntax/AST")
			}

			// Verify package is cached
			cachedPkg, ok := cache.Get(tt.pkgPath)
			if !ok {
				t.Errorf("LoadPackage() did not cache package")
			}
			if cachedPkg != pkg {
				t.Errorf("LoadPackage() cached different package instance")
			}
		})
	}
}

func TestASTCacheLoader_CachingBehavior(t *testing.T) {
	cache := NewPackageCache()
	loader := NewASTCacheLoader(cache)

	pkgPath := "fmt"

	// First load
	pkg1, err := loader.LoadPackage(pkgPath)
	if err != nil {
		t.Fatalf("First LoadPackage() failed: %v", err)
	}

	// Second load should return cached package
	pkg2, err := loader.LoadPackage(pkgPath)
	if err != nil {
		t.Fatalf("Second LoadPackage() failed: %v", err)
	}

	// Should be the same instance (pointer equality)
	if pkg1 != pkg2 {
		t.Errorf("LoadPackage() did not return cached instance on second call")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
