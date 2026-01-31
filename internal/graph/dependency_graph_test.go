package graph

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/registry"
	"golang.org/x/tools/go/analysis"
)

// TestDependencyGraph_BuildGraph tests the BuildGraph method for constructing the dependency graph.
func TestDependencyGraph_BuildGraph(t *testing.T) {
	tests := []struct {
		name      string
		providers []*registry.ProviderInfo
		injectors []*registry.InjectorInfo
		wantNodes map[string]bool     // type names that should be in graph
		wantEdges map[string][]string // from -> []to
		wantErr   bool
	}{
		{
			name: "single provider with no dependencies",
			providers: []*registry.ProviderInfo{
				{
					TypeName:        "example.com/repo.UserRepository",
					PackagePath:     "example.com/repo",
					LocalName:       "UserRepository",
					ConstructorName: "NewUserRepository",
					Dependencies:    []string{},
					IsPending:       false,
				},
			},
			injectors: nil,
			wantNodes: map[string]bool{
				"example.com/repo.UserRepository": true,
			},
			wantEdges: map[string][]string{
				"example.com/repo.UserRepository": {},
			},
			wantErr: false,
		},
		{
			name:      "single injector with no dependencies",
			providers: nil,
			injectors: []*registry.InjectorInfo{
				{
					TypeName:        "example.com/service.UserService",
					PackagePath:     "example.com/service",
					LocalName:       "UserService",
					ConstructorName: "NewUserService",
					Dependencies:    []string{},
					IsPending:       true,
				},
			},
			wantNodes: map[string]bool{
				"example.com/service.UserService": true,
			},
			wantEdges: map[string][]string{
				"example.com/service.UserService": {},
			},
			wantErr: false,
		},
		{
			name: "provider depends on injector",
			providers: []*registry.ProviderInfo{
				{
					TypeName:        "example.com/repo.UserRepository",
					PackagePath:     "example.com/repo",
					LocalName:       "UserRepository",
					ConstructorName: "NewUserRepository",
					Dependencies:    []string{},
					IsPending:       false,
				},
			},
			injectors: []*registry.InjectorInfo{
				{
					TypeName:        "example.com/service.UserService",
					PackagePath:     "example.com/service",
					LocalName:       "UserService",
					ConstructorName: "NewUserService",
					Dependencies:    []string{"example.com/repo.UserRepository"},
					IsPending:       true,
				},
			},
			wantNodes: map[string]bool{
				"example.com/repo.UserRepository": true,
				"example.com/service.UserService": true,
			},
			wantEdges: map[string][]string{
				"example.com/repo.UserRepository": {},
				"example.com/service.UserService": {"example.com/repo.UserRepository"},
			},
			wantErr: false,
		},
		{
			name: "chain of dependencies",
			providers: []*registry.ProviderInfo{
				{
					TypeName:        "example.com/repo.UserRepository",
					PackagePath:     "example.com/repo",
					LocalName:       "UserRepository",
					ConstructorName: "NewUserRepository",
					Dependencies:    []string{},
					IsPending:       false,
				},
			},
			injectors: []*registry.InjectorInfo{
				{
					TypeName:        "example.com/service.UserService",
					PackagePath:     "example.com/service",
					LocalName:       "UserService",
					ConstructorName: "NewUserService",
					Dependencies:    []string{"example.com/repo.UserRepository"},
					IsPending:       true,
				},
				{
					TypeName:        "example.com/handler.UserHandler",
					PackagePath:     "example.com/handler",
					LocalName:       "UserHandler",
					ConstructorName: "NewUserHandler",
					Dependencies:    []string{"example.com/service.UserService"},
					IsPending:       true,
				},
			},
			wantNodes: map[string]bool{
				"example.com/repo.UserRepository": true,
				"example.com/service.UserService": true,
				"example.com/handler.UserHandler": true,
			},
			wantEdges: map[string][]string{
				"example.com/repo.UserRepository": {},
				"example.com/service.UserService": {"example.com/repo.UserRepository"},
				"example.com/handler.UserHandler": {"example.com/service.UserService"},
			},
			wantErr: false,
		},
		{
			name: "multiple dependencies",
			providers: []*registry.ProviderInfo{
				{
					TypeName:        "example.com/repo.UserRepository",
					PackagePath:     "example.com/repo",
					LocalName:       "UserRepository",
					ConstructorName: "NewUserRepository",
					Dependencies:    []string{},
					IsPending:       false,
				},
				{
					TypeName:        "example.com/repo.OrderRepository",
					PackagePath:     "example.com/repo",
					LocalName:       "OrderRepository",
					ConstructorName: "NewOrderRepository",
					Dependencies:    []string{},
					IsPending:       false,
				},
			},
			injectors: []*registry.InjectorInfo{
				{
					TypeName:        "example.com/service.OrderService",
					PackagePath:     "example.com/service",
					LocalName:       "OrderService",
					ConstructorName: "NewOrderService",
					Dependencies: []string{
						"example.com/repo.UserRepository",
						"example.com/repo.OrderRepository",
					},
					IsPending: true,
				},
			},
			wantNodes: map[string]bool{
				"example.com/repo.UserRepository":  true,
				"example.com/repo.OrderRepository": true,
				"example.com/service.OrderService": true,
			},
			wantEdges: map[string][]string{
				"example.com/repo.UserRepository":  {},
				"example.com/repo.OrderRepository": {},
				"example.com/service.OrderService": {
					"example.com/repo.UserRepository",
					"example.com/repo.OrderRepository",
				},
			},
			wantErr: false,
		},
		{
			name:      "empty graph",
			providers: []*registry.ProviderInfo{},
			injectors: []*registry.InjectorInfo{},
			wantNodes: map[string]bool{},
			wantEdges: map[string][]string{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock pass
			pass := createMockPassForGraph(t)

			// Build graph
			builder := NewDependencyGraphBuilder()
			graph, err := builder.BuildGraph(pass, tt.providers, tt.injectors)

			if (err != nil) != tt.wantErr {
				t.Errorf("BuildGraph() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Check nodes
			if len(graph.Nodes) != len(tt.wantNodes) {
				t.Errorf("BuildGraph() nodes count = %d, want %d", len(graph.Nodes), len(tt.wantNodes))
			}
			for typeName := range tt.wantNodes {
				if _, ok := graph.Nodes[typeName]; !ok {
					t.Errorf("BuildGraph() missing node %s", typeName)
				}
			}

			// Check edges
			if len(graph.Edges) != len(tt.wantEdges) {
				t.Errorf("BuildGraph() edges count = %d, want %d", len(graph.Edges), len(tt.wantEdges))
			}
			for from, wantTos := range tt.wantEdges {
				gotTos, ok := graph.Edges[from]
				if !ok {
					t.Errorf("BuildGraph() missing edge from %s", from)
					continue
				}
				if len(gotTos) != len(wantTos) {
					t.Errorf("BuildGraph() edge %s -> count = %d, want %d", from, len(gotTos), len(wantTos))
					continue
				}
				for i, wantTo := range wantTos {
					if gotTos[i] != wantTo {
						t.Errorf("BuildGraph() edge %s -> [%d] = %s, want %s", from, i, gotTos[i], wantTo)
					}
				}
			}
		})
	}
}

// TestDependencyGraph_IsField tests the IsField flag in nodes.
func TestDependencyGraph_IsField(t *testing.T) {
	tests := []struct {
		name      string
		providers []*registry.ProviderInfo
		injectors []*registry.InjectorInfo
		wantFlags map[string]bool // type name -> IsField
	}{
		{
			name: "provider has IsField=false",
			providers: []*registry.ProviderInfo{
				{
					TypeName:        "example.com/repo.UserRepository",
					PackagePath:     "example.com/repo",
					LocalName:       "UserRepository",
					ConstructorName: "NewUserRepository",
					Dependencies:    []string{},
					IsPending:       false,
				},
			},
			injectors: nil,
			wantFlags: map[string]bool{
				"example.com/repo.UserRepository": false,
			},
		},
		{
			name:      "injector has IsField=true",
			providers: nil,
			injectors: []*registry.InjectorInfo{
				{
					TypeName:        "example.com/service.UserService",
					PackagePath:     "example.com/service",
					LocalName:       "UserService",
					ConstructorName: "NewUserService",
					Dependencies:    []string{},
					IsPending:       true,
				},
			},
			wantFlags: map[string]bool{
				"example.com/service.UserService": true,
			},
		},
		{
			name: "mixed providers and injectors",
			providers: []*registry.ProviderInfo{
				{
					TypeName:        "example.com/repo.UserRepository",
					PackagePath:     "example.com/repo",
					LocalName:       "UserRepository",
					ConstructorName: "NewUserRepository",
					Dependencies:    []string{},
					IsPending:       false,
				},
			},
			injectors: []*registry.InjectorInfo{
				{
					TypeName:        "example.com/service.UserService",
					PackagePath:     "example.com/service",
					LocalName:       "UserService",
					ConstructorName: "NewUserService",
					Dependencies:    []string{"example.com/repo.UserRepository"},
					IsPending:       true,
				},
			},
			wantFlags: map[string]bool{
				"example.com/repo.UserRepository": false,
				"example.com/service.UserService": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass := createMockPassForGraph(t)

			builder := NewDependencyGraphBuilder()
			graph, err := builder.BuildGraph(pass, tt.providers, tt.injectors)
			if err != nil {
				t.Fatalf("BuildGraph() error = %v", err)
			}

			for typeName, wantIsField := range tt.wantFlags {
				node, ok := graph.Nodes[typeName]
				if !ok {
					t.Errorf("BuildGraph() missing node %s", typeName)
					continue
				}
				if node.IsField != wantIsField {
					t.Errorf("BuildGraph() node %s IsField = %v, want %v", typeName, node.IsField, wantIsField)
				}
			}
		})
	}
}

// TestDependencyGraph_InterfaceResolution tests interface dependency resolution.
func TestDependencyGraph_InterfaceResolution(t *testing.T) {
	tests := []struct {
		name      string
		providers []*registry.ProviderInfo
		injectors []*registry.InjectorInfo
		wantEdges map[string][]string
		wantErr   bool
	}{
		{
			name: "interface resolved to provider",
			providers: []*registry.ProviderInfo{
				{
					TypeName:        "example.com/repo.UserRepository",
					PackagePath:     "example.com/repo",
					LocalName:       "UserRepository",
					ConstructorName: "NewUserRepository",
					Dependencies:    []string{},
					Implements:      []string{"example.com/domain.IUserRepository"},
					IsPending:       false,
				},
			},
			injectors: []*registry.InjectorInfo{
				{
					TypeName:        "example.com/service.UserService",
					PackagePath:     "example.com/service",
					LocalName:       "UserService",
					ConstructorName: "NewUserService",
					Dependencies:    []string{"example.com/domain.IUserRepository"},
					IsPending:       true,
				},
			},
			wantEdges: map[string][]string{
				"example.com/repo.UserRepository": {},
				"example.com/service.UserService": {"example.com/repo.UserRepository"},
			},
			wantErr: false,
		},
		{
			name:      "interface resolved to injector",
			providers: nil,
			injectors: []*registry.InjectorInfo{
				{
					TypeName:        "example.com/repo.UserRepository",
					PackagePath:     "example.com/repo",
					LocalName:       "UserRepository",
					ConstructorName: "NewUserRepository",
					Dependencies:    []string{},
					Implements:      []string{"example.com/domain.IUserRepository"},
					IsPending:       true,
				},
				{
					TypeName:        "example.com/service.UserService",
					PackagePath:     "example.com/service",
					LocalName:       "UserService",
					ConstructorName: "NewUserService",
					Dependencies:    []string{"example.com/domain.IUserRepository"},
					IsPending:       true,
				},
			},
			wantEdges: map[string][]string{
				"example.com/repo.UserRepository": {},
				"example.com/service.UserService": {"example.com/repo.UserRepository"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass := createMockPassForGraph(t)

			builder := NewDependencyGraphBuilder()
			graph, err := builder.BuildGraph(pass, tt.providers, tt.injectors)

			if (err != nil) != tt.wantErr {
				t.Errorf("BuildGraph() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Check edges
			for from, wantTos := range tt.wantEdges {
				gotTos, ok := graph.Edges[from]
				if !ok {
					t.Errorf("BuildGraph() missing edge from %s", from)
					continue
				}
				if len(gotTos) != len(wantTos) {
					t.Errorf("BuildGraph() edge %s -> count = %d, want %d", from, len(gotTos), len(wantTos))
					continue
				}
				for i, wantTo := range wantTos {
					if gotTos[i] != wantTo {
						t.Errorf("BuildGraph() edge %s -> [%d] = %s, want %s", from, i, gotTos[i], wantTo)
					}
				}
			}
		})
	}
}

// createMockPassForGraph creates a minimal mock analysis.Pass for graph tests.
func createMockPassForGraph(t *testing.T) *analysis.Pass {
	t.Helper()

	src := `package test`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	conf := types.Config{}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	pkg, _ := conf.Check("test", fset, []*ast.File{file}, info)

	return &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       pkg,
		TypesInfo: info,
	}
}
