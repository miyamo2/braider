package detect_test

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
)

func TestFieldAnalyzer_AnalyzeFields(t *testing.T) {
	annotationPkg := createAnnotationPackage()

	tests := []struct {
		name           string
		src            string
		pkgs           map[string]*types.Package
		expectedCount  int
		expectedFields []string
	}{
		{
			name: "single field excluding Inject",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	repo Repository
}

type Repository interface{}
`,
			pkgs:           map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount:  1,
			expectedFields: []string{"repo"},
		},
		{
			name: "multiple fields excluding Inject",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	repo   Repository
	logger Logger
	config Config
}

type Repository interface{}
type Logger interface{}
type Config struct{}
`,
			pkgs:           map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount:  3,
			expectedFields: []string{"repo", "logger", "config"},
		},
		{
			name: "Inject at end of struct",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	repo   Repository
	logger Logger
	annotation.Inject
}

type Repository interface{}
type Logger interface{}
`,
			pkgs:           map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount:  2,
			expectedFields: []string{"repo", "logger"},
		},
		{
			name: "exported and unexported fields",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	Repo   Repository
	logger Logger
}

type Repository interface{}
type Logger interface{}
`,
			pkgs:           map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount:  2,
			expectedFields: []string{"Repo", "logger"},
		},
		{
			name: "only Inject field (no injectable fields)",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
}
`,
			pkgs:           map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount:  0,
			expectedFields: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				pass, file := mockPass(t, tt.src, tt.pkgs)

				// Find the struct type and inject field
				var structType *ast.StructType
				ast.Inspect(
					file, func(n ast.Node) bool {
						if ts, ok := n.(*ast.TypeSpec); ok {
							if st, ok := ts.Type.(*ast.StructType); ok {
								if ts.Name.Name == "MyService" {
									structType = st
									return false
								}
							}
						}
						return true
					},
				)

				if structType == nil {
					t.Fatal("MyService struct not found")
				}

				injectDetector := detect.NewInjectDetector()
				injectField := injectDetector.FindInjectField(pass, structType)

				fieldAnalyzer := detect.NewFieldAnalyzer()
				fields := fieldAnalyzer.AnalyzeFields(pass, structType, injectField)

				if len(fields) != tt.expectedCount {
					t.Errorf("AnalyzeFields() returned %d fields, want %d", len(fields), tt.expectedCount)
				}

				for i, name := range tt.expectedFields {
					if i >= len(fields) {
						break
					}
					if fields[i].Name != name {
						t.Errorf("fields[%d].Name = %s, want %s", i, fields[i].Name, name)
					}
				}
			},
		)
	}
}

func TestFieldAnalyzer_FieldInfo(t *testing.T) {
	annotationPkg := createAnnotationPackage()

	src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	repo     Repository
	logger   *Logger
	config   Config
	ExportedField string
}

type Repository interface{}
type Logger struct{}
type Config struct{}
`
	pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
	pass, file := mockPass(t, src, pkgs)

	var structType *ast.StructType
	ast.Inspect(
		file, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok {
				if st, ok := ts.Type.(*ast.StructType); ok {
					if ts.Name.Name == "MyService" {
						structType = st
						return false
					}
				}
			}
			return true
		},
	)

	if structType == nil {
		t.Fatal("MyService struct not found")
	}

	injectDetector := detect.NewInjectDetector()
	injectField := injectDetector.FindInjectField(pass, structType)

	fieldAnalyzer := detect.NewFieldAnalyzer()
	fields := fieldAnalyzer.AnalyzeFields(pass, structType, injectField)

	if len(fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(fields))
	}

	// Test repo field (interface)
	if fields[0].Name != "repo" {
		t.Errorf("fields[0].Name = %s, want repo", fields[0].Name)
	}
	if !fields[0].IsInterface {
		t.Error("fields[0] should be interface")
	}
	if fields[0].IsPointer {
		t.Error("fields[0] should not be pointer")
	}
	if fields[0].IsExported {
		t.Error("fields[0] should not be exported")
	}

	// Test logger field (pointer to struct)
	if fields[1].Name != "logger" {
		t.Errorf("fields[1].Name = %s, want logger", fields[1].Name)
	}
	if fields[1].IsInterface {
		t.Error("fields[1] should not be interface")
	}
	if !fields[1].IsPointer {
		t.Error("fields[1] should be pointer")
	}

	// Test config field (struct value)
	if fields[2].Name != "config" {
		t.Errorf("fields[2].Name = %s, want config", fields[2].Name)
	}
	if fields[2].IsInterface {
		t.Error("fields[2] should not be interface")
	}
	if fields[2].IsPointer {
		t.Error("fields[2] should not be pointer")
	}

	// Test ExportedField (exported)
	if fields[3].Name != "ExportedField" {
		t.Errorf("fields[3].Name = %s, want ExportedField", fields[3].Name)
	}
	if !fields[3].IsExported {
		t.Error("fields[3] should be exported")
	}
}

