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

	validator := NewNamerValidator(nil) // nil loader for same-package validation

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

	validator := NewNamerValidator(nil)

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

// Helper function
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
