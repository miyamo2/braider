package loader_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/miyamo2/braider/internal/loader"
	"golang.org/x/tools/go/packages"
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
	if err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.24\n"), 0644); err != nil {
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

func TestPackageLoader_LoadModulePackages_InvalidDir(t *testing.T) {
	l := loader.NewPackageLoader()
	_, err := l.LoadModulePackageNames("/nonexistent/directory")

	if err == nil {
		t.Error("LoadModulePackageNames() = nil error, want error for invalid directory")
	}
}

func TestPackageLoader_CachePersistence(t *testing.T) {
	// Use the actual braider project for testing
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	l := loader.NewPackageLoader()
	moduleRoot, err := l.FindModuleRoot(wd)
	if err != nil {
		t.Skipf("skipping test: not running in a Go module: %v", err)
	}

	// First load
	seq1, err := l.LoadModulePackageAST(moduleRoot)
	if err != nil {
		t.Fatalf("LoadModulePackageAST() first call error = %v", err)
	}

	pkgs1 := slices.Collect(seq1)
	if len(pkgs1) == 0 {
		t.Fatal("LoadModulePackageAST() first call returned empty slice")
	}

	// Second load - should return cached packages
	seq2, err := l.LoadModulePackageAST(moduleRoot)
	if err != nil {
		t.Fatalf("LoadModulePackageAST() second call error = %v", err)
	}

	pkgs2 := slices.Collect(seq2)
	if len(pkgs2) != len(pkgs1) {
		t.Errorf("LoadModulePackageAST() second call returned %d packages, want %d", len(pkgs2), len(pkgs1))
	}

	// Verify that the same packages are returned (by comparing package paths)
	pkgPaths1 := make(map[string]bool)
	for _, pkg := range pkgs1 {
		pkgPaths1[pkg.PkgPath] = true
	}

	for _, pkg := range pkgs2 {
		if !pkgPaths1[pkg.PkgPath] {
			t.Errorf("LoadModulePackageAST() second call returned unexpected package: %s", pkg.PkgPath)
		}
	}
}

func TestPackageLoader_LoadPackageAfterLoadModulePackageAST(t *testing.T) {
	// Use the actual braider project for testing
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	l := loader.NewPackageLoader()
	moduleRoot, err := l.FindModuleRoot(wd)
	if err != nil {
		t.Skipf("skipping test: not running in a Go module: %v", err)
	}

	// Load all module packages
	seq, err := l.LoadModulePackageAST(moduleRoot)
	if err != nil {
		t.Fatalf("LoadModulePackageAST() error = %v", err)
	}

	pkgs := slices.Collect(seq)
	if len(pkgs) == 0 {
		t.Fatal("LoadModulePackageAST() returned empty slice")
	}

	// Select a package to load individually
	testPkgPath := pkgs[0].PkgPath

	// Load package individually - should hit cache
	pkg, err := l.LoadPackage(testPkgPath)
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	if pkg.PkgPath != testPkgPath {
		t.Errorf("LoadPackage() returned package with path %s, want %s", pkg.PkgPath, testPkgPath)
	}

	// Verify it's the same package from cache (same pointer)
	found := false
	for _, cachedPkg := range pkgs {
		if cachedPkg.PkgPath == testPkgPath && cachedPkg == pkg {
			found = true
			break
		}
	}

	if !found {
		t.Error("LoadPackage() did not return cached package from LoadModulePackageAST")
	}
}