func TestFieldAnalyzer_PreservesFieldOrder(t *testing.T) {
	annotationPkg := createAnnotationPackage()

	src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	alpha Alpha
	beta  Beta
	gamma Gamma
	delta Delta
}

type Alpha interface{}
type Beta interface{}
type Gamma interface{}
type Delta interface{}
`
	pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
	pass, file := mockPass(t, src, pkgs)

	var structType *ast.StructType
	ast.Inspect(
		file, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok {
				if st, ok := ts.Type.(*ast.StructType); ok {
					if ts.Name.Name == "MyService" {
						structType = st
						return false
					}
				}
			}
			return true
		},
	)

	injectDetector := detect.NewInjectDetector()
	injectField := injectDetector.FindInjectField(pass, structType)

	fieldAnalyzer := detect.NewFieldAnalyzer()
	fields := fieldAnalyzer.AnalyzeFields(pass, structType, injectField)

	expectedOrder := []string{"alpha", "beta", "gamma", "delta"}
	if len(fields) != len(expectedOrder) {
		t.Fatalf("expected %d fields, got %d", len(expectedOrder), len(fields))
	}

	for i, name := range expectedOrder {
		if fields[i].Name != name {
			t.Errorf("field order mismatch at %d: got %s, want %s", i, fields[i].Name, name)
		}
	}
}

func TestFieldAnalyzer_AnalyzeFields_WithoutTypesInfo(t *testing.T) {
	// Test AST fallback path (isPointerAST) when TypesInfo is nil
	tests := []struct {
		name            string
		src             string
		expectedCount   int
		expectedNames   []string
		expectedPointer []bool
	}{
		{
			name: "detects pointer types via AST fallback",
			src: `package test

type MyService struct {
	repo   *Repository
	logger Logger
	config *Config
}

type Repository struct{}
type Logger interface{}
type Config struct{}
`,
			expectedCount:   3,
			expectedNames:   []string{"repo", "logger", "config"},
			expectedPointer: []bool{true, false, true},
		},
		{
			name: "handles non-pointer types via AST fallback",
			src: `package test

type MyService struct {
	name    string
	count   int
	handler Handler
}

type Handler struct{}
`,
			expectedCount:   3,
			expectedNames:   []string{"name", "count", "handler"},
			expectedPointer: []bool{false, false, false},
		},
		{
			name: "handles mixed pointer and non-pointer types",
			src: `package test

type MyService struct {
	first  *First
	second Second
	third  *Third
	fourth Fourth
}

