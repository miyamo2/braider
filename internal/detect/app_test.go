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
// The App function returns a type embedding the internal/annotation.App marker interface.
func createAnnotationPackageWithApp() *types.Package {
	// Create synthetic internal/annotation marker interface for App
	internalPkg := types.NewPackage("github.com/miyamo2/braider/internal/annotation", "annotation")
	markerSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	markerMethod := types.NewFunc(token.NoPos, internalPkg, "_IsApp", markerSig)
	markerIface := types.NewInterfaceType([]*types.Func{markerMethod}, nil)
	markerIface.Complete()
	markerTypeName := types.NewTypeName(token.NoPos, internalPkg, "App", nil)
	markerNamed := types.NewNamed(markerTypeName, markerIface, nil)
	internalPkg.Scope().Insert(markerNamed.Obj())
	internalPkg.MarkComplete()

	// Create pkg/annotation package
	annotationPkg := types.NewPackage(detect.AnnotationPath, "annotation")

	// Create the app struct embedding the internal marker interface
	embeddedField := types.NewField(token.NoPos, nil, "", markerNamed, true)
	appStruct := types.NewStruct([]*types.Var{embeddedField}, nil)
	appReturnType := types.NewNamed(
		types.NewTypeName(token.NoPos, annotationPkg, "app", nil),
		appStruct,
		nil,
	)
	annotationPkg.Scope().Insert(appReturnType.Obj())

	// Create the App function: func[T any](func()) app
	// Generic type parameter is needed so App[T](main) type-checks.
	typeParamName := types.NewTypeName(token.NoPos, annotationPkg, "T", nil)
	anyConstraint := types.NewInterfaceType(nil, nil)
	anyConstraint.Complete()
	typeParam := types.NewTypeParam(typeParamName, anyConstraint)
	funcParam := types.NewSignatureType(nil, nil, nil, nil, nil, false) // func()
	appSig := types.NewSignatureType(
		nil, nil, []*types.TypeParam{typeParam},
		types.NewTuple(types.NewVar(token.NoPos, nil, "", funcParam)),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", appReturnType)),
		false,
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

			detector := detect.NewAppDetector(detect.ResolveMarkers())
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
			name: "multiple annotations - now allowed",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)
var _ = annotation.App(main)

func main() {}
`,
			pkgs:        map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectError: false,
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

			detector := detect.NewAppDetector(detect.ResolveMarkers())
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

			detector := detect.NewAppDetector(detect.ResolveMarkers())
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

	detector := detect.NewAppDetector(detect.ResolveMarkers())
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

func TestAppDetector_DeduplicateAppsByFile(t *testing.T) {
	annotationPkg := createAnnotationPackageWithApp()

	tests := []struct {
		name          string
		src           string
		pkgs          map[string]*types.Package
		expectedCount int
		description   string
	}{
		{
			name: "empty list returns empty",
			src: `package main

func main() {}
`,
			pkgs:          nil,
			expectedCount: 0,
			description:   "Empty input should return empty output",
		},
		{
			name: "single annotation returns single",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)

func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 1,
			description:   "Single annotation should be preserved",
		},
		{
			name: "multiple annotations in same file returns first only",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)
var _ = annotation.App(main)
var _ = annotation.App(main)

func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 1,
			description:   "Multiple annotations in same file should deduplicate to first one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPassForApp(t, tt.src, tt.pkgs)

			detector := detect.NewAppDetector(detect.ResolveMarkers())
			apps := detector.DetectAppAnnotations(pass)
			deduplicated := detector.DeduplicateAppsByFile(apps)

			if len(deduplicated) != tt.expectedCount {
				t.Errorf("DeduplicateAppsByFile() returned %d annotations, want %d (%s)", len(deduplicated), tt.expectedCount, tt.description)
			}

			// Verify all returned apps have File set
			for i, app := range deduplicated {
				if app.File == nil && tt.expectedCount > 0 {
					t.Errorf("deduplicated[%d].File should not be nil", i)
				}
			}
		})
	}
}

func TestAppDetector_DeduplicateAppsByFile_NilFile(t *testing.T) {
	// Test the fallback behavior when File is nil
	detector := detect.NewAppDetector(detect.ResolveMarkers())

	// Create annotations with nil File (edge case)
	apps := []*detect.AppAnnotation{
		{
			CallExpr: &ast.CallExpr{},
			Pos:      1,
			File:     nil, // Explicitly nil
		},
		{
			CallExpr: &ast.CallExpr{},
			Pos:      2,
			File:     nil, // Explicitly nil
		},
	}

	deduplicated := detector.DeduplicateAppsByFile(apps)

	// When File is nil, all apps should be included (fallback behavior)
	if len(deduplicated) != 2 {
		t.Errorf("DeduplicateAppsByFile() with nil Files returned %d annotations, want 2", len(deduplicated))
	}
}

func TestAppValidationError_Error_DefaultCase(t *testing.T) {
	// Test the default case in Error() method
	err := &detect.AppValidationError{
		Type:      detect.AppValidationErrorType(999), // Invalid type
		Positions: []token.Pos{1},
		FuncName:  "test",
	}

	result := err.Error()
	expected := "invalid App annotation"
	if result != expected {
		t.Errorf("Error() = %q, want %q", result, expected)
	}
}

func TestAppDetector_ValidateAppAnnotations_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupApps   func() []*detect.AppAnnotation
		pass        *analysis.Pass
		expectError bool
		description string
	}{
		{
			name: "annotation with nil MainFunc",
			setupApps: func() []*detect.AppAnnotation {
				return []*detect.AppAnnotation{
					{
						CallExpr: &ast.CallExpr{},
						Pos:      1,
						MainFunc: nil, // Nil MainFunc
					},
				}
			},
			expectError: true,
			description: "Should error when MainFunc is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal pass for validation
			fset := token.NewFileSet()
			file, _ := parser.ParseFile(fset, "test.go", "package main\nfunc main() {}", 0)

			info := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
				Defs:  make(map[*ast.Ident]types.Object),
				Uses:  make(map[*ast.Ident]types.Object),
			}

			pass := &analysis.Pass{
				Fset:      fset,
				Files:     []*ast.File{file},
				TypesInfo: info,
			}

			detector := detect.NewAppDetector(detect.ResolveMarkers())
			apps := tt.setupApps()
			err := detector.ValidateAppAnnotations(pass, apps)

			if tt.expectError && err == nil {
				t.Errorf("ValidateAppAnnotations() = nil, want error (%s)", tt.description)
			} else if !tt.expectError && err != nil {
				t.Errorf("ValidateAppAnnotations() = %v, want nil (%s)", err, tt.description)
			}
		})
	}
}

func TestAppDetector_ValidateAppAnnotations_UnknownIdentifier(t *testing.T) {
	annotationPkg := createAnnotationPackageWithApp()

	src := `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(undefinedFunc)

func main() {}
`
	pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
	pass, _ := mockPassForApp(t, src, pkgs)

	detector := detect.NewAppDetector(detect.ResolveMarkers())
	apps := detector.DetectAppAnnotations(pass)

	// Should find the annotation but validation should fail
	if len(apps) != 1 {
		t.Fatalf("expected 1 App annotation, got %d", len(apps))
	}

	err := detector.ValidateAppAnnotations(pass, apps)
	if err == nil {
		t.Error("ValidateAppAnnotations() should error for undefined function reference")
	}
}

func TestAppDetector_ValidateAppAnnotations_NonFunctionObject(t *testing.T) {
	annotationPkg := createAnnotationPackageWithApp()

	src := `package main

import "github.com/miyamo2/braider/pkg/annotation"

const myConst = 42
var _ = annotation.App(myConst)

func main() {}
`
	pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
	pass, _ := mockPassForApp(t, src, pkgs)

	detector := detect.NewAppDetector(detect.ResolveMarkers())
	apps := detector.DetectAppAnnotations(pass)

	// Should find the annotation
	if len(apps) != 1 {
		t.Fatalf("expected 1 App annotation, got %d", len(apps))
	}

	err := detector.ValidateAppAnnotations(pass, apps)
	if err == nil {
		t.Error("ValidateAppAnnotations() should error when referencing non-function")
	}
}

func TestAppDetector_DetectAppAnnotations_EdgeCases(t *testing.T) {
	annotationPkg := createAnnotationPackageWithApp()

	tests := []struct {
		name          string
		src           string
		pkgs          map[string]*types.Package
		expectedCount int
		description   string
	}{
		{
			name: "call expression not a selector",
			src: `package main

var _ = someFunc()

func someFunc() struct{} { return struct{}{} }
func main() {}
`,
			pkgs:          nil,
			expectedCount: 0,
			description:   "Non-selector call should not be detected",
		},
		{
			name: "wrong function name",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.NotApp(main)

func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 0,
			description:   "Wrong function name should not be detected",
		},
		{
			name: "wrong package path",
			src: `package main

import ann "github.com/other/annotation"

var _ = ann.App(main)

func main() {}
`,
			pkgs:          nil,
			expectedCount: 0,
			description:   "Wrong package path should not be detected",
		},
		{
			name: "selector X is not identifier",
			src: `package main

var _ = (struct{App func()}{}).App()

func main() {}
`,
			pkgs:          nil,
			expectedCount: 0,
			description:   "Selector X not being an identifier should not be detected",
		},
		{
			name: "multiple values in var spec",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _, x = annotation.App(main), 42

func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 0,
			description:   "Multiple values in var spec should not be detected",
		},
		{
			name: "valuespec not a call expression",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = 42

func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 0,
			description:   "Non-call expression value should not be detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPassForApp(t, tt.src, tt.pkgs)

			detector := detect.NewAppDetector(detect.ResolveMarkers())
			apps := detector.DetectAppAnnotations(pass)

			if len(apps) != tt.expectedCount {
				t.Errorf("DetectAppAnnotations() returned %d annotations, want %d (%s)", len(apps), tt.expectedCount, tt.description)
			}
		})
	}
}

func TestAppDetector_DetectAppAnnotations_GenericForm(t *testing.T) {
	annotationPkg := createAnnotationPackageWithApp()

	// Create a fake app options package for type arguments
	appOptionsPkg := types.NewPackage("github.com/miyamo2/braider/pkg/annotation/app", "app")
	// Add Default type
	defaultIface := types.NewInterfaceType(nil, nil)
	defaultIface.Complete()
	defaultType := types.NewTypeName(token.NoPos, appOptionsPkg, "Default", defaultIface)
	appOptionsPkg.Scope().Insert(defaultType)
	appOptionsPkg.MarkComplete()

	tests := []struct {
		name               string
		src                string
		pkgs               map[string]*types.Package
		expectedCount      int
		expectTypeArgExpr  bool // whether TypeArgExpr should be non-nil
		description        string
	}{
		{
			name: "generic App with app.Default",
			src: `package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Default](main)

