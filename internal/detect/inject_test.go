package detect_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"golang.org/x/tools/go/analysis"
)

// mockPass creates a mock analysis.Pass for testing.
func mockPass(t *testing.T, src string, additionalPkgs map[string]*types.Package) (*analysis.Pass, *ast.File) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse test source: %v", err)
	}

	conf := types.Config{
		Importer: &fakeImporter{
			packages: additionalPkgs,
			fallback: importer.Default(),
		},
		Error: func(err error) {
			// Suppress type errors that don't affect our tests
		},
	}

	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	pkg, _ := conf.Check("test", fset, []*ast.File{file}, info)

	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       pkg,
		TypesInfo: info,
	}

	return pass, file
}

type fakeImporter struct {
	packages map[string]*types.Package
	fallback types.Importer
}

func (i *fakeImporter) Import(path string) (*types.Package, error) {
	if pkg, ok := i.packages[path]; ok {
		return pkg, nil
	}
	if i.fallback != nil {
		return i.fallback.Import(path)
	}
	return nil, nil
}

// createAnnotationPackage creates a fake annotation package for testing.
func createAnnotationPackage() *types.Package {
	annotationPkg := types.NewPackage(detect.InjectAnnotationPath, "annotation")
	// Create the Inject struct type - pass nil for underlying, NewNamed will set it
	injectStruct := types.NewStruct(nil, nil)
	injectNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, annotationPkg, detect.InjectTypeName, nil),
		injectStruct,
		nil,
	)
	annotationPkg.Scope().Insert(injectNamed.Obj())
	annotationPkg.MarkComplete()
	return annotationPkg
}

// createWrongAnnotationPackage creates a fake annotation package with wrong path.
func createWrongAnnotationPackage() *types.Package {
	wrongPkg := types.NewPackage("github.com/other/annotation", "annotation")
	injectStruct := types.NewStruct(nil, nil)
	injectNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, wrongPkg, detect.InjectTypeName, nil),
		injectStruct,
		nil,
	)
	wrongPkg.Scope().Insert(injectNamed.Obj())
	wrongPkg.MarkComplete()
	return wrongPkg
}

func TestInjectDetector_HasInjectAnnotation(t *testing.T) {
	annotationPkg := createAnnotationPackage()
	wrongPkg := createWrongAnnotationPackage()

	tests := []struct {
		name     string
		src      string
		pkgs     map[string]*types.Package
		expected bool
	}{
		{
			name: "standard annotation.Inject embedding",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	repo Repository
}

type Repository interface{}
`,
			pkgs:     map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expected: true,
		},
		{
			name: "no Inject embedding",
			src: `package test

type MyService struct {
	repo Repository
}

type Repository interface{}
`,
			pkgs:     nil,
			expected: false,
		},
		{
			name: "named Inject field (not embedded)",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	inject annotation.Inject
	repo   Repository
}

type Repository interface{}
`,
			pkgs:     map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expected: false,
		},
		{
			name: "Inject embedding at end of struct",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	repo Repository
	annotation.Inject
}

type Repository interface{}
`,
			pkgs:     map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expected: true,
		},
		{
			name: "Inject from wrong package (should be ignored)",
			src: `package test

import "github.com/other/annotation"

type MyService struct {
	annotation.Inject
	repo Repository
}

type Repository interface{}
`,
			pkgs:     map[string]*types.Package{"github.com/other/annotation": wrongPkg},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, file := mockPass(t, tt.src, tt.pkgs)

			// Find the struct type
			var structType *ast.StructType
			ast.Inspect(file, func(n ast.Node) bool {
				if ts, ok := n.(*ast.TypeSpec); ok {
					if st, ok := ts.Type.(*ast.StructType); ok {
						if ts.Name.Name == "MyService" {
							structType = st
							return false
						}
					}
				}
				return true
			})

			if structType == nil {
				t.Fatal("MyService struct not found")
			}

			detector := detect.NewInjectDetector()
			result := detector.HasInjectAnnotation(pass, structType)

			if result != tt.expected {
				t.Errorf("HasInjectAnnotation() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestInjectDetector_FindInjectField(t *testing.T) {
	annotationPkg := createAnnotationPackage()

	tests := []struct {
		name          string
		src           string
		pkgs          map[string]*types.Package
		expectNil     bool
		expectedIndex int // index of the field in struct (for non-nil case)
	}{
		{
			name: "finds embedded Inject field at start",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	repo Repository
}

type Repository interface{}
`,
			pkgs:          map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expectNil:     false,
			expectedIndex: 0,
		},
		{
			name: "finds embedded Inject field at end",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	repo Repository
	annotation.Inject
}

type Repository interface{}
`,
			pkgs:          map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expectNil:     false,
			expectedIndex: 1,
		},
		{
			name: "returns nil when no Inject embedding",
			src: `package test

type MyService struct {
	repo Repository
}

type Repository interface{}
`,
			pkgs:      nil,
			expectNil: true,
		},
		{
			name: "returns nil for named Inject field",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	inject annotation.Inject
	repo   Repository
}

type Repository interface{}
`,
			pkgs:      map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, file := mockPass(t, tt.src, tt.pkgs)

			// Find the struct type
			var structType *ast.StructType
			ast.Inspect(file, func(n ast.Node) bool {
				if ts, ok := n.(*ast.TypeSpec); ok {
					if st, ok := ts.Type.(*ast.StructType); ok {
						if ts.Name.Name == "MyService" {
							structType = st
							return false
						}
					}
				}
				return true
			})

			if structType == nil {
				t.Fatal("MyService struct not found")
			}

			detector := detect.NewInjectDetector()
			field := detector.FindInjectField(pass, structType)

			if tt.expectNil {
				if field != nil {
					t.Errorf("FindInjectField() = %v, want nil", field)
				}
			} else {
				if field == nil {
					t.Fatal("FindInjectField() = nil, want non-nil")
				}
				// Verify it's the correct field by index
				if tt.expectedIndex < len(structType.Fields.List) {
					expectedField := structType.Fields.List[tt.expectedIndex]
					if field != expectedField {
						t.Errorf("FindInjectField() returned wrong field")
					}
				}
			}
		})
	}
}

func TestInjectDetector_AliasedImport(t *testing.T) {
	annotationPkg := createAnnotationPackage()

	src := `package test

import ann "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	ann.Inject
	repo Repository
}

type Repository interface{}
`
	pkgs := map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg}
	pass, file := mockPass(t, src, pkgs)

	// Find the struct type
	var structType *ast.StructType
	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if st, ok := ts.Type.(*ast.StructType); ok {
				if ts.Name.Name == "MyService" {
					structType = st
					return false
				}
			}
		}
		return true
	})

	if structType == nil {
		t.Fatal("MyService struct not found")
	}

	detector := detect.NewInjectDetector()

	if !detector.HasInjectAnnotation(pass, structType) {
		t.Error("HasInjectAnnotation() = false, want true for aliased import")
	}

	field := detector.FindInjectField(pass, structType)
	if field == nil {
		t.Error("FindInjectField() = nil, want non-nil for aliased import")
	}
}