type First struct{}
type Second struct{}
type Third struct{}
type Fourth struct{}
`,
			expectedCount:   4,
			expectedNames:   []string{"first", "second", "third", "fourth"},
			expectedPointer: []bool{true, false, true, false},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				pass, file := mockPassWithoutTypesInfo(t, tt.src)

				// Find the struct type
				var structType *ast.StructType
				ast.Inspect(
					file, func(n ast.Node) bool {
						if ts, ok := n.(*ast.TypeSpec); ok {
							if st, ok := ts.Type.(*ast.StructType); ok {
								if ts.Name.Name == "MyService" {
									structType = st
									return false
								}
							}
						}
						return true
					},
				)

				if structType == nil {
					t.Fatal("MyService struct not found")
				}

				fieldAnalyzer := detect.NewFieldAnalyzer()
				// Pass nil for injectField since we're testing without annotation package
				fields := fieldAnalyzer.AnalyzeFields(pass, structType, nil)

				if len(fields) != tt.expectedCount {
					t.Errorf("AnalyzeFields() returned %d fields, want %d", len(fields), tt.expectedCount)
					return
				}

				for i, name := range tt.expectedNames {
					if fields[i].Name != name {
						t.Errorf("fields[%d].Name = %s, want %s", i, fields[i].Name, name)
					}
				}

				for i, isPointer := range tt.expectedPointer {
					if fields[i].IsPointer != isPointer {
						t.Errorf(
							"fields[%d].IsPointer = %v, want %v (field: %s)",
							i,
							fields[i].IsPointer,
							isPointer,
							fields[i].Name,
						)
					}
				}

				// Verify Type is nil (since TypesInfo is nil)
				for i, field := range fields {
					if field.Type != nil {
						t.Errorf("fields[%d].Type should be nil when TypesInfo is nil, got %v", i, field.Type)
					}
				}

				// Verify IsInterface is false (cannot determine from AST alone)
				for i, field := range fields {
					if field.IsInterface {
						t.Errorf("fields[%d].IsInterface should be false when TypesInfo is nil", i)
					}
				}
			},
		)
	}
}

func TestFieldAnalyzer_AnalyzeFields_MultipleNamesInSingleField(t *testing.T) {
	annotationPkg := createAnnotationPackage()

	src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	a, b, c int
}
`
	pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
	pass, file := mockPass(t, src, pkgs)

	var structType *ast.StructType
	ast.Inspect(
		file, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok {
				if st, ok := ts.Type.(*ast.StructType); ok {
					if ts.Name.Name == "MyService" {
						structType = st
						return false
					}
				}
			}
			return true
		},
	)

	if structType == nil {
		t.Fatal("MyService struct not found")
	}

	injectDetector := detect.NewInjectDetector()
	injectField := injectDetector.FindInjectField(pass, structType)

	fieldAnalyzer := detect.NewFieldAnalyzer()
	fields := fieldAnalyzer.AnalyzeFields(pass, structType, injectField)

	// Should have 3 fields (a, b, c)
	if len(fields) != 3 {
		t.Errorf("AnalyzeFields() returned %d fields, want 3", len(fields))
	}

	expectedNames := []string{"a", "b", "c"}
	for i, name := range expectedNames {
		if i >= len(fields) {
			break
		}
		if fields[i].Name != name {
			t.Errorf("fields[%d].Name = %s, want %s", i, fields[i].Name, name)
		}
	}
}

func TestFieldAnalyzer_AnalyzeFields_SkipEmbeddedFields(t *testing.T) {
	annotationPkg := createAnnotationPackage()

	src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

type Base struct {
	BaseField string
}

type MyService struct {
	annotation.Inject
	Base  // embedded field, should be skipped
	normalField string
}
`
	pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
	pass, file := mockPass(t, src, pkgs)

	var structType *ast.StructType
	ast.Inspect(
		file, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok {
				if st, ok := ts.Type.(*ast.StructType); ok {
					if ts.Name.Name == "MyService" {
						structType = st
						return false
					}
				}
			}
			return true
		},
	)

	if structType == nil {
		t.Fatal("MyService struct not found")
	}

	injectDetector := detect.NewInjectDetector()
	injectField := injectDetector.FindInjectField(pass, structType)

	fieldAnalyzer := detect.NewFieldAnalyzer()
	fields := fieldAnalyzer.AnalyzeFields(pass, structType, injectField)

	// Should only have normalField (Base is embedded and should be skipped)
	if len(fields) != 1 {
		t.Errorf("AnalyzeFields() returned %d fields, want 1 (embedded fields should be skipped)", len(fields))
	}

	if len(fields) > 0 && fields[0].Name != "normalField" {
		t.Errorf("fields[0].Name = %s, want normalField", fields[0].Name)
	}
}
