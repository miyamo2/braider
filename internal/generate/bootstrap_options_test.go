package generate

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/miyamo2/braider/internal/graph"
	"golang.org/x/tools/go/analysis"
)

// TestBootstrapGenerator_InterfaceTypedVariables tests interface-typed variable declarations
func TestBootstrapGenerator_InterfaceTypedVariables(t *testing.T) {
	tests := []struct {
		name             string
		setupGraph       func() (*graph.Graph, []string)
		expectedContains []string
	}{
		{
			name: "Injectable[Typed[I]] declares interface variable",
			setupGraph: func() (*graph.Graph, []string) {
				// Create interface type
				interfaceType := types.NewNamed(
					types.NewTypeName(token.NoPos, nil, "IUserService", nil),
					types.NewStruct(nil, nil),
					nil,
				)

				// Create graph manually
				g := &graph.Graph{
					Nodes: map[string]*graph.Node{
						"example.com/service.UserService": {
							TypeName:        "example.com/service.UserService",
							PackageName:     "service",
							PackagePath:     "example.com/service",
							LocalName:       "UserService",
							ConstructorName: "NewUserService",
							IsField:         true,
							RegisteredType:  interfaceType,
							Dependencies:    []string{},
						},
					},
					Edges: map[string][]string{
						"example.com/service.UserService": {},
					},
				}

				sortedTypes := []string{"example.com/service.UserService"}
				return g, sortedTypes
			},
			expectedContains: []string{
				"userService IUserService",
				"userService := service.NewUserService()",
			},
		},
		{
			name: "Provide[Typed[I]] declares interface variable",
			setupGraph: func() (*graph.Graph, []string) {
				// Create interface type
				interfaceType := types.NewNamed(
					types.NewTypeName(token.NoPos, nil, "IRepository", nil),
					types.NewStruct(nil, nil),
					nil,
				)

				g := &graph.Graph{
					Nodes: map[string]*graph.Node{
						"example.com/repo.UserRepository": {
							TypeName:        "example.com/repo.UserRepository",
							PackageName:     "repo",
							PackagePath:     "example.com/repo",
							LocalName:       "UserRepository",
							ConstructorName: "NewUserRepository",
							IsField:         false,
							RegisteredType:  interfaceType,
							Dependencies:    []string{},
						},
						"example.com/service.UserService": {
							TypeName:        "example.com/service.UserService",
							PackageName:     "service",
							PackagePath:     "example.com/service",
							LocalName:       "UserService",
							ConstructorName: "NewUserService",
							IsField:         true,
							Dependencies:    []string{"example.com/repo.UserRepository"},
						},
					},
					Edges: map[string][]string{
						"example.com/repo.UserRepository":  {},
						"example.com/service.UserService": {"example.com/repo.UserRepository"},
					},
				}

				sortedTypes := []string{
					"example.com/repo.UserRepository",
					"example.com/service.UserService",
				}
				return g, sortedTypes
			},
			expectedContains: []string{
				"userRepository := repo.NewUserRepository()",
				"userService := service.NewUserService(userRepository)",
				"userService service.UserService",
			},
		},
		{
			name: "Mixed typed and untyped dependencies",
			setupGraph: func() (*graph.Graph, []string) {
				// Interface-typed dependency
				interfaceType := types.NewNamed(
					types.NewTypeName(token.NoPos, nil, "ILogger", nil),
					types.NewStruct(nil, nil),
					nil,
				)

				g := &graph.Graph{
					Nodes: map[string]*graph.Node{
						"example.com/logger.Logger": {
							TypeName:        "example.com/logger.Logger",
							PackageName:     "logger",
							PackagePath:     "example.com/logger",
							LocalName:       "Logger",
							ConstructorName: "NewLogger",
							IsField:         true,
							RegisteredType:  interfaceType,
							Dependencies:    []string{},
						},
						"example.com/cache.Cache": {
							TypeName:        "example.com/cache.Cache",
							PackageName:     "cache",
							PackagePath:     "example.com/cache",
							LocalName:       "Cache",
							ConstructorName: "NewCache",
							IsField:         true,
							RegisteredType:  nil,
							Dependencies:    []string{},
						},
					},
					Edges: map[string][]string{
						"example.com/logger.Logger": {},
						"example.com/cache.Cache":   {},
					},
				}

				sortedTypes := []string{
					"example.com/logger.Logger",
					"example.com/cache.Cache",
				}
				return g, sortedTypes
			},
			expectedContains: []string{
				"logger ILogger",
				"cache  cache.Cache",
				"logger := logger.NewLogger()",
				"cache := cache.NewCache()",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, sortedTypes := tt.setupGraph()

			bg := NewBootstrapGenerator()
			pass := &analysis.Pass{
				Pkg: types.NewPackage("example.com/main", "main"),
			}

			result, err := bg.GenerateBootstrap(pass, g, sortedTypes)
			if err != nil {
				t.Fatalf("GenerateBootstrap() error = %v", err)
			}

			for _, contains := range tt.expectedContains {
				if !strings.Contains(result.DependencyVar, contains) {
					t.Errorf("Code missing expected content: %q\nGot:\n%s", contains, result.DependencyVar)
				}
			}
		})
	}
}

