package detect

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestOptionExtractor_ExtractInjectOptions_Default(t *testing.T) {
	src := `
package testpkg

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type MyService struct {
	annotation.Injectable[inject.Default]
}
`

	pass, st := setupTestPass(t, src)
	extractor := NewOptionExtractor(nil) // nil validator for default option

	// Find the Injectable field
	field := findInjectableField(t, pass, st)
	concreteType := pass.TypesInfo.TypeOf(ast.NewIdent("MyService"))

	metadata, err := extractor.ExtractInjectOptions(pass, field.Type, concreteType)
	if err != nil {
		t.Fatalf("ExtractInjectOptions() unexpected error: %v", err)
	}

	if !metadata.IsDefault {
		t.Errorf("Expected IsDefault=true, got false")
	}
	if metadata.TypedInterface != nil {
		t.Errorf("Expected TypedInterface=nil, got %v", metadata.TypedInterface)
	}
	if metadata.Name != "" {
		t.Errorf("Expected Name=\"\", got %q", metadata.Name)
	}
	if metadata.WithoutConstructor {
		t.Errorf("Expected WithoutConstructor=false, got true")
	}
}

func TestOptionExtractor_ExtractInjectOptions_WithoutConstructor(t *testing.T) {
	src := `
package testpkg

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type MyService struct {
	annotation.Injectable[inject.WithoutConstructor]
}
`

	pass, st := setupTestPass(t, src)
	extractor := NewOptionExtractor(nil)

	field := findInjectableField(t, pass, st)
	concreteType := pass.TypesInfo.TypeOf(ast.NewIdent("MyService"))

	metadata, err := extractor.ExtractInjectOptions(pass, field.Type, concreteType)
	if err != nil {
		t.Fatalf("ExtractInjectOptions() unexpected error: %v", err)
	}

	if metadata.IsDefault {
		t.Errorf("Expected IsDefault=false, got true")
	}
	if !metadata.WithoutConstructor {
		t.Errorf("Expected WithoutConstructor=true, got false")
	}
}

// setupTestPass creates a test analysis pass from source code
func setupTestPass(t *testing.T, src string) (*analysis.Pass, *ast.StructType) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	conf := types.Config{
		Importer: nil,
		Error:    func(err error) {}, // Ignore import errors
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

	// Find the struct type
	var st *ast.StructType
	ast.Inspect(f, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if s, ok := ts.Type.(*ast.StructType); ok {
				st = s
				return false
			}
		}
		return true
	})

	if st == nil {
		t.Fatal("Failed to find struct type")
	}

	return pass, st
}

// findInjectableField finds the Injectable embedded field
func findInjectableField(t *testing.T, pass *analysis.Pass, st *ast.StructType) *ast.Field {
	t.Helper()

	if st.Fields == nil {
		t.Fatal("Struct has no fields")
	}

	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			// Embedded field
			return field
		}
	}

	t.Fatal("No embedded field found")
	return nil
}
