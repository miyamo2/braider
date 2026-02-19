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
			builder := NewDependencyGraphBuilder(NewInterfaceRegistry())
			graph, err := builder.BuildGraph(pass, tt.providers, tt.injectors, nil)

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
			name: "provider has IsField=true",
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
				"example.com/repo.UserRepository": true,
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
				"example.com/repo.UserRepository": true,
				"example.com/service.UserService": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass := createMockPassForGraph(t)

			builder := NewDependencyGraphBuilder(NewInterfaceRegistry())
			graph, err := builder.BuildGraph(pass, tt.providers, tt.injectors, nil)
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

			builder := NewDependencyGraphBuilder(NewInterfaceRegistry())
			graph, err := builder.BuildGraph(pass, tt.providers, tt.injectors, nil)

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

// TestUnresolvableTypeError_Error tests the Error() method of UnresolvableTypeError.
func TestUnresolvableTypeError_Error(t *testing.T) {
	err := &UnresolvableTypeError{TypeName: "example.com/pkg.MissingType"}
	want := "unresolvable dependency type: example.com/pkg.MissingType"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

// TestDependencyGraph_BuildGraph_PointerDependencies tests that pointer-type dependencies are
// resolved by stripping the "*" prefix and matching against the concrete type node.
func TestDependencyGraph_BuildGraph_PointerDependencies(t *testing.T) {
	tests := []struct {
		name      string
		providers []*registry.ProviderInfo
		injectors []*registry.InjectorInfo
		wantEdges map[string][]string
		wantErr   bool
	}{
		{
			name: "pointer dependency resolved to concrete node",
			providers: []*registry.ProviderInfo{
				{
					TypeName:        "example.com/repo.UserRepository",
					PackagePath:     "example.com/repo",
					PackageName:     "repo",
					LocalName:       "UserRepository",
					ConstructorName: "NewUserRepository",
					Dependencies:    []string{},
				},
			},
			injectors: []*registry.InjectorInfo{
				{
					TypeName:        "example.com/service.UserService",
					PackagePath:     "example.com/service",
					PackageName:     "service",
					LocalName:       "UserService",
					ConstructorName: "NewUserService",
					Dependencies:    []string{"*example.com/repo.UserRepository"},
				},
			},
			wantEdges: map[string][]string{
				"example.com/repo.UserRepository": {},
				"example.com/service.UserService": {"example.com/repo.UserRepository"},
			},
			wantErr: false,
		},
		{
			name: "pointer dependency not in graph remains unresolvable",
			injectors: []*registry.InjectorInfo{
				{
					TypeName:        "example.com/service.UserService",
					PackagePath:     "example.com/service",
					PackageName:     "service",
					LocalName:       "UserService",
					ConstructorName: "NewUserService",
					Dependencies:    []string{"*example.com/external.Unknown"},
				},
			},
			wantErr: true,
		},
		{
			name: "non-pointer dependency still works",
			providers: []*registry.ProviderInfo{
				{
					TypeName:        "example.com/repo.UserRepository",
					PackagePath:     "example.com/repo",
					PackageName:     "repo",
					LocalName:       "UserRepository",
					ConstructorName: "NewUserRepository",
					Dependencies:    []string{},
				},
			},
			injectors: []*registry.InjectorInfo{
				{
					TypeName:        "example.com/service.UserService",
					PackagePath:     "example.com/service",
					PackageName:     "service",
					LocalName:       "UserService",
					ConstructorName: "NewUserService",
					Dependencies:    []string{"example.com/repo.UserRepository"},
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
			builder := NewDependencyGraphBuilder(NewInterfaceRegistry())
			graph, err := builder.BuildGraph(pass, tt.providers, tt.injectors, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("BuildGraph() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
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

// TestDependencyGraph_BuildGraph_NamedDependencies tests graph construction with named dependencies.
func TestDependencyGraph_BuildGraph_NamedDependencies(t *testing.T) {
	// Scenario: Two instances of Repository with different names, Service depends on one
	providers := []*registry.ProviderInfo{
		{
			TypeName:        "example.com/repo.Repository",
			PackagePath:     "example.com/repo",
			PackageName:     "repo",
			LocalName:       "Repository",
			ConstructorName: "NewRepository",
			Dependencies:    []string{},
			Name:            "primaryRepo",
		},
		{
			TypeName:        "example.com/repo.Repository",
			PackagePath:     "example.com/repo",
			PackageName:     "repo",
			LocalName:       "Repository",
			ConstructorName: "NewRepository",
			Dependencies:    []string{},
			Name:            "secondaryRepo",
		},
	}

	injectors := []*registry.InjectorInfo{
		{
			TypeName:        "example.com/service.Service",
			PackagePath:     "example.com/service",
			PackageName:     "service",
			LocalName:       "Service",
			ConstructorName: "NewService",
			Dependencies:    []string{"example.com/repo.Repository#primaryRepo"},
		},
	}

	builder := NewDependencyGraphBuilder(NewInterfaceRegistry())
	pass := createMockPassForGraph(t)

	graph, err := builder.BuildGraph(pass, providers, injectors, nil)
	if err != nil {
		t.Fatalf("BuildGraph() unexpected error = %v", err)
	}

	// Check that both named Repository instances are in the graph with composite keys
	expectedKeys := []string{
		"example.com/repo.Repository#primaryRepo",
		"example.com/repo.Repository#secondaryRepo",
		"example.com/service.Service",
	}

	for _, key := range expectedKeys {
		if _, exists := graph.Nodes[key]; !exists {
			t.Errorf("Expected node %s not found in graph", key)
		}
	}

	// Check that Service depends on the named Repository
	serviceEdges := graph.Edges["example.com/service.Service"]
	if len(serviceEdges) != 1 {
		t.Fatalf("Service should have 1 dependency, got %d", len(serviceEdges))
	}
	if serviceEdges[0] != "example.com/repo.Repository#primaryRepo" {
		t.Errorf("Service should depend on primaryRepo, got %s", serviceEdges[0])
	}

	// Check Name field is preserved
	primaryRepoNode := graph.Nodes["example.com/repo.Repository#primaryRepo"]
	if primaryRepoNode.Name != "primaryRepo" {
		t.Errorf("primaryRepo node should have Name=primaryRepo, got %s", primaryRepoNode.Name)
	}

	secondaryRepoNode := graph.Nodes["example.com/repo.Repository#secondaryRepo"]
	if secondaryRepoNode.Name != "secondaryRepo" {
		t.Errorf("secondaryRepo node should have Name=secondaryRepo, got %s", secondaryRepoNode.Name)
	}
}

// TestNode_VariableMetadataFields tests that Node struct supports Variable metadata fields.
func TestNode_VariableMetadataFields(t *testing.T) {
	t.Run("ExpressionText field stores expression source text", func(t *testing.T) {
		node := &Node{
			TypeName:       "os.File",
			ExpressionText: "os.Stdout",
		}
		if node.ExpressionText != "os.Stdout" {
			t.Errorf("ExpressionText = %q, want %q", node.ExpressionText, "os.Stdout")
		}
	})

	t.Run("ExpressionPkgs field stores package paths", func(t *testing.T) {
		node := &Node{
			TypeName:       "os.File",
			ExpressionPkgs: []string{"os"},
		}
		if len(node.ExpressionPkgs) != 1 || node.ExpressionPkgs[0] != "os" {
			t.Errorf("ExpressionPkgs = %v, want [\"os\"]", node.ExpressionPkgs)
		}
	})

	t.Run("IsQualified field stores qualification status", func(t *testing.T) {
		node := &Node{
			TypeName:    "os.File",
			IsQualified: true,
		}
		if !node.IsQualified {
			t.Error("IsQualified should be true")
		}
	})

	t.Run("Variable metadata fields default to zero values for non-Variable nodes", func(t *testing.T) {
		node := &Node{
			TypeName:        "example.com/repo.UserRepository",
			PackagePath:     "example.com/repo",
			PackageName:     "repo",
			LocalName:       "UserRepository",
			ConstructorName: "NewUserRepository",
			Dependencies:    []string{},
			InDegree:        0,
			IsField:         false,
		}
		if node.ExpressionText != "" {
			t.Errorf("ExpressionText should be empty for non-Variable node, got %q", node.ExpressionText)
		}
		if node.ExpressionPkgs != nil {
			t.Errorf("ExpressionPkgs should be nil for non-Variable node, got %v", node.ExpressionPkgs)
		}
		if node.IsQualified {
			t.Error("IsQualified should be false for non-Variable node")
		}
	})
}

// TestDependencyGraph_BuildGraph_VariableNodes tests graph construction with Variable nodes.
func TestDependencyGraph_BuildGraph_VariableNodes(t *testing.T) {
	t.Run("single variable with no dependencies creates node", func(t *testing.T) {
		pass := createMockPassForGraph(t)
		builder := NewDependencyGraphBuilder(NewInterfaceRegistry())

		variables := []*registry.VariableInfo{
			{
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				ExpressionText: "os.Stdout",
				ExpressionPkgs: []string{"os"},
				IsQualified:    true,
				Dependencies:   []string{},
			},
		}

		graph, err := builder.BuildGraph(pass, nil, nil, variables)
		if err != nil {
			t.Fatalf("BuildGraph() error = %v", err)
		}

		node, ok := graph.Nodes["os.File"]
		if !ok {
			t.Fatal("expected Variable node 'os.File' in graph")
		}
		if node.TypeName != "os.File" {
			t.Errorf("TypeName = %q, want %q", node.TypeName, "os.File")
		}
		if node.PackagePath != "os" {
			t.Errorf("PackagePath = %q, want %q", node.PackagePath, "os")
		}
		if node.PackageName != "os" {
			t.Errorf("PackageName = %q, want %q", node.PackageName, "os")
		}
		if node.LocalName != "File" {
			t.Errorf("LocalName = %q, want %q", node.LocalName, "File")
		}
		if node.ConstructorName != "" {
			t.Errorf("ConstructorName should be empty for Variable node, got %q", node.ConstructorName)
		}
		if node.IsField {
			t.Error("IsField should be false for Variable node")
		}
		if node.ExpressionText != "os.Stdout" {
			t.Errorf("ExpressionText = %q, want %q", node.ExpressionText, "os.Stdout")
		}
		if len(node.ExpressionPkgs) != 1 || node.ExpressionPkgs[0] != "os" {
			t.Errorf("ExpressionPkgs = %v, want [\"os\"]", node.ExpressionPkgs)
		}
		if !node.IsQualified {
			t.Error("IsQualified should be true")
		}
		if node.InDegree != 0 {
			t.Errorf("InDegree = %d, want 0 (Variables have no dependencies)", node.InDegree)
		}
		if len(node.Dependencies) != 0 {
			t.Errorf("Dependencies = %v, want empty", node.Dependencies)
		}
	})

	t.Run("named variable uses composite key", func(t *testing.T) {
		pass := createMockPassForGraph(t)
		builder := NewDependencyGraphBuilder(NewInterfaceRegistry())

		variables := []*registry.VariableInfo{
			{
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				ExpressionText: "os.Stdout",
				ExpressionPkgs: []string{"os"},
				IsQualified:    true,
				Dependencies:   []string{},
				Name:           "output",
			},
		}

		graph, err := builder.BuildGraph(pass, nil, nil, variables)
		if err != nil {
			t.Fatalf("BuildGraph() error = %v", err)
		}

		// Should use composite key "TypeName#Name"
		node, ok := graph.Nodes["os.File#output"]
		if !ok {
			t.Fatal("expected Variable node 'os.File#output' in graph")
		}
		if node.Name != "output" {
			t.Errorf("Name = %q, want %q", node.Name, "output")
		}
	})

	t.Run("variable coexists with provider and injector nodes", func(t *testing.T) {
		pass := createMockPassForGraph(t)
		builder := NewDependencyGraphBuilder(NewInterfaceRegistry())

		providers := []*registry.ProviderInfo{
			{
				TypeName:        "example.com/repo.UserRepository",
				PackagePath:     "example.com/repo",
				PackageName:     "repo",
				LocalName:       "UserRepository",
				ConstructorName: "NewUserRepository",
				Dependencies:    []string{},
			},
		}
		injectors := []*registry.InjectorInfo{
			{
				TypeName:        "example.com/service.UserService",
				PackagePath:     "example.com/service",
				PackageName:     "service",
				LocalName:       "UserService",
				ConstructorName: "NewUserService",
				Dependencies:    []string{"example.com/repo.UserRepository"},
			},
		}
		variables := []*registry.VariableInfo{
			{
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				ExpressionText: "os.Stdout",
				ExpressionPkgs: []string{"os"},
				IsQualified:    true,
				Dependencies:   []string{},
			},
		}

		graph, err := builder.BuildGraph(pass, providers, injectors, variables)
		if err != nil {
			t.Fatalf("BuildGraph() error = %v", err)
		}

		// All three node types should be present
		if len(graph.Nodes) != 3 {
			t.Errorf("expected 3 nodes, got %d", len(graph.Nodes))
		}

		// Provider node
		if _, ok := graph.Nodes["example.com/repo.UserRepository"]; !ok {
			t.Error("expected provider node")
		}
		// Injector node
		if _, ok := graph.Nodes["example.com/service.UserService"]; !ok {
			t.Error("expected injector node")
		}
		// Variable node
		varNode, ok := graph.Nodes["os.File"]
		if !ok {
			t.Error("expected variable node")
		} else {
			if varNode.IsField {
				t.Error("variable node should have IsField=false")
			}
			if varNode.ExpressionText != "os.Stdout" {
				t.Errorf("variable node ExpressionText = %q, want %q", varNode.ExpressionText, "os.Stdout")
			}
		}
	})

	t.Run("variable with RegisteredType preserves the field", func(t *testing.T) {
		pass := createMockPassForGraph(t)
		builder := NewDependencyGraphBuilder(NewInterfaceRegistry())

		variables := []*registry.VariableInfo{
			{
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				ExpressionText: "os.Stdout",
				Dependencies:   []string{},
				// RegisteredType would be set for Typed[I] but nil here for default
			},
		}

		graph, err := builder.BuildGraph(pass, nil, nil, variables)
		if err != nil {
			t.Fatalf("BuildGraph() error = %v", err)
		}

		node := graph.Nodes["os.File"]
		if node == nil {
			t.Fatal("expected Variable node in graph")
		}
		// RegisteredType should be nil if not set (default Variable)
		if node.RegisteredType != nil {
			t.Errorf("RegisteredType should be nil for default Variable, got %v", node.RegisteredType)
		}
	})

	t.Run("backward compatibility: nil variables parameter works", func(t *testing.T) {
		pass := createMockPassForGraph(t)
		builder := NewDependencyGraphBuilder(NewInterfaceRegistry())

		providers := []*registry.ProviderInfo{
			{
				TypeName:        "example.com/repo.UserRepository",
				PackagePath:     "example.com/repo",
				PackageName:     "repo",
				LocalName:       "UserRepository",
				ConstructorName: "NewUserRepository",
				Dependencies:    []string{},
			},
		}

		graph, err := builder.BuildGraph(pass, providers, nil, nil)
		if err != nil {
			t.Fatalf("BuildGraph() error = %v", err)
		}

		if len(graph.Nodes) != 1 {
			t.Errorf("expected 1 node, got %d", len(graph.Nodes))
		}
	})

	t.Run("variable node has zero edges", func(t *testing.T) {
		pass := createMockPassForGraph(t)
		builder := NewDependencyGraphBuilder(NewInterfaceRegistry())

		variables := []*registry.VariableInfo{
			{
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				ExpressionText: "os.Stdout",
				Dependencies:   []string{},
			},
		}

		graph, err := builder.BuildGraph(pass, nil, nil, variables)
		if err != nil {
			t.Fatalf("BuildGraph() error = %v", err)
		}

		edges := graph.Edges["os.File"]
		if len(edges) != 0 {
			t.Errorf("Variable node should have 0 edges, got %d", len(edges))
		}
	})

	t.Run("injector depends on variable node", func(t *testing.T) {
		pass := createMockPassForGraph(t)
		builder := NewDependencyGraphBuilder(NewInterfaceRegistry())

		variables := []*registry.VariableInfo{
			{
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				ExpressionText: "os.Stdout",
				Dependencies:   []string{},
			},
		}
		injectors := []*registry.InjectorInfo{
			{
				TypeName:        "example.com/service.Service",
				PackagePath:     "example.com/service",
				PackageName:     "service",
				LocalName:       "Service",
				ConstructorName: "NewService",
				Dependencies:    []string{"os.File"},
			},
		}

		graph, err := builder.BuildGraph(pass, nil, injectors, variables)
		if err != nil {
			t.Fatalf("BuildGraph() error = %v", err)
		}

		// Service should depend on the Variable node
		serviceEdges := graph.Edges["example.com/service.Service"]
		if len(serviceEdges) != 1 {
			t.Fatalf("Service should have 1 dependency, got %d", len(serviceEdges))
		}
		if serviceEdges[0] != "os.File" {
			t.Errorf("Service should depend on os.File, got %s", serviceEdges[0])
		}

		// Variable node should have InDegree=0
		varNode := graph.Nodes["os.File"]
		if varNode.InDegree != 0 {
			t.Errorf("Variable InDegree = %d, want 0", varNode.InDegree)
		}

		// Service node should have InDegree=1
		serviceNode := graph.Nodes["example.com/service.Service"]
		if serviceNode.InDegree != 1 {
			t.Errorf("Service InDegree = %d, want 1", serviceNode.InDegree)
		}
	})
}