// TestBootstrapGenerator_NamedVariableNaming tests named variable naming using info.Name field
func TestBootstrapGenerator_NamedVariableNaming(t *testing.T) {
	tests := []struct {
		name             string
		setupGraph       func() (*graph.Graph, []string)
		expectedContains []string
		notContains      []string
	}{
		{
			name: "Named dependency uses custom name",
			setupGraph: func() (*graph.Graph, []string) {
				g := &graph.Graph{
					Nodes: map[string]*graph.Node{
						"example.com/repo.Repository": {
							TypeName:        "example.com/repo.Repository",
							PackageName:     "repo",
							PackagePath:     "example.com/repo",
							LocalName:       "Repository",
							ConstructorName: "NewRepository",
							IsField:         true,
							Name:            "primaryRepo",
							Dependencies:    []string{},
						},
					},
					Edges: map[string][]string{
						"example.com/repo.Repository": {},
					},
				}

				sortedTypes := []string{"example.com/repo.Repository"}
				return g, sortedTypes
			},
			expectedContains: []string{
				"primaryRepo repo.Repository",
				"primaryRepo := repo.NewRepository()",
			},
			notContains: []string{
				"repository :=",
			},
		},
		{
			name: "Unnamed dependency uses derived name",
			setupGraph: func() (*graph.Graph, []string) {
				g := &graph.Graph{
					Nodes: map[string]*graph.Node{
						"example.com/service.UserService": {
							TypeName:        "example.com/service.UserService",
							PackageName:     "service",
							PackagePath:     "example.com/service",
							LocalName:       "UserService",
							ConstructorName: "NewUserService",
							IsField:         true,
							Name:            "",
							Dependencies:    []string{},
						},
					},
					Edges: map[string][]string{
						"example.com/service.UserService": {},
					},
				}

				sortedTypes := []string{"example.com/service.UserService"}
				return g, sortedTypes
			},
			expectedContains: []string{
				"userService service.UserService",
				"userService := service.NewUserService()",
			},
		},
		{
			name: "Named provider used in dependency",
			setupGraph: func() (*graph.Graph, []string) {
				g := &graph.Graph{
					Nodes: map[string]*graph.Node{
						"example.com/repo.Repository": {
							TypeName:        "example.com/repo.Repository",
							PackageName:     "repo",
							PackagePath:     "example.com/repo",
							LocalName:       "Repository",
							ConstructorName: "NewRepository",
							IsField:         false,
							Name:            "primaryRepo",
							Dependencies:    []string{},
						},
						"example.com/service.UserService": {
							TypeName:        "example.com/service.UserService",
							PackageName:     "service",
							PackagePath:     "example.com/service",
							LocalName:       "UserService",
							ConstructorName: "NewUserService",
							IsField:         true,
							Name:            "",
							Dependencies:    []string{"example.com/repo.Repository"},
						},
					},
					Edges: map[string][]string{
						"example.com/repo.Repository":      {},
						"example.com/service.UserService": {"example.com/repo.Repository"},
					},
				}

				sortedTypes := []string{
					"example.com/repo.Repository",
					"example.com/service.UserService",
				}
				return g, sortedTypes
			},
			expectedContains: []string{
				"primaryRepo := repo.NewRepository()",
				"userService := service.NewUserService(primaryRepo)",
				"userService service.UserService",
			},
			notContains: []string{
				"repository :=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, sortedTypes := tt.setupGraph()

			bg := NewBootstrapGenerator()
			pass := &analysis.Pass{
				Pkg: types.NewPackage("example.com/main", "main"),
			}

			result, err := bg.GenerateBootstrap(pass, g, sortedTypes)
			if err != nil {
				t.Fatalf("GenerateBootstrap() error = %v", err)
			}

			for _, contains := range tt.expectedContains {
				if !strings.Contains(result.DependencyVar, contains) {
					t.Errorf("Code missing expected content: %q\nGot:\n%s", contains, result.DependencyVar)
				}
			}

			for _, notContain := range tt.notContains {
				if strings.Contains(result.DependencyVar, notContain) {
					t.Errorf("Code should not contain: %q\nGot:\n%s", notContain, result.DependencyVar)
				}
			}
		})
	}
}

