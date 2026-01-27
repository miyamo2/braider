package detect_test

import (
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
)

func TestStructDetector_DetectCandidates(t *testing.T) {
	annotationPkg := createAnnotationPackage()

	tests := []struct {
		name           string
		src            string
		pkgs           map[string]*types.Package
		expectedCount  int
		expectedNames  []string
	}{
		{
			name: "single struct with Inject",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	repo Repository
}

type Repository interface{}
`,
			pkgs:          map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expectedCount: 1,
			expectedNames: []string{"MyService"},
		},
		{
			name: "multiple structs with Inject",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type ServiceA struct {
	annotation.Inject
	repo Repository
}

type ServiceB struct {
	annotation.Inject
	logger Logger
}

type Repository interface{}
type Logger interface{}
`,
			pkgs:          map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expectedCount: 2,
			expectedNames: []string{"ServiceA", "ServiceB"},
		},
		{
			name: "struct without Inject is skipped",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type ServiceA struct {
	annotation.Inject
	repo Repository
}

type ServiceB struct {
	logger Logger
}

type Repository interface{}
type Logger interface{}
`,
			pkgs:          map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expectedCount: 1,
			expectedNames: []string{"ServiceA"},
		},
		{
			name: "no structs with Inject",
			src: `package test

type ServiceA struct {
	repo Repository
}

type Repository interface{}
`,
			pkgs:          nil,
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name: "interface type is skipped",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	repo Repository
}

type Repository interface {
	Find() error
}
`,
			pkgs:          map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expectedCount: 1,
			expectedNames: []string{"MyService"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPass(t, tt.src, tt.pkgs)

			detector := detect.NewStructDetector(detect.NewInjectDetector())
			candidates := detector.DetectCandidates(pass)

			if len(candidates) != tt.expectedCount {
				t.Errorf("DetectCandidates() returned %d candidates, want %d", len(candidates), tt.expectedCount)
			}

			for i, name := range tt.expectedNames {
				if i >= len(candidates) {
					break
				}
				if candidates[i].TypeSpec.Name.Name != name {
					t.Errorf("candidate[%d].Name = %s, want %s", i, candidates[i].TypeSpec.Name.Name, name)
				}
			}
		})
	}
}

func TestStructDetector_FindExistingConstructor(t *testing.T) {
	annotationPkg := createAnnotationPackage()

	tests := []struct {
		name        string
		src         string
		structName  string
		pkgs        map[string]*types.Package
		expectFound bool
	}{
		{
			name: "finds existing constructor",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	repo Repository
}

func NewMyService(repo Repository) *MyService {
	return &MyService{repo: repo}
}

type Repository interface{}
`,
			structName:  "MyService",
			pkgs:        map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expectFound: true,
		},
		{
			name: "no existing constructor",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	repo Repository
}

type Repository interface{}
`,
			structName:  "MyService",
			pkgs:        map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expectFound: false,
		},
		{
			name: "wrong constructor name (different prefix)",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	repo Repository
}

func CreateMyService(repo Repository) *MyService {
	return &MyService{repo: repo}
}

type Repository interface{}
`,
			structName:  "MyService",
			pkgs:        map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expectFound: false,
		},
		{
			name: "constructor returns wrong type",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	repo Repository
}

type OtherService struct{}

func NewMyService(repo Repository) *OtherService {
	return &OtherService{}
}

type Repository interface{}
`,
			structName:  "MyService",
			pkgs:        map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expectFound: false,
		},
		{
			name: "constructor returns non-pointer",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyService struct {
	annotation.Inject
	repo Repository
}

func NewMyService(repo Repository) MyService {
	return MyService{repo: repo}
}

type Repository interface{}
`,
			structName:  "MyService",
			pkgs:        map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg},
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPass(t, tt.src, tt.pkgs)

			detector := detect.NewStructDetector(detect.NewInjectDetector())
			fn := detector.FindExistingConstructor(pass, tt.structName)

			if tt.expectFound && fn == nil {
				t.Error("FindExistingConstructor() = nil, want non-nil")
			}
			if !tt.expectFound && fn != nil {
				t.Errorf("FindExistingConstructor() = %v, want nil", fn.Name.Name)
			}
		})
	}
}

func TestStructDetector_CandidateHasExistingConstructor(t *testing.T) {
	annotationPkg := createAnnotationPackage()

	src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

type ServiceA struct {
	annotation.Inject
	repo Repository
}

func NewServiceA(repo Repository) *ServiceA {
	return &ServiceA{repo: repo}
}

type ServiceB struct {
	annotation.Inject
	logger Logger
}

type Repository interface{}
type Logger interface{}
`
	pkgs := map[string]*types.Package{detect.InjectAnnotationPath: annotationPkg}
	pass, _ := mockPass(t, src, pkgs)

	detector := detect.NewStructDetector(detect.NewInjectDetector())
	candidates := detector.DetectCandidates(pass)

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	// ServiceA should have existing constructor
	var serviceA, serviceB *detect.ConstructorCandidate
	for i := range candidates {
		if candidates[i].TypeSpec.Name.Name == "ServiceA" {
			serviceA = &candidates[i]
		}
		if candidates[i].TypeSpec.Name.Name == "ServiceB" {
			serviceB = &candidates[i]
		}
	}

	if serviceA == nil {
		t.Fatal("ServiceA candidate not found")
	}
	if serviceB == nil {
		t.Fatal("ServiceB candidate not found")
	}

	if serviceA.ExistingConstructor == nil {
		t.Error("ServiceA should have existing constructor")
	}
	if serviceB.ExistingConstructor != nil {
		t.Error("ServiceB should not have existing constructor")
	}
}
