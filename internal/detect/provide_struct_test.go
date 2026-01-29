package detect_test

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
)

func TestProvideStructDetector_DetectProviders(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	tests := []struct {
		name          string
		src           string
		pkgs          map[string]*types.Package
		expectedCount int
		expectedNames []string
	}{
		{
			name: "single struct with Provide",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	annotation.Provide
}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 1,
			expectedNames: []string{"MyRepository"},
		},
		{
			name: "multiple structs with Provide",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type RepositoryA struct {
	annotation.Provide
}

type RepositoryB struct {
	annotation.Provide
}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 2,
			expectedNames: []string{"RepositoryA", "RepositoryB"},
		},
		{
			name: "struct without Provide is skipped",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type RepositoryA struct {
	annotation.Provide
}

type RepositoryB struct {
	name string
}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 1,
			expectedNames: []string{"RepositoryA"},
		},
		{
			name: "no structs with Provide",
			src: `package test

type Repository struct {
	name string
}
`,
			pkgs:          nil,
			expectedCount: 0,
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPassForProvide(t, tt.src, tt.pkgs)

			provideDetector := detect.NewProvideDetector()
			detector := detect.NewProvideStructDetector(provideDetector)
			candidates := detector.DetectProviders(pass)

			if len(candidates) != tt.expectedCount {
				t.Errorf("DetectProviders() returned %d candidates, want %d", len(candidates), tt.expectedCount)
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

func TestProvideStructDetector_FindExistingConstructor(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

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

type MyRepository struct {
	annotation.Provide
}

func NewMyRepository() *MyRepository {
	return &MyRepository{}
}
`,
			structName:  "MyRepository",
			pkgs:        map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectFound: true,
		},
		{
			name: "no existing constructor",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	annotation.Provide
}
`,
			structName:  "MyRepository",
			pkgs:        map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectFound: false,
		},
		{
			name: "wrong constructor name",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	annotation.Provide
}

func CreateMyRepository() *MyRepository {
	return &MyRepository{}
}
`,
			structName:  "MyRepository",
			pkgs:        map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectFound: false,
		},
		{
			name: "constructor returns wrong type",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	annotation.Provide
}

type OtherRepo struct{}

func NewMyRepository() *OtherRepo {
	return &OtherRepo{}
}
`,
			structName:  "MyRepository",
			pkgs:        map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectFound: false,
		},
		{
			name: "method is excluded",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	annotation.Provide
}

type Factory struct{}

func (f *Factory) NewMyRepository() *MyRepository {
	return &MyRepository{}
}
`,
			structName:  "MyRepository",
			pkgs:        map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectFound: false,
		},
		{
			name: "constructor returns non-pointer",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	annotation.Provide
}

func NewMyRepository() MyRepository {
	return MyRepository{}
}
`,
			structName:  "MyRepository",
			pkgs:        map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPassForProvide(t, tt.src, tt.pkgs)

			provideDetector := detect.NewProvideDetector()
			detector := detect.NewProvideStructDetector(provideDetector)
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

func TestProvideStructDetector_CandidateHasExistingConstructor(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

type RepositoryA struct {
	annotation.Provide
}

func NewRepositoryA() *RepositoryA {
	return &RepositoryA{}
}

type RepositoryB struct {
	annotation.Provide
}
`
	pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
	pass, _ := mockPassForProvide(t, src, pkgs)

	provideDetector := detect.NewProvideDetector()
	detector := detect.NewProvideStructDetector(provideDetector)
	candidates := detector.DetectProviders(pass)

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	var repoA, repoB *detect.ProviderCandidate
	for i := range candidates {
		if candidates[i].TypeSpec.Name.Name == "RepositoryA" {
			repoA = &candidates[i]
		}
		if candidates[i].TypeSpec.Name.Name == "RepositoryB" {
			repoB = &candidates[i]
		}
	}

	if repoA == nil {
		t.Fatal("RepositoryA candidate not found")
	}
	if repoB == nil {
		t.Fatal("RepositoryB candidate not found")
	}

	if repoA.ExistingConstructor == nil {
		t.Error("RepositoryA should have existing constructor")
	}
	if repoB.ExistingConstructor != nil {
		t.Error("RepositoryB should not have existing constructor")
	}
}

func TestProvideStructDetector_CandidatePackagePath(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	annotation.Provide
}
`
	pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
	pass, _ := mockPassForProvide(t, src, pkgs)

	provideDetector := detect.NewProvideDetector()
	detector := detect.NewProvideStructDetector(provideDetector)
	candidates := detector.DetectProviders(pass)

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	if candidates[0].PackagePath != "test" {
		t.Errorf("PackagePath = %q, want %q", candidates[0].PackagePath, "test")
	}
}

func TestProvideStructDetector_DetectImplementedInterfaces(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	tests := []struct {
		name       string
		src        string
		pkgs       map[string]*types.Package
		structName string
		expected   []string
	}{
		{
			name: "implements single interface in same package",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type Reader interface {
	Read() error
}

type MyRepository struct {
	annotation.Provide
}

func (m *MyRepository) Read() error {
	return nil
}
`,
			pkgs:       map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			structName: "MyRepository",
			expected:   []string{"test.Reader"},
		},
		{
			name: "implements multiple interfaces in same package",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type Reader interface {
	Read() error
}

type Writer interface {
	Write() error
}

type MyRepository struct {
	annotation.Provide
}

func (m *MyRepository) Read() error {
	return nil
}

func (m *MyRepository) Write() error {
	return nil
}
`,
			pkgs:       map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			structName: "MyRepository",
			expected:   []string{"test.Reader", "test.Writer"},
		},
		{
			name: "implements no interface",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type Reader interface {
	Read() error
}

type MyRepository struct {
	annotation.Provide
}

func (m *MyRepository) DoSomething() {
}
`,
			pkgs:       map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			structName: "MyRepository",
			expected:   []string{},
		},
		{
			name: "implements interface with value receiver",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type Stringer interface {
	String() string
}

type MyRepository struct {
	annotation.Provide
	name string
}

func (m MyRepository) String() string {
	return m.name
}
`,
			pkgs:       map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			structName: "MyRepository",
			expected:   []string{"test.Stringer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, file := mockPassForProvide(t, tt.src, tt.pkgs)

			// Find the TypeSpec for the struct
			var typeSpec *ast.TypeSpec
			ast.Inspect(file, func(n ast.Node) bool {
				if ts, ok := n.(*ast.TypeSpec); ok {
					if ts.Name.Name == tt.structName {
						typeSpec = ts
						return false
					}
				}
				return true
			})

			if typeSpec == nil {
				t.Fatalf("%s struct not found", tt.structName)
			}

			provideDetector := detect.NewProvideDetector()
			detector := detect.NewProvideStructDetector(provideDetector)
			interfaces := detector.DetectImplementedInterfaces(pass, typeSpec)

			if len(interfaces) != len(tt.expected) {
				t.Errorf("DetectImplementedInterfaces() returned %d interfaces, want %d", len(interfaces), len(tt.expected))
				t.Logf("got: %v", interfaces)
				t.Logf("want: %v", tt.expected)
				return
			}

			// Check that all expected interfaces are found (order may vary)
			for _, expected := range tt.expected {
				found := false
				for _, iface := range interfaces {
					if iface == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected interface %q not found in %v", expected, interfaces)
				}
			}
		})
	}
}

func TestProvideStructDetector_DetectImplementedInterfaces_NilTypesInfo(t *testing.T) {
	// When TypesInfo.Defs returns nil, should return empty slice
	src := `package test

type MyRepository struct {
	name string
}
`
	pass, file := mockPassWithoutTypesInfo(t, src)

	var typeSpec *ast.TypeSpec
	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if ts.Name.Name == "MyRepository" {
				typeSpec = ts
				return false
			}
		}
		return true
	})

	if typeSpec == nil {
		t.Fatal("MyRepository struct not found")
	}

	provideDetector := detect.NewProvideDetector()
	detector := detect.NewProvideStructDetector(provideDetector)
	interfaces := detector.DetectImplementedInterfaces(pass, typeSpec)

	if len(interfaces) != 0 {
		t.Errorf("expected empty interfaces when TypesInfo is nil, got %v", interfaces)
	}
}

