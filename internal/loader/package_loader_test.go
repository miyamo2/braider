package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/miyamo2/braider/internal/loader"
)

func TestPackageLoader_FindModuleRoot(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "package_loader_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a go.mod file in the temp directory
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "internal", "pkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	tests := []struct {
		name        string
		dir         string
		expectedDir string
		expectError bool
	}{
		{
			name:        "directory is module root",
			dir:         tmpDir,
			expectedDir: tmpDir,
			expectError: false,
		},
		{
			name:        "directory is subdirectory of module",
			dir:         subDir,
			expectedDir: tmpDir,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				l := loader.NewPackageLoader()
				result, err := l.FindModuleRoot(tt.dir)

				if tt.expectError {
					if err == nil {
						t.Error("FindModuleRoot() = nil error, want error")
					}
					return
				}

				if err != nil {
					t.Errorf("FindModuleRoot() error = %v, want nil", err)
					return
				}

				if result != tt.expectedDir {
					t.Errorf("FindModuleRoot() = %q, want %q", result, tt.expectedDir)
				}
			},
		)
	}
}

func TestPackageLoader_FindModuleRoot_NoGoMod(t *testing.T) {
	// Create a temporary directory without go.mod
	tmpDir, err := os.MkdirTemp("", "package_loader_test_no_gomod")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	l := loader.NewPackageLoader()
	_, err = l.FindModuleRoot(tmpDir)

	if !os.IsNotExist(err) {
		t.Errorf("FindModuleRoot() error = %v, want os.ErrNotExist", err)
	}
}

func TestPackageLoader_LoadModulePackages(t *testing.T) {
	// Use the actual braider project for testing
	// This assumes the test is running from within the braider project
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	l := loader.NewPackageLoader()
	moduleRoot, err := l.FindModuleRoot(wd)
	if err != nil {
		t.Skipf("skipping test: not running in a Go module: %v", err)
	}

	paths, err := l.LoadModulePackageNames(moduleRoot)
	if err != nil {
		t.Fatalf("LoadModulePackageNames() error = %v", err)
	}

	if len(paths) == 0 {
		t.Error("LoadModulePackageNames() returned empty slice, want at least one package")
	}

	// Verify that the main package is included
	foundMain := false
	foundInternal := false
	for _, path := range paths {
		if path == "github.com/miyamo2/braider" || path == "github.com/miyamo2/braider/cmd/braider" {
			foundMain = true
		}
		if filepath.Base(filepath.Dir(path)) == "internal" || filepath.Base(path) == "internal" {
			foundInternal = true
		}
	}

	if !foundMain {
		t.Error("LoadModulePackageNames() did not include main package")
	}

	// At least some internal package should be found
	if !foundInternal {
		t.Log("Warning: no internal packages found, this might be expected for some module structures")
	}
}

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

func TestPackageLoader_LoadModulePackages_InvalidDir(t *testing.T) {
	l := loader.NewPackageLoader()
	_, err := l.LoadModulePackageNames("/nonexistent/directory")

	if err == nil {
		t.Error("LoadModulePackageNames() = nil error, want error for invalid directory")
	}
}

func TestPackageLoader_LoadModulePackageNames_Only(t *testing.T) {
	// Verify that LoadModulePackageNames returns correct package paths
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	l := loader.NewPackageLoader()
	moduleRoot, err := l.FindModuleRoot(wd)
	if err != nil {
		t.Skipf("skipping test: not running in a Go module: %v", err)
	}

	paths, err := l.LoadModulePackageNames(moduleRoot)
	if err != nil {
		t.Fatalf("LoadModulePackageNames() error = %v", err)
	}

	if len(paths) == 0 {
		t.Fatal("LoadModulePackageNames() returned empty slice, want at least one package")
	}

	// All paths should be non-empty strings
	for _, p := range paths {
		if p == "" {
			t.Error("LoadModulePackageNames() returned an empty string in paths")
		}
	}

	// Verify known packages are present
	foundLoader := false
	for _, p := range paths {
		if p == "github.com/miyamo2/braider/internal/loader" {
			foundLoader = true
		}
	}
	if !foundLoader {
		t.Error("LoadModulePackageNames() did not include the loader package")
	}

	// Second call should return the same result from cache
	paths2, err := l.LoadModulePackageNames(moduleRoot)
	if err != nil {
		t.Fatalf("LoadModulePackageNames() second call error = %v", err)
	}

	if len(paths2) != len(paths) {
		t.Errorf("LoadModulePackageNames() second call returned %d paths, want %d", len(paths2), len(paths))
	}
}