func main() {}
`,
			pkgs: map[string]*types.Package{
				detect.AnnotationPath: annotationPkg,
				"github.com/miyamo2/braider/pkg/annotation/app": appOptionsPkg,
			},
			expectedCount:     1,
			expectTypeArgExpr: true,
			description:       "Generic App[app.Default](main) should be detected with TypeArgExpr",
		},
		{
			name: "non-generic App continues to work",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)

func main() {}
`,
			pkgs: map[string]*types.Package{
				detect.AnnotationPath: annotationPkg,
			},
			expectedCount:     1,
			expectTypeArgExpr: false,
			description:       "Non-generic App(main) should have nil TypeArgExpr",
		},
		{
			name: "generic App with arbitrary type arg",
			src: `package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Default](main)
var _ = annotation.App(main)

func main() {}
`,
			pkgs: map[string]*types.Package{
				detect.AnnotationPath: annotationPkg,
				"github.com/miyamo2/braider/pkg/annotation/app": appOptionsPkg,
			},
			expectedCount:     2,
			expectTypeArgExpr: true, // At least the first one has TypeArgExpr
			description:       "Mix of generic and non-generic forms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPassForApp(t, tt.src, tt.pkgs)

			detector := detect.NewAppDetector(detect.ResolveMarkers())
			apps := detector.DetectAppAnnotations(pass)

			if len(apps) != tt.expectedCount {
				t.Errorf("DetectAppAnnotations() returned %d annotations, want %d (%s)", len(apps), tt.expectedCount, tt.description)
				return
			}

			if tt.expectedCount > 0 && tt.expectTypeArgExpr {
				if apps[0].TypeArgExpr == nil {
					t.Error("Expected TypeArgExpr to be non-nil for generic form")
				}
			}

			if tt.expectedCount > 0 && !tt.expectTypeArgExpr {
				if apps[0].TypeArgExpr != nil {
					t.Error("Expected TypeArgExpr to be nil for non-generic form")
				}
			}
		})
	}
}

