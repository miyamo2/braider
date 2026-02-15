package detect

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestNamerValidator_ExtractName_SamePackage(t *testing.T) {
	src := `
package testpkg

import "github.com/miyamo2/braider/pkg/annotation/namer"

// Valid literal namer
type DatabaseName struct{}
func (DatabaseName) Name() string { return "database" }

var _ namer.Namer = DatabaseName{}

// Invalid computed namer
type ComputedName struct{}
func (ComputedName) Name() string {
	name := "computed"
	return name
}

// Invalid concatenation namer
type ConcatName struct{}
func (ConcatName) Name() string { return "prefix" + "suffix" }

// Invalid function call namer
type FuncCallName struct{}
func (FuncCallName) Name() string { return getName() }
func getName() string { return "name" }

// Invalid multiple return statements namer
type BranchingName struct{ flag bool }
func (b BranchingName) Name() string {
	if b.flag {
		return "primary"
	}
	return "secondary"
}

// Invalid keyword namer - "type" is a Go keyword
type TypeKeywordName struct{}
func (TypeKeywordName) Name() string { return "type" }

// Invalid keyword namer - "func" is a Go keyword
type FuncKeywordName struct{}
func (FuncKeywordName) Name() string { return "func" }
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse test source: %v", err)
	}

	// Type check the file
	conf := types.Config{
		Importer: nil,
		Error:    func(err error) {}, // Ignore errors
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	pkg, _ := conf.Check("testpkg", fset, []*ast.File{f}, info)
	if pkg == nil {
		t.Fatal("Failed to type-check package")
	}

	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{f},
		TypesInfo: info,
		Pkg:       pkg,
	}

	validator := NewNamerValidatorImpl(nil) // nil loader for same-package validation

	tests := []struct {
		name        string
		typeName    string
		wantName    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid literal namer",
			typeName: "DatabaseName",
			wantName: "database",
			wantErr:  false,
		},
		{
			name:        "computed value namer",
			typeName:    "ComputedName",
			wantErr:     true,
			errContains: "must return hardcoded string literal",
		},
		{
			name:        "concatenation namer",
			typeName:    "ConcatName",
			wantErr:     true,
			errContains: "must return hardcoded string literal",
		},
		{
			name:        "function call namer",
			typeName:    "FuncCallName",
			wantErr:     true,
			errContains: "must return hardcoded string literal",
		},
		{
			name:        "multiple return statements namer",
			typeName:    "BranchingName",
			wantErr:     true,
			errContains: "must have exactly one return statement",
		},
		{
			name:        "keyword namer - type",
			typeName:    "TypeKeywordName",
			wantErr:     true,
			errContains: "valid Go identifier",
		},
		{
			name:        "keyword namer - func",
			typeName:    "FuncKeywordName",
			wantErr:     true,
			errContains: "valid Go identifier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Find the type in the package
			obj := pkg.Scope().Lookup(tt.typeName)
			if obj == nil {
				t.Fatalf("Type %s not found in package", tt.typeName)
			}

			typeName, ok := obj.Type().(*types.Named)
			if !ok {
				t.Fatalf("Type %s is not a named type", tt.typeName)
			}

			name, err := validator.ExtractName(pass, typeName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractName() expected error, got nil")
					return
				}
				if tt.errContains != "" && !containsStr(err.Error(), tt.errContains) {
					t.Errorf("ExtractName() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractName() unexpected error: %v", err)
				return
			}

			if name != tt.wantName {
				t.Errorf("ExtractName() = %q, want %q", name, tt.wantName)
			}
		})
	}
}

func TestNamerValidator_ExtractName_MethodNotFound(t *testing.T) {
	src := `
package testpkg

type NoMethodType struct{}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse test source: %v", err)
	}

	conf := types.Config{Importer: nil}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	pkg, err := conf.Check("testpkg", fset, []*ast.File{f}, info)
	if err != nil {
		// Ignore type check errors
	}

	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{f},
		TypesInfo: info,
		Pkg:       pkg,
	}

	validator := NewNamerValidatorImpl(nil)

	obj := pkg.Scope().Lookup("NoMethodType")
	if obj == nil {
		t.Fatalf("Type NoMethodType not found")
	}

	typeName := obj.Type().(*types.Named)

	_, err = validator.ExtractName(pass, typeName)
	if err == nil {
		t.Errorf("ExtractName() expected error for type without Name() method, got nil")
	}
}

func TestIsValidGoIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Go keywords (all 25) - must be rejected
		{"keyword break", "break", false},
		{"keyword case", "case", false},
		{"keyword chan", "chan", false},
		{"keyword const", "const", false},
		{"keyword continue", "continue", false},
		{"keyword default", "default", false},
		{"keyword defer", "defer", false},
		{"keyword else", "else", false},
		{"keyword fallthrough", "fallthrough", false},
		{"keyword for", "for", false},
		{"keyword func", "func", false},
		{"keyword go", "go", false},
		{"keyword goto", "goto", false},
		{"keyword if", "if", false},
		{"keyword import", "import", false},
		{"keyword interface", "interface", false},
		{"keyword map", "map", false},
		{"keyword package", "package", false},
		{"keyword range", "range", false},
		{"keyword return", "return", false},
		{"keyword select", "select", false},
		{"keyword struct", "struct", false},
		{"keyword switch", "switch", false},
		{"keyword type", "type", false},
		{"keyword var", "var", false},

		// Valid identifiers
		{"simple lowercase", "myVar", true},
		{"underscore prefix", "_private", true},
		{"pascal case", "FooBar", true},
		{"with digit", "x1", true},
		{"single letter", "a", true},
		{"underscore only", "_", true},
		{"unicode letter", "日本語", true},

		// Near-miss identifiers (contain keyword as prefix but are NOT keywords)
		{"near-miss typeOf", "typeOf", true},
		{"near-miss funcName", "funcName", true},
		{"near-miss ifElse", "ifElse", true},
		{"near-miss forEach", "forEach", true},
		{"near-miss important", "important", true},
		{"near-miss defaultValue", "defaultValue", true},

		// Invalid character composition
		{"starts with digit", "123", false},
		{"empty string", "", false},
		{"contains hyphen", "foo-bar", false},
		{"contains space", "foo bar", false},
		{"contains dot", "foo.bar", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidGoIdentifier(tt.input)
			if got != tt.want {
				t.Errorf("isValidGoIdentifier(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// Helper function
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
