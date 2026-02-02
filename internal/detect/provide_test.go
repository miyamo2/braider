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
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// createAnnotationPackageWithProvide creates a fake annotation package with both Inject and Provide types.
func createAnnotationPackageWithProvide() *types.Package {
	annotationPkg := types.NewPackage(detect.AnnotationPath, "annotation")

	// Create the Inject struct type
	injectStruct := types.NewStruct(nil, nil)
	injectNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, annotationPkg, detect.InjectTypeName, nil),
		injectStruct,
		nil,
	)
	annotationPkg.Scope().Insert(injectNamed.Obj())

	// Create the Provide struct type
	provideStruct := types.NewStruct(nil, nil)
	provideNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, annotationPkg, detect.ProvideTypeName, nil),
		provideStruct,
		nil,
	)
	annotationPkg.Scope().Insert(provideNamed.Obj())

	annotationPkg.MarkComplete()
	return annotationPkg
}

// createWrongAnnotationPackageWithProvide creates a fake annotation package with wrong path.
func createWrongAnnotationPackageWithProvide() *types.Package {
	wrongPkg := types.NewPackage("github.com/other/annotation", "annotation")
	provideStruct := types.NewStruct(nil, nil)
	provideNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, wrongPkg, detect.ProvideTypeName, nil),
		provideStruct,
		nil,
	)
	wrongPkg.Scope().Insert(provideNamed.Obj())
	wrongPkg.MarkComplete()
	return wrongPkg
}