func TestAppDetector_DetectAppAnnotations_GenericForm_WithInspector(t *testing.T) {
	annotationPkg := createAnnotationPackageWithApp()

	appOptionsPkg := types.NewPackage("github.com/miyamo2/braider/pkg/annotation/app", "app")
	defaultIface := types.NewInterfaceType(nil, nil)
	defaultIface.Complete()
	defaultType := types.NewTypeName(token.NoPos, appOptionsPkg, "Default", defaultIface)
	appOptionsPkg.Scope().Insert(defaultType)
	appOptionsPkg.MarkComplete()

	src := `package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Default](main)

func main() {}
`
	pkgs := map[string]*types.Package{
		detect.AnnotationPath: annotationPkg,
		"github.com/miyamo2/braider/pkg/annotation/app": appOptionsPkg,
	}
	pass, _ := mockPassWithInspectorForApp(t, src, pkgs)

	detector := detect.NewAppDetector(detect.ResolveMarkers())
	apps := detector.DetectAppAnnotations(pass)

	if len(apps) != 1 {
		t.Fatalf("DetectAppAnnotations() with Inspector returned %d annotations, want 1", len(apps))
	}

	if apps[0].TypeArgExpr == nil {
		t.Error("Expected TypeArgExpr to be non-nil for generic form (Inspector path)")
	}
}

