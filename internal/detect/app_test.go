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

// createAnnotationPackageWithApp creates a fake annotation package with App function.
func createAnnotationPackageWithApp() *types.Package {
	annotationPkg := types.NewPackage(detect.AnnotationPath, "annotation")

	// Create the App function signature: func(func()) struct{}
	emptyStruct := types.NewStruct(nil, nil)
	funcParam := types.NewSignatureType(nil, nil, nil, nil, nil, false) // func()
	appSig := types.NewSignatureType(
		nil,                                                               // receiver
		nil,                                                               // recv type params
		nil,                                                               // type params
		types.NewTuple(types.NewVar(token.NoPos, nil, "", funcParam)),     // params
		types.NewTuple(types.NewVar(token.NoPos, nil, "", emptyStruct)),   // results
		false,                                                             // variadic
	)
	appFunc := types.NewFunc(token.NoPos, annotationPkg, detect.AppFuncName, appSig)
	annotationPkg.Scope().Insert(appFunc)

	annotationPkg.MarkComplete()
	return annotationPkg
}

// mockPassForApp creates a mock analysis.Pass for testing App detection.
func mockPassForApp(t *testing.T, src string, additionalPkgs map[string]*types.Package) (*analysis.Pass, *ast.File) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse test source: %v", err)
	}

	conf := types.Config{
		Importer: &fakeAppImporter{
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

	pkg, _ := conf.Check("main", fset, []*ast.File{file}, info)

	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       pkg,
		TypesInfo: info,
	}

	return pass, file
}

// mockPassWithInspectorForApp creates a mock analysis.Pass with Inspector for testing App detection optimized paths.
func mockPassWithInspectorForApp(t *testing.T, src string, additionalPkgs map[string]*types.Package) (*analysis.Pass, *ast.File) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse test source: %v", err)
	}

	conf := types.Config{
		Importer: &fakeAppImporter{
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

	pkg, _ := conf.Check("main", fset, []*ast.File{file}, info)

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

type fakeAppImporter struct {
	packages map[string]*types.Package
	fallback types.Importer
}

func (i *fakeAppImporter) Import(path string) (*types.Package, error) {
	if pkg, ok := i.packages[path]; ok {
		return pkg, nil
	}
	if i.fallback != nil {
		return i.fallback.Import(path)
	}
	return nil, nil
}

func TestAppDetector_DetectAppAnnotations(t *testing.T) {
	annotationPkg := createAnnotationPackageWithApp()

	tests := []struct {
		name          string
		src           string
		pkgs          map[string]*types.Package
		expectedCount int
	}{
		{
			name: "valid App annotation with main",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)

func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 1,
		},
		{
			name: "no App annotation",
			src: `package main

func main() {}
`,
			pkgs:          nil,
			expectedCount: 0,
		},
		{
			name: "multiple App annotations",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)
var _ = annotation.App(main)

func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 2,
		},
		{
			name: "aliased import",
			src: `package main

import ann "github.com/miyamo2/braider/pkg/annotation"

var _ = ann.App(main)

func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 1,
		},
		{
			name: "App call not in var _ = pattern",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var x = annotation.App(main)

func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 0,
		},
		{
			name: "non-main function reference",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(init)

func init() {}
func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 1, // Detection still finds it, validation will fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPassForApp(t, tt.src, tt.pkgs)

			detector := detect.NewAppDetector()
			apps := detector.DetectAppAnnotations(pass)

			if len(apps) != tt.expectedCount {
				t.Errorf("DetectAppAnnotations() returned %d annotations, want %d", len(apps), tt.expectedCount)
			}
		})
	}
}

func TestAppDetector_ValidateAppAnnotations(t *testing.T) {
	annotationPkg := createAnnotationPackageWithApp()

	tests := []struct {
		name        string
		src         string
		pkgs        map[string]*types.Package
		expectError bool
		errorType   detect.AppValidationErrorType
	}{
		{
			name: "empty annotations - returns nil",
			src: `package main

func main() {}
`,
			pkgs:        nil,
			expectError: false,
		},
		{
			name: "single valid annotation - returns nil",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)

func main() {}
`,
			pkgs:        map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectError: false,
		},
		{
			name: "multiple annotations - returns error",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)
var _ = annotation.App(main)

func main() {}
`,
			pkgs:        map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectError: true,
			errorType:   detect.MultipleAppAnnotations,
		},
		{
			name: "non-main reference - returns error",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(other)

func other() {}
func main() {}
`,
			pkgs:        map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectError: true,
			errorType:   detect.NonMainReference,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPassForApp(t, tt.src, tt.pkgs)

			detector := detect.NewAppDetector()
			apps := detector.DetectAppAnnotations(pass)
			err := detector.ValidateAppAnnotations(pass, apps)

			if tt.expectError {
				if err == nil {
					t.Error("ValidateAppAnnotations() = nil, want error")
					return
				}
				appErr, ok := err.(*detect.AppValidationError)
				if !ok {
					t.Errorf("expected AppValidationError, got %T", err)
					return
				}
				if appErr.Type != tt.errorType {
					t.Errorf("error type = %v, want %v", appErr.Type, tt.errorType)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAppAnnotations() = %v, want nil", err)
				}
			}
		})
	}
}

func TestAppValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *detect.AppValidationError
		expected string
	}{
		{
			name: "multiple app annotations",
			err: &detect.AppValidationError{
				Type:      detect.MultipleAppAnnotations,
				Positions: []token.Pos{1, 2},
			},
			expected: "multiple annotation.App declarations in package",
		},
		{
			name: "non-main reference",
			err: &detect.AppValidationError{
				Type:      detect.NonMainReference,
				Positions: []token.Pos{1},
				FuncName:  "init",
			},
			expected: "annotation.App must reference main function, got init",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAppDetector_DetectAppAnnotations_WithInspector(t *testing.T) {
	annotationPkg := createAnnotationPackageWithApp()

	tests := []struct {
		name          string
		src           string
		pkgs          map[string]*types.Package
		expectedCount int
	}{
		{
			name: "valid App annotation via Inspector",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)

func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 1,
		},
		{
			name: "multiple App annotations via Inspector",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)
var _ = annotation.App(main)

func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPassWithInspectorForApp(t, tt.src, tt.pkgs)

			detector := detect.NewAppDetector()
			apps := detector.DetectAppAnnotations(pass)

			if len(apps) != tt.expectedCount {
				t.Errorf("DetectAppAnnotations() with Inspector returned %d annotations, want %d", len(apps), tt.expectedCount)
			}
		})
	}
}

func TestAppDetector_DetectAppAnnotations_FieldsVerification(t *testing.T) {
	annotationPkg := createAnnotationPackageWithApp()

	src := `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)

func main() {}
`
	pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
	pass, _ := mockPassForApp(t, src, pkgs)

	detector := detect.NewAppDetector()
	apps := detector.DetectAppAnnotations(pass)

	if len(apps) != 1 {
		t.Fatalf("expected 1 App annotation, got %d", len(apps))
	}

	app := apps[0]

	// Verify CallExpr is set
	if app.CallExpr == nil {
		t.Error("AppAnnotation.CallExpr should not be nil")
	}

	// Verify GenDecl is set
	if app.GenDecl == nil {
		t.Error("AppAnnotation.GenDecl should not be nil")
	}

	// Verify MainFunc is set (points to main function)
	if app.MainFunc == nil {
		t.Error("AppAnnotation.MainFunc should not be nil")
	} else if app.MainFunc.Name != "main" {
		t.Errorf("AppAnnotation.MainFunc.Name = %q, want %q", app.MainFunc.Name, "main")
	}

	// Verify Pos is valid
	if app.Pos == 0 {
		t.Error("AppAnnotation.Pos should be non-zero")
	}
}