// mockPassForProvide creates a mock analysis.Pass for testing Provide detection.
func mockPassForProvide(t *testing.T, src string, additionalPkgs map[string]*types.Package) (*analysis.Pass, *ast.File) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse test source: %v", err)
	}

	conf := types.Config{
		Importer: &fakeProvideImporter{
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

// mockPassWithInspectorForProvide creates a mock analysis.Pass with Inspector for testing Provide detection optimized paths.
func mockPassWithInspectorForProvide(t *testing.T, src string, additionalPkgs map[string]*types.Package) (*analysis.Pass, *ast.File) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse test source: %v", err)
	}

	conf := types.Config{
		Importer: &fakeProvideImporter{
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

	// Create Inspector
	insp := inspector.New([]*ast.File{file})

	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       pkg,
		TypesInfo: info,
		ResultOf: map[*analysis.Analyzer]any{
			inspect.Analyzer: insp,
		},
	}

	return pass, file
}

type fakeProvideImporter struct {
	packages map[string]*types.Package
	fallback types.Importer
}

func (i *fakeProvideImporter) Import(path string) (*types.Package, error) {
	if pkg, ok := i.packages[path]; ok {
		return pkg, nil
	}
	if i.fallback != nil {
		return i.fallback.Import(path)
	}
	return nil, nil
}

func TestProvideDetector_HasProvideAnnotation(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()
	wrongPkg := createWrongAnnotationPackageWithProvide()

	tests := []struct {
		name     string
		src      string
		pkgs     map[string]*types.Package
		expected bool
	}{
		{
			name: "standard annotation.Provide embedding",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	annotation.Provide
}
`,
			pkgs:     map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expected: true,
		},
		{
			name: "no Provide embedding",
			src: `package test

type MyRepository struct {
	name string
}
`,
			pkgs:     nil,
			expected: false,
		},
		{
			name: "named Provide field (not embedded)",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	provide annotation.Provide
}
`,
			pkgs:     map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expected: false,
		},
		{
			name: "Provide embedding at end of struct",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	name string
	annotation.Provide
}
`,
			pkgs:     map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expected: true,
		},
		{
			name: "Provide from wrong package (should be ignored)",
			src: `package test

import "github.com/other/annotation"

type MyRepository struct {
	annotation.Provide
}
`,
			pkgs:     map[string]*types.Package{"github.com/other/annotation": wrongPkg},
			expected: false,
		},
		{
			name: "Inject embedding (not Provide)",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
}
`,
			pkgs:     map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, file := mockPassForProvide(t, tt.src, tt.pkgs)

			// Find the first struct type
			var structType *ast.StructType
			ast.Inspect(file, func(n ast.Node) bool {
				if ts, ok := n.(*ast.TypeSpec); ok {
					if st, ok := ts.Type.(*ast.StructType); ok {
						structType = st
						return false
					}
				}
				return true
			})

			if structType == nil {
				t.Fatal("struct not found")
			}

			detector := detect.NewProvideDetector()
			result := detector.HasProvideAnnotation(pass, structType)

			if result != tt.expected {
				t.Errorf("HasProvideAnnotation() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestProvideDetector_FindProvideField(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	tests := []struct {
		name          string
		src           string
		pkgs          map[string]*types.Package
		expectNil     bool
		expectedIndex int
	}{
		{
			name: "finds embedded Provide field at start",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	annotation.Provide
	name string
}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectNil:     false,
			expectedIndex: 0,
		},
		{
			name: "finds embedded Provide field at end",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	name string
	annotation.Provide
}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectNil:     false,
			expectedIndex: 1,
		},
		{
			name: "returns nil when no Provide embedding",
			src: `package test

type MyRepository struct {
	name string
}
`,
			pkgs:      nil,
			expectNil: true,
		},
		{
			name: "returns nil for named Provide field",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	provide annotation.Provide
}
`,
			pkgs:      map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, file := mockPassForProvide(t, tt.src, tt.pkgs)

			// Find the first struct type
			var structType *ast.StructType
			ast.Inspect(file, func(n ast.Node) bool {
				if ts, ok := n.(*ast.TypeSpec); ok {
					if st, ok := ts.Type.(*ast.StructType); ok {
						structType = st
						return false
					}
				}
				return true
			})

			if structType == nil {
				t.Fatal("struct not found")
			}

			detector := detect.NewProvideDetector()
			field := detector.FindProvideField(pass, structType)

			if tt.expectNil {
				if field != nil {
					t.Errorf("FindProvideField() = %v, want nil", field)
				}
			} else {
				if field == nil {
					t.Fatal("FindProvideField() = nil, want non-nil")
				}
				// Verify it's the correct field by index
				if tt.expectedIndex < len(structType.Fields.List) {
					expectedField := structType.Fields.List[tt.expectedIndex]
					if field != expectedField {
						t.Errorf("FindProvideField() returned wrong field")
					}
				}
			}
		})
	}
}

func TestProvideDetector_FindProvideField_TypeOfFallback(t *testing.T) {
	// Test the TypeOf fallback path when Types map doesn't contain the expression.
	// This happens when type checking fails or is incomplete.

	src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	annotation.Provide
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Create TypesInfo with empty Types map to force TypeOf fallback
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue), // empty - will force TypeOf fallback
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		TypesInfo: info,
	}

	// Find the struct type
	var structType *ast.StructType
	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if st, ok := ts.Type.(*ast.StructType); ok {
				structType = st
				return false
			}
		}
		return true
	})

	if structType == nil {
		t.Fatal("struct not found")
	}

	// When Types map is empty and TypeOf returns nil, should return nil (no provide found)
	detector := detect.NewProvideDetector()
	field := detector.FindProvideField(pass, structType)

	// Should return nil because type information is not available
	if field != nil {
		t.Errorf("FindProvideField() should return nil when type info is incomplete, got field")
	}
}

func TestProvideDetector_AliasedImport(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	src := `package test

import ann "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	ann.Provide
}
`
	pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
	pass, file := mockPassForProvide(t, src, pkgs)

	// Find the struct type
	var structType *ast.StructType
	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if st, ok := ts.Type.(*ast.StructType); ok {
				structType = st
				return false
			}
		}
		return true
	})

	if structType == nil {
		t.Fatal("struct not found")
	}

	detector := detect.NewProvideDetector()

	if !detector.HasProvideAnnotation(pass, structType) {
		t.Error("HasProvideAnnotation() = false, want true for aliased import")
	}

	field := detector.FindProvideField(pass, structType)
	if field == nil {
		t.Error("FindProvideField() = nil, want non-nil for aliased import")
	}
}

func TestProvideDetector_EdgeCases(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	tests := []struct {
		name           string
		src            string
		pkgs           map[string]*types.Package
		expectProvide  bool
		description    string
	}{
		{
			name: "nil struct fields",
			src: `package test

type MyService struct {
}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectProvide: false,
			description:   "Empty struct should not have provide",
		},
		{
			name: "struct with only named fields",
			src: `package test

type MyService struct {
	repo Repository
}

type Repository interface{}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectProvide: false,
			description:   "Struct with only named fields should not have provide",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, file := mockPassForProvide(t, tt.src, tt.pkgs)

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

			detector := detect.NewProvideDetector()
			hasProvide := detector.HasProvideAnnotation(pass, structType)

			if hasProvide != tt.expectProvide {
				t.Errorf("HasProvideAnnotation() = %v, want %v (%s)", hasProvide, tt.expectProvide, tt.description)
			}
		})
	}
}

func TestProvideDetector_FindProvideField_NilStructFields(t *testing.T) {
	// Test with a struct that has nil Fields
	pass, _ := mockPassForProvide(t, "package test", nil)

	structType := &ast.StructType{
		Fields: nil, // Explicitly nil
	}

	detector := detect.NewProvideDetector()
	field := detector.FindProvideField(pass, structType)

	if field != nil {
		t.Errorf("FindProvideField() with nil struct fields = %v, want nil", field)
	}
}

func TestProvideDetector_TypeCheckingEdgeCases(t *testing.T) {
	wrongPkg := createWrongAnnotationPackageWithProvide()

	tests := []struct {
		name          string
		src           string
		pkgs          map[string]*types.Package
		expectProvide bool
		description   string
	}{
		{
			name: "provide with wrong package path",
			src: `package test

import "github.com/other/annotation"

type MyService struct {
	annotation.Provide
}
`,
			pkgs:          map[string]*types.Package{"github.com/other/annotation": wrongPkg},
			expectProvide: false,
			description:   "Provide from wrong package should not be detected",
		},
		{
			name: "provide with wrong type name",
			src: `package test

type Provide struct{}

type MyService struct {
	Provide
}
`,
			pkgs:          nil,
			expectProvide: false,
			description:   "Local Provide type should not be detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, file := mockPassForProvide(t, tt.src, tt.pkgs)

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

			detector := detect.NewProvideDetector()
			hasProvide := detector.HasProvideAnnotation(pass, structType)

			if hasProvide != tt.expectProvide {
				t.Errorf("HasProvideAnnotation() = %v, want %v (%s)", hasProvide, tt.expectProvide, tt.description)
			}
		})
	}
}
