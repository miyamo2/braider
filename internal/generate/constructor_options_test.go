package generate_test

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/registry"
)

// TestConstructorGenerator_OptionAwareReturnType tests
// option-aware return type selection in ConstructorGenerator
func TestConstructorGenerator_OptionAwareReturnType(t *testing.T) {
	tests := []struct {
		name             string
		structName       string
		fields           []detect.FieldInfo
		injectorInfo     *registry.InjectorInfo
		expectedFuncName string
		expectedContains []string
		expectSkip       bool // For WithoutConstructor option
	}{
		{
			name:       "Default option - returns pointer to struct",
			structName: "UserService",
			fields: []detect.FieldInfo{
				{Name: "repo", TypeExpr: &ast.Ident{Name: "Repository"}},
			},
			injectorInfo: &registry.InjectorInfo{
				TypeName:       "example.com/service.UserService",
				LocalName:      "UserService",
				RegisteredType: types.NewPointer(types.NewNamed(types.NewTypeName(token.NoPos, nil, "UserService", nil), nil, nil)),
				OptionMetadata: detect.OptionMetadata{
					IsDefault: true,
				},
			},
			expectedFuncName: "NewUserService",
			expectedContains: []string{
				"func NewUserService(repo Repository) *UserService",
				"return &UserService{",
			},
		},
		{
			name:       "Typed[I] option - returns interface type",
			structName: "UserService",
			fields: []detect.FieldInfo{
				{Name: "repo", TypeExpr: &ast.Ident{Name: "Repository"}},
			},
			injectorInfo: &registry.InjectorInfo{
				TypeName:  "example.com/service.UserService",
				LocalName: "UserService",
				// RegisteredType is the interface type
				RegisteredType: types.NewNamed(types.NewTypeName(token.NoPos, nil, "UserServiceInterface", nil), nil, nil),
				OptionMetadata: detect.OptionMetadata{
					TypedInterface: types.NewNamed(types.NewTypeName(token.NoPos, nil, "UserServiceInterface", nil), nil, nil),
				},
			},
			expectedFuncName: "NewUserService",
			expectedContains: []string{
				"func NewUserService(repo Repository) UserServiceInterface",
				"return &UserService{",
			},
		},
		{
			name:       "WithoutConstructor option - skip generation",
			structName: "CustomService",
			fields: []detect.FieldInfo{
				{Name: "config", TypeExpr: &ast.Ident{Name: "Config"}},
			},
			injectorInfo: &registry.InjectorInfo{
				TypeName:  "example.com/service.CustomService",
				LocalName: "CustomService",
				OptionMetadata: detect.OptionMetadata{
					WithoutConstructor: true,
				},
			},
			expectedFuncName: "NewCustomService",
			expectSkip:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := generate.NewConstructorGenerator()
			candidate := detect.ConstructorCandidate{
				TypeSpec: &ast.TypeSpec{
					Name: &ast.Ident{Name: tt.structName},
				},
			}

			result, err := gen.GenerateConstructorWithOptions(candidate, tt.fields, tt.injectorInfo)

			if tt.expectSkip {
				// Should return nil for WithoutConstructor option
				if result != nil {
					t.Errorf("Expected nil result for WithoutConstructor option, got: %v", result)
				}
				if err != nil {
					t.Errorf("Expected no error for WithoutConstructor option, got: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("GenerateConstructorWithOptions() error = %v", err)
			}

			if result.FuncName != tt.expectedFuncName {
				t.Errorf("FuncName = %s, want %s", result.FuncName, tt.expectedFuncName)
			}

			for _, contains := range tt.expectedContains {
				if !strings.Contains(result.Code, contains) {
					t.Errorf("Code missing expected content: %q\nGot:\n%s", contains, result.Code)
				}
			}
		})
	}
}

// TestConstructorGenerator_NamedDependencyParameters tests
// named dependency parameter naming in ConstructorGenerator
func TestConstructorGenerator_NamedDependencyParameters(t *testing.T) {
	tests := []struct {
		name             string
		structName       string
		fields           []detect.FieldInfo
		injectorInfo     *registry.InjectorInfo
		dependencyNames  map[string]string // Maps field type to named dependency name
		expectedContains []string
	}{
		{
			name:       "Named dependencies with custom parameter names",
			structName: "OrderService",
			fields: []detect.FieldInfo{
				{Name: "primaryRepo", TypeExpr: &ast.Ident{Name: "Repository"}},
				{Name: "cacheRepo", TypeExpr: &ast.Ident{Name: "Repository"}},
				{Name: "logger", TypeExpr: &ast.Ident{Name: "Logger"}},
			},
			injectorInfo: &registry.InjectorInfo{
				TypeName:  "example.com/service.OrderService",
				LocalName: "OrderService",
				OptionMetadata: detect.OptionMetadata{
					IsDefault: true,
				},
			},
			dependencyNames: map[string]string{
				"primaryRepo": "primary",
				"cacheRepo":   "cache",
			},
			expectedContains: []string{
				"func NewOrderService(primary Repository, cache Repository, logger Logger) *OrderService",
				"primaryRepo: primary,",
				"cacheRepo:   cache,",
				"logger:      logger,",
			},
		},
		{
			name:       "Mixed named and unnamed dependencies",
			structName: "UserService",
			fields: []detect.FieldInfo{
				{Name: "mainDB", TypeExpr: &ast.Ident{Name: "Database"}},
				{Name: "cache", TypeExpr: &ast.Ident{Name: "Cache"}},
			},
			injectorInfo: &registry.InjectorInfo{
				TypeName:  "example.com/service.UserService",
				LocalName: "UserService",
				OptionMetadata: detect.OptionMetadata{
					IsDefault: true,
				},
			},
			dependencyNames: map[string]string{
				"mainDB": "primary",
			},
			expectedContains: []string{
				"func NewUserService(primary Database, cache Cache) *UserService",
				"mainDB: primary,",
				"cache:  cache,",
			},
		},
		{
			name:       "No named dependencies - uses default parameter names",
			structName: "SimpleService",
			fields: []detect.FieldInfo{
				{Name: "repo", TypeExpr: &ast.Ident{Name: "Repository"}},
			},
			injectorInfo: &registry.InjectorInfo{
				TypeName:  "example.com/service.SimpleService",
				LocalName: "SimpleService",
				OptionMetadata: detect.OptionMetadata{
					IsDefault: true,
				},
			},
			dependencyNames: map[string]string{},
			expectedContains: []string{
				"func NewSimpleService(repo Repository) *SimpleService",
				"repo: repo,",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := generate.NewConstructorGenerator()
			candidate := detect.ConstructorCandidate{
				TypeSpec: &ast.TypeSpec{
					Name: &ast.Ident{Name: tt.structName},
				},
			}

			result, err := gen.GenerateConstructorWithNamedDeps(candidate, tt.fields, tt.injectorInfo, tt.dependencyNames)
			if err != nil {
				t.Fatalf("GenerateConstructorWithNamedDeps() error = %v", err)
			}

			for _, contains := range tt.expectedContains {
				if !strings.Contains(result.Code, contains) {
					t.Errorf("Code missing expected content: %q\nGot:\n%s", contains, result.Code)
				}
			}
		})
	}
}

// TestConstructorGenerator_CombinedOptionsAndNamedDeps tests the combination of both features
func TestConstructorGenerator_CombinedOptionsAndNamedDeps(t *testing.T) {
	gen := generate.NewConstructorGenerator()
	candidate := detect.ConstructorCandidate{
		TypeSpec: &ast.TypeSpec{
			Name: &ast.Ident{Name: "OrderService"},
		},
	}

	fields := []detect.FieldInfo{
		{Name: "primaryRepo", TypeExpr: &ast.Ident{Name: "Repository"}},
		{Name: "logger", TypeExpr: &ast.Ident{Name: "Logger"}},
	}

	injectorInfo := &registry.InjectorInfo{
		TypeName:  "example.com/service.OrderService",
		LocalName: "OrderService",
		// Typed option - returns interface
		RegisteredType: types.NewNamed(types.NewTypeName(token.NoPos, nil, "OrderServiceInterface", nil), nil, nil),
		OptionMetadata: detect.OptionMetadata{
			TypedInterface: types.NewNamed(types.NewTypeName(token.NoPos, nil, "OrderServiceInterface", nil), nil, nil),
		},
	}

	dependencyNames := map[string]string{
		"primaryRepo": "primary",
	}

	result, err := gen.GenerateConstructorWithNamedDeps(candidate, fields, injectorInfo, dependencyNames)
	if err != nil {
		t.Fatalf("GenerateConstructorWithNamedDeps() error = %v", err)
	}

	// Should have interface return type AND named parameters
	expectedContains := []string{
		"func NewOrderService(primary Repository, logger Logger) OrderServiceInterface",
		"primaryRepo: primary,",
		"logger:      logger,",
		"return &OrderService{",
	}

	for _, contains := range expectedContains {
		if !strings.Contains(result.Code, contains) {
			t.Errorf("Code missing expected content: %q\nGot:\n%s", contains, result.Code)
		}
	}
}