func TestProvideStructDetector_FindExistingConstructor_WithoutTypesInfo(t *testing.T) {
	// Test AST fallback path (isPointerToStructAST) when TypesInfo is nil
	tests := []struct {
		name        string
		src         string
		structName  string
		expectFound bool
	}{
		{
			name: "finds constructor via AST fallback - correct return type",
			src: `package test

type MyRepository struct {
	name string
}

func NewMyRepository() *MyRepository {
	return &MyRepository{}
}
`,
			structName:  "MyRepository",
			expectFound: true,
		},
		{
			name: "rejects constructor with wrong return type via AST",
			src: `package test

type MyRepository struct{}

type OtherRepo struct{}

func NewMyRepository() *OtherRepo {
	return &OtherRepo{}
}
`,
			structName:  "MyRepository",
			expectFound: false,
		},
		{
			name: "rejects constructor with non-pointer return via AST",
			src: `package test

type MyRepository struct{}

func NewMyRepository() MyRepository {
	return MyRepository{}
}
`,
			structName:  "MyRepository",
			expectFound: false,
		},
		{
			name: "no constructor defined",
			src: `package test

type MyRepository struct{}
`,
			structName:  "MyRepository",
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPassWithoutTypesInfo(t, tt.src)

			provideDetector := detect.NewProvideDetector()
			detector := detect.NewProvideStructDetector(provideDetector)
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

func TestProvideStructDetector_DetectProviders_WithInspector(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	tests := []struct {
		name          string
		src           string
		pkgs          map[string]*types.Package
		expectedCount int
		expectedNames []string
	}{
		{
			name: "single struct with Provide via Inspector",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type MyRepository struct {
	annotation.Provide
}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 1,
			expectedNames: []string{"MyRepository"},
		},
		{
			name: "multiple structs with Provide via Inspector",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

type RepositoryA struct {
	annotation.Provide
}

type RepositoryB struct {
	annotation.Provide
}
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 2,
			expectedNames: []string{"RepositoryA", "RepositoryB"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, _ := mockPassWithInspectorForProvide(t, tt.src, tt.pkgs)

			provideDetector := detect.NewProvideDetector()
			detector := detect.NewProvideStructDetector(provideDetector)
			candidates := detector.DetectProviders(pass)

			if len(candidates) != tt.expectedCount {
				t.Errorf("DetectProviders() with Inspector returned %d candidates, want %d", len(candidates), tt.expectedCount)
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
