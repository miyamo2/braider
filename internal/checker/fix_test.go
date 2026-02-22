package checker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	gochecker "golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

func loadTestPackages(t *testing.T, dir string) []*packages.Package {
	t.Helper()
	pkgs, err := packages.Load(
		&packages.Config{Mode: packages.LoadAllSyntax, Dir: dir},
		"./...",
	)
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}
	return pkgs
}

func TestApplyFixes_NoFixes(t *testing.T) {
	dir := setupTestModule(t, map[string]string{
		"main.go": minimalMain,
	})

	pkgs := loadTestPackages(t, dir)
	graph, err := gochecker.Analyze([]*analysis.Analyzer{noopAnalyzer}, pkgs, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if err := ApplyFixes(graph, false, false); err != nil {
		t.Fatalf("ApplyFixes: %v", err)
	}
}

func TestApplyFixes_WithFixes(t *testing.T) {
	dir := setupTestModule(t, map[string]string{
		"main.go": `package main

var bar = 1

func main() {
	_ = bar
}
`,
	})

	pkgs := loadTestPackages(t, dir)
	graph, err := gochecker.Analyze([]*analysis.Analyzer{renameAnalyzer}, pkgs, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if err := ApplyFixes(graph, false, false); err != nil {
		t.Fatalf("ApplyFixes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "bar") {
		t.Errorf("file still contains 'bar' after fix:\n%s", content)
	}
	if !strings.Contains(string(content), "baz") {
		t.Errorf("file does not contain 'baz' after fix:\n%s", content)
	}
}

func TestApplyFixes_PrintDiffMode(t *testing.T) {
	const src = `package main

var bar = 1

func main() {
	_ = bar
}
`
	dir := setupTestModule(t, map[string]string{
		"main.go": src,
	})

	pkgs := loadTestPackages(t, dir)
	graph, err := gochecker.Analyze([]*analysis.Analyzer{renameAnalyzer}, pkgs, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if err := ApplyFixes(graph, true, false); err != nil {
		t.Fatalf("ApplyFixes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != src {
		t.Errorf("file was modified in print-diff mode:\ngot:\n%s\nwant:\n%s", content, src)
	}
}

func TestApplyFixes_PreservesPermissions(t *testing.T) {
	dir := setupTestModule(t, map[string]string{
		"main.go": `package main

var bar = 1

func main() {
	_ = bar
}
`,
	})

	mainPath := filepath.Join(dir, "main.go")
	if err := os.Chmod(mainPath, 0755); err != nil {
		t.Fatal(err)
	}

	pkgs := loadTestPackages(t, dir)
	graph, err := gochecker.Analyze([]*analysis.Analyzer{renameAnalyzer}, pkgs, nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if err := ApplyFixes(graph, false, false); err != nil {
		t.Fatalf("ApplyFixes: %v", err)
	}

	fi, err := os.Stat(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0755 {
		t.Errorf("file permissions = %o, want 0755", perm)
	}
}
