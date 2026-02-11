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

// TestConstructorGenerator_WithoutConstructor_SkipLogic tests WithoutConstructor skip logic
func TestConstructorGenerator_WithoutConstructor_SkipLogic(t *testing.T) {
	t.Run("skips generation for WithoutConstructor option", func(t *testing.T) {
		gen := generate.NewConstructorGenerator()
		candidate := detect.ConstructorCandidate{
			TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "CustomService"},
			},
		}

		fields := []detect.FieldInfo{
			{Name: "config", TypeExpr: &ast.Ident{Name: "Config"}},
		}

		injectorInfo := &registry.InjectorInfo{
			TypeName:  "example.com/service.CustomService",
			LocalName: "CustomService",
			OptionMetadata: detect.OptionMetadata{
				WithoutConstructor: true,
			},
		}

		result, err := gen.GenerateConstructorWithOptions(candidate, fields, injectorInfo)

		if err != nil {
			t.Errorf("GenerateConstructorWithOptions() unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("GenerateConstructorWithOptions() expected nil result for WithoutConstructor, got: %v", result)
		}
	})

	t.Run("skips generation with named deps for WithoutConstructor option", func(t *testing.T) {
		gen := generate.NewConstructorGenerator()
		candidate := detect.ConstructorCandidate{
			TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "CustomService"},
			},
		}

		fields := []detect.FieldInfo{
			{Name: "config", TypeExpr: &ast.Ident{Name: "Config"}},
		}

		injectorInfo := &registry.InjectorInfo{
			TypeName:  "example.com/service.CustomService",
			LocalName: "CustomService",
			OptionMetadata: detect.OptionMetadata{
				WithoutConstructor: true,
			},
		}

		dependencyNames := map[string]string{
			"config": "customConfig",
		}

		result, err := gen.GenerateConstructorWithNamedDeps(candidate, fields, injectorInfo, dependencyNames)

		if err != nil {
			t.Errorf("GenerateConstructorWithNamedDeps() unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("GenerateConstructorWithNamedDeps() expected nil result for WithoutConstructor, got: %v", result)
		}
	})

	t.Run("generates code for Default option", func(t *testing.T) {
		gen := generate.NewConstructorGenerator()
		candidate := detect.ConstructorCandidate{
			TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "UserService"},
			},
		}

		fields := []detect.FieldInfo{
			{Name: "repo", TypeExpr: &ast.Ident{Name: "Repository"}},
		}

		injectorInfo := &registry.InjectorInfo{
			TypeName:  "example.com/service.UserService",
			LocalName: "UserService",
			OptionMetadata: detect.OptionMetadata{
				IsDefault: true,
			},
		}

		result, err := gen.GenerateConstructorWithOptions(candidate, fields, injectorInfo)

		if err != nil {
			t.Fatalf("GenerateConstructorWithOptions() unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("GenerateConstructorWithOptions() expected non-nil result for Default option")
		}
		if result.FuncName != "NewUserService" {
			t.Errorf("FuncName = %s, want NewUserService", result.FuncName)
		}
	})
}

// TestConstructorGenerator_ReturnTypeSelection tests return type selection based on options
func TestConstructorGenerator_ReturnTypeSelection(t *testing.T) {
	tests := []struct {
		name             string
		structName       string
		injectorInfo     *registry.InjectorInfo
		expectedReturn   string
		expectedContains []string
	}{
		{
			name:       "Default option returns *ConcreteStruct",
			structName: "UserService",
			injectorInfo: &registry.InjectorInfo{
				TypeName:  "example.com/service.UserService",
				LocalName: "UserService",
				OptionMetadata: detect.OptionMetadata{
					IsDefault: true,
				},
			},
			expectedReturn: "*UserService",
			expectedContains: []string{
				"func NewUserService() *UserService",
			},
		},
		{
			name:       "Typed[I] option returns interface",
			structName: "UserService",
			injectorInfo: &registry.InjectorInfo{
				TypeName:  "example.com/service.UserService",
				LocalName: "UserService",
				RegisteredType: types.NewNamed(
					types.NewTypeName(token.NoPos, nil, "IUserService", nil),
					types.NewStruct(nil, nil),
					nil,
				),
				OptionMetadata: detect.OptionMetadata{
					TypedInterface: types.NewNamed(
						types.NewTypeName(token.NoPos, nil, "IUserService", nil),
						types.NewStruct(nil, nil),
						nil,
					),
				},
			},
			expectedReturn: "IUserService",
			expectedContains: []string{
				"func NewUserService() IUserService",
			},
		},
		{
			name:       "Typed[I] with qualified package name",
			structName: "OrderService",
			injectorInfo: &registry.InjectorInfo{
				TypeName:  "example.com/service.OrderService",
				LocalName: "OrderService",
				RegisteredType: types.NewNamed(
					types.NewTypeName(
						token.NoPos,
						types.NewPackage("example.com/interfaces", "interfaces"),
						"IOrderService",
						nil,
					),
					types.NewStruct(nil, nil),
					nil,
				),
				OptionMetadata: detect.OptionMetadata{
					TypedInterface: types.NewNamed(
						types.NewTypeName(
							token.NoPos,
							types.NewPackage("example.com/interfaces", "interfaces"),
							"IOrderService",
							nil,
						),
						types.NewStruct(nil, nil),
						nil,
					),
				},
			},
			expectedReturn: "IOrderService",
			expectedContains: []string{
				"func NewOrderService() IOrderService",
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

			result, err := gen.GenerateConstructorWithOptions(candidate, nil, tt.injectorInfo)
			if err != nil {
				t.Fatalf("GenerateConstructorWithOptions() error = %v", err)
			}

			for _, contains := range tt.expectedContains {
				if !strings.Contains(result.Code, contains) {
					t.Errorf("Code missing expected content: %q\nGot:\n%s", contains, result.Code)
				}
			}
		})
	}
}

