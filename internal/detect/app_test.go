package detect_test

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
)

// createAnnotationPackageWithApp creates a mock annotation package with the App function.
func createAnnotationPackageWithApp() *types.Package {
	annotationPkg := types.NewPackage(detect.AnnotationPath, "annotation")
	// func App(interface{})
	appFunc := types.NewFunc(token.NoPos, annotationPkg, detect.AppFuncName,
		types.NewSignature(nil,
			types.NewTuple(types.NewVar(token.NoPos, annotationPkg, "fn", types.NewInterfaceType(nil, nil))),
			nil, false))
	annotationPkg.Scope().Insert(appFunc)
	annotationPkg.MarkComplete()
	return annotationPkg
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
			name: "detects App(main)",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)

func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 1,
		},
		{
			name: "detects multiple App calls",
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
			name: "ignores other function calls",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = otherFunc()

func otherFunc() int { return 0 }
func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 0,
		},
		{
			name: "ignores App call not assigned to blank identifier",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var x = annotation.App(main)

func main() {}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPass(t, tt.src, tt.pkgs)

			detector := detect.NewAppDetector()
			apps := detector.DetectAppAnnotations(pass)

			if len(apps) != tt.expectedCount {
				t.Errorf("DetectAppAnnotations() returned %d apps, want %d", len(apps), tt.expectedCount)
			}
		})
	}
}

func TestAppDetector_ValidateAppAnnotations(t *testing.T) {
	annotationPkg := createAnnotationPackageWithApp()

	tests := []struct {
		name      string
		src       string
		pkgs      map[string]*types.Package
		expectErr bool
	}{
		{
			name: "valid App(main)",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)

func main() {}
`,
			pkgs:      map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectErr: false,
		},
		{
			name: "invalid App(other) - non-main function",
			src: `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(other)

func other() {}
func main() {}
`,
			pkgs:      map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPass(t, tt.src, tt.pkgs)

			detector := detect.NewAppDetector()
			apps := detector.DetectAppAnnotations(pass)

			// Ensure we detected exactly one App annotation
			if len(apps) != 1 {
				t.Fatalf("DetectAppAnnotations() returned %d apps, want 1", len(apps))
			}

			err := detector.ValidateAppAnnotations(pass, apps)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateAppAnnotations() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestAppDetector_DeduplicateAppsByFile(t *testing.T) {
	// Create dummy files and annotations
	file1 := &ast.File{}
	file2 := &ast.File{}

	app1 := &detect.AppAnnotation{File: file1}
	app2 := &detect.AppAnnotation{File: file1} // Duplicate in file1
	app3 := &detect.AppAnnotation{File: file2}

	apps := []*detect.AppAnnotation{app1, app2, app3}

	detector := detect.NewAppDetector()
	result := detector.DeduplicateAppsByFile(apps)

	if len(result) != 2 {
		t.Errorf("DeduplicateAppsByFile() returned %d apps, want 2", len(result))
	}

	// Verify order and uniqueness (first one per file is kept)
	if result[0] != app1 {
		t.Error("First app should be app1")
	}
	if result[1] != app3 {
		t.Error("Second app should be app3")
	}
}
