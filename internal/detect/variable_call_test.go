package detect_test

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
)

// createAnnotationPackageWithVariable creates a fake annotation package with a non-generic Variable function.
// The function signature is: func Variable(any) _variable
// where _variable is a named type in the annotation package.
func createAnnotationPackageWithVariable() *types.Package {
	annotationPkg := types.NewPackage(detect.AnnotationPath, "annotation")

	// Create the _variable named type (returned by Variable)
	variableStruct := types.NewStruct(nil, nil)
	variableNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, annotationPkg, "_variable", nil),
		variableStruct,
		nil,
	)
	annotationPkg.Scope().Insert(variableNamed.Obj())

	// Create the Variable function: func(any) _variable
	anyType := types.Universe.Lookup("any").Type()
	variableSig := types.NewSignatureType(
		nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "variable", anyType)),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", variableNamed)),
		false,
	)
	variableFunc := types.NewFunc(token.NoPos, annotationPkg, detect.VariableTypeName, variableSig)
	annotationPkg.Scope().Insert(variableFunc)

	annotationPkg.MarkComplete()
	return annotationPkg
}

func TestVariableCallDetector_DetectVariables(t *testing.T) {
	annotationPkg := createAnnotationPackageWithVariable()

	// Create an os package with Stdout
	osPkg := types.NewPackage("os", "os")
	fileStruct := types.NewStruct(nil, nil)
	fileNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, osPkg, "File", nil),
		fileStruct,
		nil,
	)
	osPkg.Scope().Insert(fileNamed.Obj())
	// Stdout and Stderr are *os.File
	stdoutVar := types.NewVar(token.NoPos, osPkg, "Stdout", types.NewPointer(fileNamed))
	osPkg.Scope().Insert(stdoutVar)
	stderrVar := types.NewVar(token.NoPos, osPkg, "Stderr", types.NewPointer(fileNamed))
	osPkg.Scope().Insert(stderrVar)
	osPkg.MarkComplete()

	tests := []struct {
		name          string
		src           string
		pkgs          map[string]*types.Package
		expectedCount int
	}{
		{
			name: "valid annotation.Variable(os.Stdout)",
			src: `package test

import (
	"os"
	"github.com/miyamo2/braider/pkg/annotation"
)

var _ = annotation.Variable(os.Stdout)
`,
			pkgs: map[string]*types.Package{
				detect.AnnotationPath: annotationPkg,
				"os":                  osPkg,
			},
			expectedCount: 1,
		},
		{
			name: "multiple Variable calls",
			src: `package test

import (
	"os"
	"github.com/miyamo2/braider/pkg/annotation"
)

var _ = annotation.Variable(os.Stdout)
var _ = annotation.Variable(os.Stderr)
`,
			pkgs: map[string]*types.Package{
				detect.AnnotationPath: annotationPkg,
				"os":                  osPkg,
			},
			expectedCount: 2,
		},
		{
			name: "no Variable calls",
			src: `package test

var x = 42
`,
			pkgs:          nil,
			expectedCount: 0,
		},
		{
			name: "wrong function name annotation.NotVariable",
			src: `package test

import (
	"os"
	"github.com/miyamo2/braider/pkg/annotation"
)

var _ = annotation.Provider(os.Stdout)
`,
			pkgs: map[string]*types.Package{
				detect.AnnotationPath: annotationPkg,
				"os":                  osPkg,
			},
			expectedCount: 0,
		},
		{
			name: "value is not call expression",
			src: `package test

var _ = 42
`,
			pkgs:          nil,
			expectedCount: 0,
		},
		{
			name: "aliased import",
			src: `package test

import (
	"os"
	ann "github.com/miyamo2/braider/pkg/annotation"
)

var _ = ann.Variable(os.Stdout)
`,
			pkgs: map[string]*types.Package{
				detect.AnnotationPath: annotationPkg,
				"os":                  osPkg,
			},
			expectedCount: 1,
		},
		{
			name: "Provide call should not be detected as Variable",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

func NewRepo() *MyRepo { return nil }

type MyRepo struct{}

var _ = annotation.Provide(NewRepo)
`,
			pkgs: func() map[string]*types.Package {
				// Also add Provider to the annotation package for this test
				pkg := types.NewPackage(detect.AnnotationPath, "annotation")
				providerStruct := types.NewStruct(nil, nil)
				providerNamed := types.NewNamed(
					types.NewTypeName(token.NoPos, pkg, "Provider", nil),
					providerStruct,
					nil,
				)
				pkg.Scope().Insert(providerNamed.Obj())
				anyType := types.Universe.Lookup("any").Type()
				provideSig := types.NewSignatureType(
					nil, nil, nil,
					types.NewTuple(types.NewVar(token.NoPos, nil, "providerFunc", anyType)),
					types.NewTuple(types.NewVar(token.NoPos, nil, "", providerNamed)),
					false,
				)
				provideFunc := types.NewFunc(token.NoPos, pkg, "Provide", provideSig)
				pkg.Scope().Insert(provideFunc)
				pkg.MarkComplete()
				return map[string]*types.Package{detect.AnnotationPath: pkg}
			}(),
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPass(t, tt.src, tt.pkgs)

			detector := detect.NewVariableCallDetector()
			candidates, _ := detector.DetectVariables(pass)

			if len(candidates) != tt.expectedCount {
				t.Errorf("DetectVariables() returned %d candidates, want %d", len(candidates), tt.expectedCount)
			}
		})
	}
}

func TestVariableCallDetector_DetectVariables_CandidateFields(t *testing.T) {
	annotationPkg := createAnnotationPackageWithVariable()

	osPkg := types.NewPackage("os", "os")
	fileStruct := types.NewStruct(nil, nil)
	fileNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, osPkg, "File", nil),
		fileStruct,
		nil,
	)
	osPkg.Scope().Insert(fileNamed.Obj())
	stdoutVar := types.NewVar(token.NoPos, osPkg, "Stdout", types.NewPointer(fileNamed))
	osPkg.Scope().Insert(stdoutVar)
	osPkg.MarkComplete()

	src := `package test

import (
	"os"
	"github.com/miyamo2/braider/pkg/annotation"
)

var _ = annotation.Variable(os.Stdout)
`
	pkgs := map[string]*types.Package{
		detect.AnnotationPath: annotationPkg,
		"os":                  osPkg,
	}
	pass, _ := mockPass(t, src, pkgs)

	detector := detect.NewVariableCallDetector()
	candidates, errs := detector.DetectVariables(pass)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	c := candidates[0]

	// Verify CallExpr is set
	if c.CallExpr == nil {
		t.Error("VariableCandidate.CallExpr should not be nil")
	}

	// Verify ArgumentExpr is set
	if c.ArgumentExpr == nil {
		t.Error("VariableCandidate.ArgumentExpr should not be nil")
	}

	// Verify ArgumentType is set
	if c.ArgumentType == nil {
		t.Error("VariableCandidate.ArgumentType should not be nil")
	}

	// Verify ExpressionText is non-empty
	if c.ExpressionText == "" {
		t.Error("VariableCandidate.ExpressionText should not be empty")
	}

	// Verify IsQualified is true for os.Stdout (SelectorExpr)
	if !c.IsQualified {
		t.Error("VariableCandidate.IsQualified should be true for os.Stdout")
	}

	// Verify ExpressionPkgs contains the os package
	if len(c.ExpressionPkgs) == 0 {
		t.Error("VariableCandidate.ExpressionPkgs should not be empty for os.Stdout")
	}
}

func TestVariableCallDetector_DetectVariables_WithInspector(t *testing.T) {
	annotationPkg := createAnnotationPackageWithVariable()

	osPkg := types.NewPackage("os", "os")
	fileStruct := types.NewStruct(nil, nil)
	fileNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, osPkg, "File", nil),
		fileStruct,
		nil,
	)
	osPkg.Scope().Insert(fileNamed.Obj())
	stdoutVar2 := types.NewVar(token.NoPos, osPkg, "Stdout", types.NewPointer(fileNamed))
	osPkg.Scope().Insert(stdoutVar2)
	stderrVar2 := types.NewVar(token.NoPos, osPkg, "Stderr", types.NewPointer(fileNamed))
	osPkg.Scope().Insert(stderrVar2)
	osPkg.MarkComplete()

	tests := []struct {
		name          string
		src           string
		pkgs          map[string]*types.Package
		expectedCount int
	}{
		{
			name: "valid Variable via Inspector",
			src: `package test

import (
	"os"
	"github.com/miyamo2/braider/pkg/annotation"
)

var _ = annotation.Variable(os.Stdout)
`,
			pkgs: map[string]*types.Package{
				detect.AnnotationPath: annotationPkg,
				"os":                  osPkg,
			},
			expectedCount: 1,
		},
		{
			name: "multiple Variables via Inspector",
			src: `package test

import (
	"os"
	"github.com/miyamo2/braider/pkg/annotation"
)

var _ = annotation.Variable(os.Stdout)
var _ = annotation.Variable(os.Stderr)
`,
			pkgs: map[string]*types.Package{
				detect.AnnotationPath: annotationPkg,
				"os":                  osPkg,
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPassWithInspector(t, tt.src, tt.pkgs)

			detector := detect.NewVariableCallDetector()
			candidates, _ := detector.DetectVariables(pass)

			if len(candidates) != tt.expectedCount {
				t.Errorf("DetectVariables() with Inspector returned %d candidates, want %d", len(candidates), tt.expectedCount)
			}
		})
	}
}

func TestVariableCallDetector_DetectVariables_LocalVariable(t *testing.T) {
	annotationPkg := createAnnotationPackageWithVariable()

	src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

type Config struct{}

var defaultConfig = &Config{}

var _ = annotation.Variable(defaultConfig)
`
	pkgs := map[string]*types.Package{
		detect.AnnotationPath: annotationPkg,
	}
	pass, _ := mockPass(t, src, pkgs)

	detector := detect.NewVariableCallDetector()
	candidates, errs := detector.DetectVariables(pass)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	c := candidates[0]

	// Local variable should NOT be qualified
	if c.IsQualified {
		t.Error("VariableCandidate.IsQualified should be false for local variable")
	}

	// ExpressionText should contain "defaultConfig"
	if c.ExpressionText == "" {
		t.Error("VariableCandidate.ExpressionText should not be empty")
	}
}

func TestVariableCallDetector_DetectVariables_NoArguments(t *testing.T) {
	annotationPkg := createAnnotationPackageWithVariable()

	src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.Variable()
`
	pkgs := map[string]*types.Package{
		detect.AnnotationPath: annotationPkg,
	}
	pass, _ := mockPass(t, src, pkgs)

	detector := detect.NewVariableCallDetector()
	candidates, errs := detector.DetectVariables(pass)

	// No arguments -> no candidate
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates for Variable() with no args, got %d", len(candidates))
	}
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestVariableCallDetector_DetectVariables_AliasedImportNormalization(t *testing.T) {
	annotationPkg := createAnnotationPackageWithVariable()

	osPkg := types.NewPackage("os", "os")
	fileStruct := types.NewStruct(nil, nil)
	fileNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, osPkg, "File", nil),
		fileStruct,
		nil,
	)
	osPkg.Scope().Insert(fileNamed.Obj())
	stdoutVar := types.NewVar(token.NoPos, osPkg, "Stdout", types.NewPointer(fileNamed))
	osPkg.Scope().Insert(stdoutVar)
	osPkg.MarkComplete()

	src := `package test

import (
	myos "os"
	"github.com/miyamo2/braider/pkg/annotation"
)

var _ = annotation.Variable(myos.Stdout)
`
	pkgs := map[string]*types.Package{
		detect.AnnotationPath: annotationPkg,
		"os":                  osPkg,
	}
	pass, _ := mockPass(t, src, pkgs)

	detector := detect.NewVariableCallDetector()
	candidates, errs := detector.DetectVariables(pass)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	c := candidates[0]

	// ExpressionText should be normalized to declared package name "os", not alias "myos"
	if c.ExpressionText != "os.Stdout" {
		t.Errorf("ExpressionText = %q, want %q (should normalize alias to declared name)", c.ExpressionText, "os.Stdout")
	}

	// ExpressionPkgs should use declared name "os", not alias "myos"
	if name, ok := c.ExpressionPkgs["os"]; !ok {
		t.Error("ExpressionPkgs should contain key \"os\"")
	} else if name != "os" {
		t.Errorf("ExpressionPkgs[\"os\"] = %q, want %q", name, "os")
	}

	// IsQualified should be true (SelectorExpr)
	if !c.IsQualified {
		t.Error("IsQualified should be true for package-qualified expression")
	}
}

func TestVariableCallDetector_DetectVariables_UnsupportedExpression(t *testing.T) {
	annotationPkg := createAnnotationPackageWithVariable()

	tests := []struct {
		name             string
		src              string
		pkgs             map[string]*types.Package
		expectedErrors   int
		expectedExprDesc string
	}{
		{
			name: "basic literal (int)",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.Variable(42)
`,
			pkgs: map[string]*types.Package{
				detect.AnnotationPath: annotationPkg,
			},
			expectedErrors:   1,
			expectedExprDesc: "literal value",
		},
		{
			name: "function call",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

func getVal() int { return 42 }

var _ = annotation.Variable(getVal())
`,
			pkgs: map[string]*types.Package{
				detect.AnnotationPath: annotationPkg,
			},
			expectedErrors:   1,
			expectedExprDesc: "function call",
		},
		{
			name: "non-package selector (struct field access)",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type Config struct {
	Value int
}

var cfg = Config{Value: 42}

var _ = annotation.Variable(cfg.Value)
`,
			pkgs: map[string]*types.Package{
				detect.AnnotationPath: annotationPkg,
			},
			expectedErrors:   1,
			expectedExprDesc: "non-package selector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPass(t, tt.src, tt.pkgs)

			detector := detect.NewVariableCallDetector()
			candidates, errs := detector.DetectVariables(pass)

			if len(candidates) != 0 {
				t.Errorf("expected 0 candidates, got %d", len(candidates))
			}

			if len(errs) != tt.expectedErrors {
				t.Fatalf("expected %d errors, got %d", tt.expectedErrors, len(errs))
			}

			if tt.expectedErrors > 0 {
				if errs[0].ExprDescription != tt.expectedExprDesc {
					t.Errorf("error ExprDescription = %q, want %q", errs[0].ExprDescription, tt.expectedExprDesc)
				}
				// Verify error message format
				errMsg := errs[0].Error()
				if !strings.Contains(errMsg, "unsupported Variable argument") {
					t.Errorf("error message should contain 'unsupported Variable argument', got %q", errMsg)
				}
				if !strings.Contains(errMsg, tt.expectedExprDesc) {
					t.Errorf("error message should contain %q, got %q", tt.expectedExprDesc, errMsg)
				}
			}
		})
	}
}