// TestConstructorGenerator_NamedDependencyParameterNaming tests named dependency parameter naming
func TestConstructorGenerator_NamedDependencyParameterNaming(t *testing.T) {
	tests := []struct {
		name             string
		structName       string
		fields           []detect.FieldInfo
		dependencyNames  map[string]string
		expectedContains []string
	}{
		{
			name:       "single named dependency",
			structName: "UserService",
			fields: []detect.FieldInfo{
				{Name: "repo", TypeExpr: &ast.Ident{Name: "Repository"}},
			},
			dependencyNames: map[string]string{
				"repo": "primaryRepo",
			},
			expectedContains: []string{
				"func NewUserService(primaryRepo Repository) *UserService",
				"repo: primaryRepo,",
			},
		},
		{
			name:       "multiple named dependencies",
			structName: "OrderService",
			fields: []detect.FieldInfo{
				{Name: "userRepo", TypeExpr: &ast.Ident{Name: "UserRepository"}},
				{Name: "orderRepo", TypeExpr: &ast.Ident{Name: "OrderRepository"}},
			},
			dependencyNames: map[string]string{
				"userRepo":  "primary",
				"orderRepo": "secondary",
			},
			expectedContains: []string{
				"func NewOrderService(primary UserRepository, secondary OrderRepository) *OrderService",
				"userRepo:  primary,",
				"orderRepo: secondary,",
			},
		},
		{
			name:       "mixed named and default dependencies",
			structName: "PaymentService",
			fields: []detect.FieldInfo{
				{Name: "repo", TypeExpr: &ast.Ident{Name: "Repository"}},
				{Name: "logger", TypeExpr: &ast.Ident{Name: "Logger"}},
				{Name: "cache", TypeExpr: &ast.Ident{Name: "Cache"}},
			},
			dependencyNames: map[string]string{
				"repo": "primaryRepo",
				// logger and cache use default names
			},
			expectedContains: []string{
				"func NewPaymentService(primaryRepo Repository, logger Logger, cache Cache) *PaymentService",
				"repo:   primaryRepo,",
				"logger: logger,",
				"cache:  cache,",
			},
		},
		{
			name:       "no named dependencies - all default",
			structName: "SimpleService",
			fields: []detect.FieldInfo{
				{Name: "config", TypeExpr: &ast.Ident{Name: "Config"}},
			},
			dependencyNames: map[string]string{},
			expectedContains: []string{
				"func NewSimpleService(config Config) *SimpleService",
				"config: config,",
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

			injectorInfo := &registry.InjectorInfo{
				TypeName:  "example.com/service." + tt.structName,
				LocalName: tt.structName,
				OptionMetadata: detect.OptionMetadata{
					IsDefault: true,
				},
			}

			result, err := gen.GenerateConstructorWithNamedDeps(candidate, tt.fields, injectorInfo, tt.dependencyNames)
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

// TestConstructorGenerator_CombinedTypedAndNamed tests Typed[I] with Named dependencies
func TestConstructorGenerator_CombinedTypedAndNamed(t *testing.T) {
	gen := generate.NewConstructorGenerator()
	candidate := detect.ConstructorCandidate{
		TypeSpec: &ast.TypeSpec{
			Name: &ast.Ident{Name: "PaymentService"},
		},
	}

	fields := []detect.FieldInfo{
		{Name: "primaryRepo", TypeExpr: &ast.Ident{Name: "Repository"}},
		{Name: "cacheRepo", TypeExpr: &ast.Ident{Name: "Repository"}},
	}

	interfaceType := types.NewNamed(
		types.NewTypeName(token.NoPos, nil, "IPaymentService", nil),
		types.NewStruct(nil, nil),
		nil,
	)

	injectorInfo := &registry.InjectorInfo{
		TypeName:       "example.com/service.PaymentService",
		LocalName:      "PaymentService",
		RegisteredType: interfaceType,
		OptionMetadata: detect.OptionMetadata{
			TypedInterface: interfaceType,
		},
	}

	dependencyNames := map[string]string{
		"primaryRepo": "primary",
		"cacheRepo":   "cache",
	}

	result, err := gen.GenerateConstructorWithNamedDeps(candidate, fields, injectorInfo, dependencyNames)
	if err != nil {
		t.Fatalf("GenerateConstructorWithNamedDeps() error = %v", err)
	}

	expectedContains := []string{
		"func NewPaymentService(primary Repository, cache Repository) IPaymentService",
		"primaryRepo: primary,",
		"cacheRepo:   cache,",
		"return &PaymentService{",
	}

	for _, contains := range expectedContains {
		if !strings.Contains(result.Code, contains) {
			t.Errorf("Code missing expected content: %q\nGot:\n%s", contains, result.Code)
		}
	}
}