func TestPackageLoader_LoadPackageBeforeLoadModulePackageAST(t *testing.T) {
	// Use the actual braider project for testing
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	l := loader.NewPackageLoader()
	moduleRoot, err := l.FindModuleRoot(wd)
	if err != nil {
		t.Skipf("skipping test: not running in a Go module: %v", err)
	}

	// Load an external package first (a dependency outside the braider module)
	externalPkgPath := "golang.org/x/tools/go/packages"
	pkg1, err := l.LoadPackage(externalPkgPath)
	if err != nil {
		t.Skipf("skipping test: external package not available: %v", err)
	}

	if pkg1.PkgPath != externalPkgPath {
		t.Errorf("LoadPackage() returned package with path %s, want %s", pkg1.PkgPath, externalPkgPath)
	}

	// Load all module packages - should NOT include the external package
	seq, err := l.LoadModulePackageAST(moduleRoot)
	if err != nil {
		t.Fatalf("LoadModulePackageAST() error = %v", err)
	}

	pkgs := slices.Collect(seq)

	// Verify the external package is NOT in the results
	for _, pkg := range pkgs {
		if pkg.PkgPath == externalPkgPath {
			t.Errorf(
				"LoadModulePackageAST() should not include external package %s loaded via LoadPackage",
				externalPkgPath,
			)
		}
	}

	// Verify that module packages are present
	if len(pkgs) == 0 {
		t.Fatal("LoadModulePackageAST() returned empty slice")
	}

	// Load the external package again - should still hit cache
	pkg2, err := l.LoadPackage(externalPkgPath)
	if err != nil {
		t.Fatalf("LoadPackage() second call error = %v", err)
	}

	if pkg2.PkgPath != externalPkgPath {
		t.Errorf("LoadPackage() second call returned package with path %s, want %s", pkg2.PkgPath, externalPkgPath)
	}

	// Verify LoadPackage returns the cached pointer
	if pkg1 != pkg2 {
		t.Error("LoadPackage() second call did not return cached package pointer")
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

func TestPackageLoader_LoadModulePackageNames_ThenAST(t *testing.T) {
	// Verify that LoadModulePackageNames → LoadModulePackageAST works correctly:
	// After lightweight load, AST load should perform a full load and populate pkgCache.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	l := loader.NewPackageLoader()
	moduleRoot, err := l.FindModuleRoot(wd)
	if err != nil {
		t.Skipf("skipping test: not running in a Go module: %v", err)
	}

	// Step 1: Lightweight load (NeedName only, does NOT populate pkgCache)
	namePaths, err := l.LoadModulePackageNames(moduleRoot)
	if err != nil {
		t.Fatalf("LoadModulePackageNames() error = %v", err)
	}
	if len(namePaths) == 0 {
		t.Fatal("LoadModulePackageNames() returned empty slice")
	}

	// Step 2: AST load should detect that pkgCache is empty and perform full load
	seq, err := l.LoadModulePackageAST(moduleRoot)
	if err != nil {
		t.Fatalf("LoadModulePackageAST() after LoadModulePackageNames error = %v", err)
	}

	pkgs := slices.Collect(seq)
	if len(pkgs) == 0 {
		t.Fatal("LoadModulePackageAST() returned empty slice after LoadModulePackageNames")
	}

	// All returned packages should have AST data (Syntax field populated)
	for _, pkg := range pkgs {
		if pkg.Types == nil {
			t.Errorf("LoadModulePackageAST() package %s has nil Types, want populated AST data", pkg.PkgPath)
		}
	}

	// Step 3: Subsequent AST load should hit cache (both modulePkgPaths and pkgCache)
	seq2, err := l.LoadModulePackageAST(moduleRoot)
	if err != nil {
		t.Fatalf("LoadModulePackageAST() second call error = %v", err)
	}

	pkgs2 := slices.Collect(seq2)
	if len(pkgs2) != len(pkgs) {
		t.Errorf("LoadModulePackageAST() second call returned %d packages, want %d", len(pkgs2), len(pkgs))
	}

	// Verify cached pointers match
	pkgMap := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		pkgMap[pkg.PkgPath] = pkg
	}
	for _, pkg := range pkgs2 {
		if cached, ok := pkgMap[pkg.PkgPath]; ok {
			if cached != pkg {
				t.Errorf("LoadModulePackageAST() second call returned different pointer for %s", pkg.PkgPath)
			}
		}
	}
}

func TestPackageLoader_LoadModulePackageAST_ThenNames(t *testing.T) {
	// Verify that LoadModulePackageAST → LoadModulePackageNames returns cached data.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	l := loader.NewPackageLoader()
	moduleRoot, err := l.FindModuleRoot(wd)
	if err != nil {
		t.Skipf("skipping test: not running in a Go module: %v", err)
	}

	// Step 1: Full AST load (populates both modulePkgPaths and pkgCache)
	seq, err := l.LoadModulePackageAST(moduleRoot)
	if err != nil {
		t.Fatalf("LoadModulePackageAST() error = %v", err)
	}

	pkgs := slices.Collect(seq)
	if len(pkgs) == 0 {
		t.Fatal("LoadModulePackageAST() returned empty slice")
	}

	// Collect AST package paths for comparison
	astPaths := make(map[string]bool)
	for _, pkg := range pkgs {
		astPaths[pkg.PkgPath] = true
	}

	// Step 2: Lightweight load should return cached modulePkgPaths
	namePaths, err := l.LoadModulePackageNames(moduleRoot)
	if err != nil {
		t.Fatalf("LoadModulePackageNames() after LoadModulePackageAST error = %v", err)
	}

	if len(namePaths) != len(pkgs) {
		t.Errorf("LoadModulePackageNames() returned %d paths, want %d (matching AST load)", len(namePaths), len(pkgs))
	}

	// Verify all name paths match AST paths
	for _, p := range namePaths {
		if !astPaths[p] {
			t.Errorf("LoadModulePackageNames() returned path %s not found in AST load results", p)
		}
	}
}

func TestPackageLoader_CacheWithDifferentDirInSameModule(t *testing.T) {
	// Use the actual braider project for testing
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	l := loader.NewPackageLoader()
	moduleRoot, err := l.FindModuleRoot(wd)
	if err != nil {
		t.Skipf("skipping test: not running in a Go module: %v", err)
	}

	// Load from module root
	seq1, err := l.LoadModulePackageAST(moduleRoot)
	if err != nil {
		t.Fatalf("LoadModulePackageAST() from module root error = %v", err)
	}

	pkgs1 := slices.Collect(seq1)
	if len(pkgs1) == 0 {
		t.Fatal("LoadModulePackageAST() from module root returned empty slice")
	}

	// Load from a subdirectory (should resolve to same module root and hit cache)
	subDir := filepath.Join(moduleRoot, "internal")
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Skipf("skipping test: internal directory does not exist")
	}

	seq2, err := l.LoadModulePackageAST(subDir)
	if err != nil {
		t.Fatalf("LoadModulePackageAST() from subdirectory error = %v", err)
	}

	pkgs2 := slices.Collect(seq2)

	// Should return same number of packages (cache hit)
	if len(pkgs2) != len(pkgs1) {
		t.Errorf(
			"LoadModulePackageAST() from subdirectory returned %d packages, want %d (cache hit)",
			len(pkgs2),
			len(pkgs1),
		)
	}

	// Verify that the same packages are returned (by comparing package paths)
	pkgPaths1 := make(map[string]bool)
	for _, pkg := range pkgs1 {
		pkgPaths1[pkg.PkgPath] = true
	}

	for _, pkg := range pkgs2 {
		if !pkgPaths1[pkg.PkgPath] {
			t.Errorf("LoadModulePackageAST() from subdirectory returned unexpected package: %s", pkg.PkgPath)
		}
	}
}