// TestBootstrapGenerator_TopologicalSortPreservation tests topological sort for typed and named
func TestBootstrapGenerator_TopologicalSortPreservation(t *testing.T) {
	tests := []struct {
		name             string
		setupGraph       func() (*graph.Graph, []string)
		expectedContains []string
	}{
		{
			name: "Typed dependencies in correct order",
			setupGraph: func() (*graph.Graph, []string) {
				interfaceType := types.NewNamed(
					types.NewTypeName(token.NoPos, nil, "IRepository", nil),
					types.NewStruct(nil, nil),
					nil,
				)

				g := &graph.Graph{
					Nodes: map[string]*graph.Node{
						"example.com/repo.Repository": {
							TypeName:        "example.com/repo.Repository",
							PackageName:     "repo",
							PackagePath:     "example.com/repo",
							LocalName:       "Repository",
							ConstructorName: "NewRepository",
							IsField:         true,
							RegisteredType:  interfaceType,
							Dependencies:    []string{},
						},
						"example.com/service.UserService": {
							TypeName:        "example.com/service.UserService",
							PackageName:     "service",
							PackagePath:     "example.com/service",
							LocalName:       "UserService",
							ConstructorName: "NewUserService",
							IsField:         true,
							Dependencies:    []string{"example.com/repo.Repository"},
						},
					},
					Edges: map[string][]string{
						"example.com/repo.Repository":      {},
						"example.com/service.UserService": {"example.com/repo.Repository"},
					},
				}

				sortedTypes := []string{
					"example.com/repo.Repository",
					"example.com/service.UserService",
				}
				return g, sortedTypes
			},
			expectedContains: []string{
				"repository := repo.NewRepository()",
				"userService := service.NewUserService(repository)",
			},
		},
		{
			name: "Named dependencies preserve initialization order",
			setupGraph: func() (*graph.Graph, []string) {
				g := &graph.Graph{
					Nodes: map[string]*graph.Node{
						"example.com/repo.Repository": {
							TypeName:        "example.com/repo.Repository",
							PackageName:     "repo",
							PackagePath:     "example.com/repo",
							LocalName:       "Repository",
							ConstructorName: "NewRepository",
							IsField:         true,
							Name:            "primary",
							Dependencies:    []string{},
						},
					},
					Edges: map[string][]string{
						"example.com/repo.Repository": {},
					},
				}

				sortedTypes := []string{"example.com/repo.Repository"}
				return g, sortedTypes
			},
			expectedContains: []string{
				"primary := repo.NewRepository()",
				"primary repo.Repository",
			},
		},
		{
			name: "Complex dependency chain with mixed types",
			setupGraph: func() (*graph.Graph, []string) {
				interfaceType := types.NewNamed(
					types.NewTypeName(token.NoPos, nil, "IRepository", nil),
					types.NewStruct(nil, nil),
					nil,
				)

				g := &graph.Graph{
					Nodes: map[string]*graph.Node{
						"example.com/logger.Logger": {
							TypeName:        "example.com/logger.Logger",
							PackageName:     "logger",
							PackagePath:     "example.com/logger",
							LocalName:       "Logger",
							ConstructorName: "NewLogger",
							IsField:         true,
							Dependencies:    []string{},
						},
						"example.com/repo.Repository": {
							TypeName:        "example.com/repo.Repository",
							PackageName:     "repo",
							PackagePath:     "example.com/repo",
							LocalName:       "Repository",
							ConstructorName: "NewRepository",
							IsField:         true,
							RegisteredType:  interfaceType,
							Name:            "mainRepo",
							Dependencies:    []string{"example.com/logger.Logger"},
						},
						"example.com/service.UserService": {
							TypeName:        "example.com/service.UserService",
							PackageName:     "service",
							PackagePath:     "example.com/service",
							LocalName:       "UserService",
							ConstructorName: "NewUserService",
							IsField:         true,
							Dependencies:    []string{"example.com/repo.Repository"},
						},
					},
					Edges: map[string][]string{
						"example.com/logger.Logger":        {},
						"example.com/repo.Repository":      {"example.com/logger.Logger"},
						"example.com/service.UserService": {"example.com/repo.Repository"},
					},
				}

				sortedTypes := []string{
					"example.com/logger.Logger",
					"example.com/repo.Repository",
					"example.com/service.UserService",
				}
				return g, sortedTypes
			},
			expectedContains: []string{
				"logger := logger.NewLogger()",
				"mainRepo := repo.NewRepository(logger)",
				"userService := service.NewUserService(mainRepo)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, sortedTypes := tt.setupGraph()

			bg := NewBootstrapGenerator()
			pass := &analysis.Pass{
				Pkg: types.NewPackage("example.com/main", "main"),
			}

			result, err := bg.GenerateBootstrap(pass, g, sortedTypes)
			if err != nil {
				t.Fatalf("GenerateBootstrap() error = %v", err)
			}

			for _, contains := range tt.expectedContains {
				if !strings.Contains(result.DependencyVar, contains) {
					t.Errorf("Code missing expected content: %q\nGot:\n%s", contains, result.DependencyVar)
				}
			}
		})
	}
}