func TestAppDetector_DetectAppAnnotations_GenericForm_FieldsVerification(t *testing.T) {
	annotationPkg := createAnnotationPackageWithApp()

	appOptionsPkg := types.NewPackage("github.com/miyamo2/braider/pkg/annotation/app", "app")
	defaultIface := types.NewInterfaceType(nil, nil)
	defaultIface.Complete()
	defaultType := types.NewTypeName(token.NoPos, appOptionsPkg, "Default", defaultIface)
	appOptionsPkg.Scope().Insert(defaultType)
	appOptionsPkg.MarkComplete()

	src := `package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Default](main)

func main() {}
`
	pkgs := map[string]*types.Package{
		detect.AnnotationPath: annotationPkg,
		"github.com/miyamo2/braider/pkg/annotation/app": appOptionsPkg,
	}
	pass, _ := mockPassForApp(t, src, pkgs)

	detector := detect.NewAppDetector(detect.ResolveMarkers())
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

	// Verify TypeArgExpr is set for generic form
	if app.TypeArgExpr == nil {
		t.Error("AppAnnotation.TypeArgExpr should not be nil for generic form")
	}

	// Verify Pos is valid
	if app.Pos == 0 {
		t.Error("AppAnnotation.Pos should be non-zero")
	}
}

func TestAppDetector_FindFileForNode_NoFileFound(t *testing.T) {
	// Create a pass with no files
	fset := token.NewFileSet()
	pass := &analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{}, // Empty files
	}

	detector := detect.NewAppDetector(detect.ResolveMarkers())
	// Since findFileForNode is private, we test it indirectly through DetectAppAnnotations
	// With empty files, should return empty
	apps := detector.DetectAppAnnotations(pass)

	// With empty files, should return empty
	if len(apps) != 0 {
		t.Errorf("DetectAppAnnotations() with empty files returned %d, want 0", len(apps))
	}
}