package generate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/miyamo2/braider/internal/graph"
	"golang.org/x/tools/go/analysis"
)

func TestBootstrapGenerator_GenerateBootstrap(t *testing.T) {
	tests := []struct {
		name        string
		graph       *graph.Graph
		sortedTypes []string
		wantErr     bool
		checkOutput func(t *testing.T, bootstrap *GeneratedBootstrap)
	}{
		{
			name:        "empty graph",
			graph:       nil,
			sortedTypes: nil,
			wantErr:     true,
		},
		{
			name: "single inject type",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/pkg.Service": {
						TypeName:        "example.com/pkg.Service",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "Service",
						ConstructorName: "NewService",
						Dependencies:    []string{},
						IsField:         true,
					},
				},
				Edges: map[string][]string{
					"example.com/pkg.Service": {},
				},
			},
			sortedTypes: []string{"example.com/pkg.Service"},
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				if !strings.Contains(bootstrap.DependencyVar, "var dependency") {
					t.Error("missing var dependency declaration")
				}
				if !strings.Contains(bootstrap.DependencyVar, "service pkg.Service") {
					t.Error("missing service field")
				}
				if !strings.Contains(bootstrap.DependencyVar, "service := pkg.NewService()") {
					t.Error("missing NewService call")
				}
				if bootstrap.Hash == "" {
					t.Error("hash is empty")
				}
			},
		},
		{
			name: "inject with interface RegisteredType",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/pkg.Service": {
						TypeName:        "example.com/pkg.Service",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "Service",
						ConstructorName: "NewService",
						Dependencies:    []string{},
						IsField:         true,
						RegisteredType:  types.NewInterfaceType([]*types.Func{}, []types.Type{}), // Mock interface type
					},
				},
				Edges: map[string][]string{
					"example.com/pkg.Service": {},
				},
			},
			sortedTypes: []string{"example.com/pkg.Service"},
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Should use RegisteredType for variable declaration
				if !strings.Contains(bootstrap.DependencyVar, "service interface{}") {
					t.Error("missing interface-typed service field")
				}
				if !strings.Contains(bootstrap.DependencyVar, "service := pkg.NewService()") {
					t.Error("missing NewService call")
				}
			},
		},
		{
			name: "provide with interface RegisteredType",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/pkg.Repository": {
						TypeName:        "example.com/pkg.Repository",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "Repository",
						ConstructorName: "NewRepository",
						Dependencies:    []string{},
						IsField:         true, // Provide - now in return struct
						RegisteredType:  types.NewInterfaceType([]*types.Func{}, []types.Type{}), // Mock interface type
					},
					"example.com/pkg.Service": {
						TypeName:        "example.com/pkg.Service",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "Service",
						ConstructorName: "NewService",
						Dependencies:    []string{"example.com/pkg.Repository"},
						IsField:         true, // Inject - in return struct
					},
				},
				Edges: map[string][]string{
					"example.com/pkg.Repository": {},
					"example.com/pkg.Service":    {"example.com/pkg.Repository"},
				},
			},
			sortedTypes: []string{"example.com/pkg.Repository", "example.com/pkg.Service"},
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Repository (Provide) should appear in struct fields with interface type
				if !strings.Contains(bootstrap.DependencyVar, "repository interface{}") {
					t.Error("Provide type should be in struct fields with interface type")
				}
				// Repository should be initialized
				if !strings.Contains(bootstrap.DependencyVar, "repository := pkg.NewRepository()") {
					t.Error("missing NewRepository call for Provide")
				}
				// Service should depend on repository
				if !strings.Contains(bootstrap.DependencyVar, "service := pkg.NewService(repository)") {
					t.Error("Service should use repository as dependency")
				}
			},
		},
		{
			name: "inject with named dependency",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/pkg.Service": {
						TypeName:        "example.com/pkg.Service",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "Service",
						ConstructorName: "NewService",
						Dependencies:    []string{},
						IsField:         true,
						Name:            "primaryService", // Named dependency
					},
				},
				Edges: map[string][]string{
					"example.com/pkg.Service": {},
				},
			},
			sortedTypes: []string{"example.com/pkg.Service"},
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Should use custom name "primaryService" instead of default "service"
				if !strings.Contains(bootstrap.DependencyVar, "primaryService pkg.Service") {
					t.Error("missing named field primaryService")
				}
				if !strings.Contains(bootstrap.DependencyVar, "primaryService := pkg.NewService()") {
					t.Error("missing NewService call with named variable")
				}
				// Should NOT contain default name
				if strings.Contains(bootstrap.DependencyVar, "service pkg.Service") && !strings.Contains(bootstrap.DependencyVar, "primaryService pkg.Service") {
					t.Error("should use custom name, not default")
				}
			},
		},
		{
			name: "provide with named dependency used by inject",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/pkg.Repository": {
						TypeName:        "example.com/pkg.Repository",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "Repository",
						ConstructorName: "NewRepository",
						Dependencies:    []string{},
						IsField:         true, // Provide - now in return struct
						Name:            "userRepo", // Named dependency
					},
					"example.com/pkg.Service": {
						TypeName:        "example.com/pkg.Service",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "Service",
						ConstructorName: "NewService",
						Dependencies:    []string{"example.com/pkg.Repository"},
						IsField:         true, // Inject
					},
				},
				Edges: map[string][]string{
					"example.com/pkg.Repository": {},
					"example.com/pkg.Service":    {"example.com/pkg.Repository"},
				},
			},
			sortedTypes: []string{"example.com/pkg.Repository", "example.com/pkg.Service"},
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Repository should use named variable "userRepo" and be a struct field
				if !strings.Contains(bootstrap.DependencyVar, "userRepo pkg.Repository") {
					t.Error("missing userRepo struct field")
				}
				if !strings.Contains(bootstrap.DependencyVar, "userRepo := pkg.NewRepository()") {
					t.Error("missing named variable userRepo for Repository")
				}
				// Service should use userRepo as dependency
				if !strings.Contains(bootstrap.DependencyVar, "service := pkg.NewService(userRepo)") {
					t.Error("Service should use userRepo as dependency")
				}
			},
		},
		{
			name: "inject with provide dependency",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/pkg.Repository": {
						TypeName:        "example.com/pkg.Repository",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "Repository",
						ConstructorName: "NewRepository",
						Dependencies:    []string{},
						IsField:         true, // Provide - now in return struct
					},
					"example.com/pkg.Service": {
						TypeName:        "example.com/pkg.Service",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "Service",
						ConstructorName: "NewService",
						Dependencies:    []string{"example.com/pkg.Repository"},
						IsField:         true, // Inject
					},
				},
				Edges: map[string][]string{
					"example.com/pkg.Repository": {},
					"example.com/pkg.Service":    {"example.com/pkg.Repository"},
				},
			},
			sortedTypes: []string{"example.com/pkg.Repository", "example.com/pkg.Service"},
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Both should be struct fields (check longest name field which has no gofmt padding)
				if !strings.Contains(bootstrap.DependencyVar, "repository pkg.Repository") {
					t.Error("missing repository field")
				}
				// Both should have initialization
				if !strings.Contains(bootstrap.DependencyVar, "repository := pkg.NewRepository()") {
					t.Error("missing NewRepository call")
				}
				if !strings.Contains(bootstrap.DependencyVar, "service := pkg.NewService(repository)") {
					t.Error("missing NewService call with repository")
				}
			},
		},
		{
			name: "all provide types no inject",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/pkg.ConfigProvider": {
						TypeName:        "example.com/pkg.ConfigProvider",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "ConfigProvider",
						ConstructorName: "NewConfigProvider",
						Dependencies:    []string{},
						IsField:         true, // Provide - now in return struct
					},
					"example.com/pkg.DBProvider": {
						TypeName:        "example.com/pkg.DBProvider",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "DBProvider",
						ConstructorName: "NewDBProvider",
						Dependencies:    []string{},
						IsField:         true, // Provide - now in return struct
					},
				},
				Edges: map[string][]string{
					"example.com/pkg.ConfigProvider": {},
					"example.com/pkg.DBProvider":     {},
				},
			},
			sortedTypes: []string{"example.com/pkg.ConfigProvider", "example.com/pkg.DBProvider"},
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Should have var dependency with Provide fields
				if !strings.Contains(bootstrap.DependencyVar, "var dependency") {
					t.Error("missing var dependency declaration")
				}
				// Check longest name field which has no gofmt padding
				if !strings.Contains(bootstrap.DependencyVar, "configProvider pkg.ConfigProvider") {
					t.Error("missing configProvider field")
				}
				// Both should have initialization
				if !strings.Contains(bootstrap.DependencyVar, "configProvider := pkg.NewConfigProvider()") {
					t.Error("missing NewConfigProvider call")
				}
				if !strings.Contains(bootstrap.DependencyVar, "dbProvider := pkg.NewDBProvider()") {
					t.Error("missing NewDBProvider call")
				}
			},
		},
		{
			name: "complex dependency chain",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/pkg.A": {
						TypeName:        "example.com/pkg.A",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "A",
						ConstructorName: "NewA",
						Dependencies:    []string{},
						IsField:         true, // Provide - now in return struct
					},
					"example.com/pkg.B": {
						TypeName:        "example.com/pkg.B",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "B",
						ConstructorName: "NewB",
						Dependencies:    []string{"example.com/pkg.A"},
						IsField:         true, // Provide - now in return struct
					},
					"example.com/pkg.C": {
						TypeName:        "example.com/pkg.C",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "C",
						ConstructorName: "NewC",
						Dependencies:    []string{"example.com/pkg.B"},
						IsField:         true,
					},
				},
				Edges: map[string][]string{
					"example.com/pkg.A": {},
					"example.com/pkg.B": {"example.com/pkg.A"},
					"example.com/pkg.C": {"example.com/pkg.B"},
				},
			},
			sortedTypes: []string{"example.com/pkg.A", "example.com/pkg.B", "example.com/pkg.C"},
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Verify dependency chain initialization order
				if !strings.Contains(bootstrap.DependencyVar, "a := pkg.NewA()") {
					t.Error("missing NewA call")
				}
				if !strings.Contains(bootstrap.DependencyVar, "b := pkg.NewB(a)") {
					t.Error("missing NewB call with a")
				}
				if !strings.Contains(bootstrap.DependencyVar, "c := pkg.NewC(b)") {
					t.Error("missing NewC call with b")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bg := NewBootstrapGenerator()

			// Create minimal pass
			pass := &analysis.Pass{
				Pkg: types.NewPackage("main", "main"),
			}

			bootstrap, err := bg.GenerateBootstrap(pass, tt.graph, tt.sortedTypes)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateBootstrap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkOutput != nil {
				tt.checkOutput(t, bootstrap)
			}
		})
	}
}

func TestBootstrapGenerator_GenerateBootstrap_VariableExpressionAssignment(t *testing.T) {
	tests := []struct {
		name        string
		graph       *graph.Graph
		sortedTypes []string
		currentPkg  string
		currentName string
		wantErr     bool
		checkOutput func(t *testing.T, bootstrap *GeneratedBootstrap)
	}{
		{
			name: "basic variable node emits expression assignment",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"os.File": {
						TypeName:       "os.File",
						PackagePath:    "os",
						PackageName:    "os",
						LocalName:      "File",
						Dependencies:   []string{},
						IsField:        false,
						ExpressionText: "os.Stdout",
						IsQualified:    true,
					},
				},
				Edges: map[string][]string{
					"os.File": {},
				},
			},
			sortedTypes: []string{"os.File"},
			currentPkg:  "main",
			currentName: "main",
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Variable not depended upon by any other node -> blank assignment
				if !strings.Contains(bootstrap.DependencyVar, "_ = os.Stdout") {
					t.Errorf("missing blank expression assignment, got: %s", bootstrap.DependencyVar)
				}
				// Should NOT contain any constructor call for this node
				if strings.Contains(bootstrap.DependencyVar, "NewFile") {
					t.Error("should not contain constructor call for Variable node")
				}
			},
		},
		{
			name: "variable node does not trigger constructor error",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"os.File": {
						TypeName:        "os.File",
						PackagePath:     "os",
						PackageName:     "os",
						LocalName:       "File",
						ConstructorName: "", // No constructor
						Dependencies:    []string{},
						IsField:         false,
						ExpressionText:  "os.Stdout",
						IsQualified:     true,
					},
				},
				Edges: map[string][]string{
					"os.File": {},
				},
			},
			sortedTypes: []string{"os.File"},
			currentPkg:  "main",
			currentName: "main",
			wantErr:     false, // Must NOT error, even though ConstructorName is empty
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Variable not depended upon -> blank assignment
				if !strings.Contains(bootstrap.DependencyVar, "_ = os.Stdout") {
					t.Errorf("missing blank expression assignment, got: %s", bootstrap.DependencyVar)
				}
			},
		},
		{
			name: "variable with named dependency uses custom name",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"os.File": {
						TypeName:       "os.File",
						PackagePath:    "os",
						PackageName:    "os",
						LocalName:      "File",
						Dependencies:   []string{},
						IsField:        false,
						ExpressionText: "os.Stdout",
						IsQualified:    true,
						Name:           "stdoutWriter",
					},
				},
				Edges: map[string][]string{
					"os.File": {},
				},
			},
			sortedTypes: []string{"os.File"},
			currentPkg:  "main",
			currentName: "main",
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Variable not depended upon -> blank assignment (name is irrelevant)
				if !strings.Contains(bootstrap.DependencyVar, "_ = os.Stdout") {
					t.Errorf("missing blank expression assignment, got: %s", bootstrap.DependencyVar)
				}
			},
		},
		{
			name: "unqualified variable from another package gets qualification",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/config.Config": {
						TypeName:       "example.com/config.Config",
						PackagePath:    "example.com/config",
						PackageName:    "config",
						LocalName:      "Config",
						Dependencies:   []string{},
						IsField:        false,
						ExpressionText: "DefaultConfig",
						IsQualified:    false, // Local reference, not yet qualified
					},
				},
				Edges: map[string][]string{
					"example.com/config.Config": {},
				},
			},
			sortedTypes: []string{"example.com/config.Config"},
			currentPkg:  "main",
			currentName: "main",
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Variable not depended upon -> blank assignment with qualified expression
				if !strings.Contains(bootstrap.DependencyVar, "_ = config.DefaultConfig") {
					t.Errorf("missing blank qualified expression assignment, got: %s", bootstrap.DependencyVar)
				}
			},
		},
		{
			name: "unqualified variable from same package stays unqualified",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"main.Config": {
						TypeName:       "main.Config",
						PackagePath:    "main",
						PackageName:    "main",
						LocalName:      "Config",
						Dependencies:   []string{},
						IsField:        false,
						ExpressionText: "DefaultConfig",
						IsQualified:    false, // Local reference in same package
					},
				},
				Edges: map[string][]string{
					"main.Config": {},
				},
			},
			sortedTypes: []string{"main.Config"},
			currentPkg:  "main",
			currentName: "main",
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Variable not depended upon -> blank assignment, same package stays unqualified
				if !strings.Contains(bootstrap.DependencyVar, "_ = DefaultConfig") {
					t.Errorf("should not qualify same-package expression, got: %s", bootstrap.DependencyVar)
				}
			},
		},
		{
			name: "unqualified variable with package alias uses alias as qualifier",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/config.Config": {
						TypeName:       "example.com/config.Config",
						PackagePath:    "example.com/config",
						PackageName:    "config",
						PackageAlias:   "cfg",
						LocalName:      "Config",
						Dependencies:   []string{},
						IsField:        false,
						ExpressionText: "DefaultConfig",
						IsQualified:    false,
					},
				},
				Edges: map[string][]string{
					"example.com/config.Config": {},
				},
			},
			sortedTypes: []string{"example.com/config.Config"},
			currentPkg:  "main",
			currentName: "main",
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Variable not depended upon -> blank assignment with alias qualifier
				if !strings.Contains(bootstrap.DependencyVar, "_ = cfg.DefaultConfig") {
					t.Errorf("should use package alias as qualifier, got: %s", bootstrap.DependencyVar)
				}
			},
		},
		{
			name: "mixed variable and provider nodes in correct order",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"os.File": {
						TypeName:       "os.File",
						PackagePath:    "os",
						PackageName:    "os",
						LocalName:      "File",
						Dependencies:   []string{},
						IsField:        false,
						ExpressionText: "os.Stdout",
						IsQualified:    true,
					},
					"example.com/pkg.Service": {
						TypeName:        "example.com/pkg.Service",
						PackagePath:     "example.com/pkg",
						PackageName:     "pkg",
						LocalName:       "Service",
						ConstructorName: "NewService",
						Dependencies:    []string{"os.File"},
						IsField:         true,
					},
				},
				Edges: map[string][]string{
					"os.File":                {},
					"example.com/pkg.Service": {"os.File"},
				},
			},
			sortedTypes: []string{"os.File", "example.com/pkg.Service"},
			currentPkg:  "main",
			currentName: "main",
			wantErr:     false,
			checkOutput: func(t *testing.T, bootstrap *GeneratedBootstrap) {
				if bootstrap == nil {
					t.Fatal("bootstrap is nil")
				}
				// Variable should use expression assignment
				if !strings.Contains(bootstrap.DependencyVar, "file := os.Stdout") {
					t.Errorf("missing Variable expression assignment, got: %s", bootstrap.DependencyVar)
				}
				// Service should use constructor call
				if !strings.Contains(bootstrap.DependencyVar, "service := pkg.NewService(file)") {
					t.Errorf("missing Service constructor call, got: %s", bootstrap.DependencyVar)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bg := NewBootstrapGenerator()

			currentPkg := tt.currentPkg
			if currentPkg == "" {
				currentPkg = "main"
			}
			currentName := tt.currentName
			if currentName == "" {
				currentName = "main"
			}

			pass := &analysis.Pass{
				Pkg: types.NewPackage(currentPkg, currentName),
			}

			bootstrap, err := bg.GenerateBootstrap(pass, tt.graph, tt.sortedTypes)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateBootstrap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkOutput != nil {
				tt.checkOutput(t, bootstrap)
			}
		})
	}
}

func TestBootstrapGenerator_DetectExistingBootstrap(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   bool // true if bootstrap should be detected
	}{
		{
			name:   "no bootstrap",
			source: `package main`,
			want:   false,
		},
		{
			name: "bootstrap exists",
			source: `package main

var dependency = func() struct {
	service Service
} {
	service := NewService()
	return struct {
		service Service
	}{
		service: service,
	}
}()
`,
			want: true,
		},
		{
			name: "other variable",
			source: `package main

var config = &Config{}
`,
			want: false,
		},
		{
			name: "const declaration - not a var",
			source: `package main

const dependency = "value"
`,
			want: false,
		},
		{
			name: "type declaration",
			source: `package main

type dependency struct{}
`,
			want: false,
		},
		{
			name: "multiple var declarations",
			source: `package main

var (
	config Config
	dependency = NewDependency()
)
`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "", tt.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse source: %v", err)
			}

			pass := &analysis.Pass{
				Files: []*ast.File{file},
			}

			bg := NewBootstrapGenerator()
			existing := bg.DetectExistingBootstrap(pass)

			got := existing != nil
			if got != tt.want {
				t.Errorf("DetectExistingBootstrap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBootstrapGenerator_CheckBootstrapCurrent(t *testing.T) {
	// Create a graph
	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {
				TypeName:        "example.com/pkg.Service",
				PackagePath:     "example.com/pkg",
				LocalName:       "Service",
				ConstructorName: "NewService",
				IsField:         true,
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service": {},
		},
	}

	hash := ComputeGraphHash(g)

	source := `package main

// braider:hash:` + hash + `
var dependency = func() struct {
	service Service
} {
	service := NewService()
	return struct {
		service Service
	}{
		service: service,
	}
}()
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	pass := &analysis.Pass{
		Files: []*ast.File{file},
	}

	bg := NewBootstrapGenerator()
	existing := bg.DetectExistingBootstrap(pass)
	if existing == nil {
		t.Fatal("Failed to detect existing bootstrap")
	}

	isCurrent := bg.CheckBootstrapCurrent(pass, existing, g)
	if !isCurrent {
		t.Error("CheckBootstrapCurrent() should return true for matching hash")
	}

	// Test with different graph (different hash)
	g2 := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {
				TypeName:        "example.com/pkg.Service",
				PackagePath:     "example.com/pkg",
				LocalName:       "Service",
				ConstructorName: "NewService",
				IsField:         true,
			},
			"example.com/pkg.Repository": {
				TypeName:        "example.com/pkg.Repository",
				PackagePath:     "example.com/pkg",
				LocalName:       "Repository",
				ConstructorName: "NewRepository",
				IsField:         true,
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service":    {},
			"example.com/pkg.Repository": {},
		},
	}

	isCurrent2 := bg.CheckBootstrapCurrent(pass, existing, g2)
	if isCurrent2 {
		t.Error("CheckBootstrapCurrent() should return false for different graph")
	}

	// Test with nil Doc
	sourceNilDoc := `package main

var dependency = func() struct {
	service Service
} {
	service := NewService()
	return struct {
		service Service
	}{
		service: service,
	}
}()
`
	fileNilDoc, err := parser.ParseFile(token.NewFileSet(), "", sourceNilDoc, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	passNilDoc := &analysis.Pass{
		Files: []*ast.File{fileNilDoc},
	}

	existingNilDoc := bg.DetectExistingBootstrap(passNilDoc)
	if existingNilDoc == nil {
		t.Fatal("Failed to detect existing bootstrap without doc")
	}

	isCurrentNilDoc := bg.CheckBootstrapCurrent(passNilDoc, existingNilDoc, g)
	if isCurrentNilDoc {
		t.Error("CheckBootstrapCurrent() should return false when Doc is nil")
	}

	// Test with incorrect hash format
	sourceWrongHash := `package main

// braider-hash:wrongformat
var dependency = func() struct {
	service Service
} {
	service := NewService()
	return struct {
		service Service
	}{
		service: service,
	}
}()
`
	fileWrongHash, err := parser.ParseFile(token.NewFileSet(), "", sourceWrongHash, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	passWrongHash := &analysis.Pass{
		Files: []*ast.File{fileWrongHash},
	}

	existingWrongHash := bg.DetectExistingBootstrap(passWrongHash)
	if existingWrongHash == nil {
		t.Fatal("Failed to detect existing bootstrap")
	}

	isCurrentWrongHash := bg.CheckBootstrapCurrent(passWrongHash, existingWrongHash, g)
	if isCurrentWrongHash {
		t.Error("CheckBootstrapCurrent() should return false for wrong hash format")
	}
}

func TestExtractHashFromComments(t *testing.T) {
	tests := []struct {
		name     string
		comments []string
		want     string
	}{
		{
			name:     "nil comment group",
			comments: nil,
			want:     "",
		},
		{
			name:     "no hash in comments",
			comments: []string{"// This is a comment", "// Another comment"},
			want:     "",
		},
		{
			name:     "hash at beginning",
			comments: []string{"// braider:hash:abc123", "// Other comment"},
			want:     "abc123",
		},
		{
			name:     "hash in middle",
			comments: []string{"// First comment", "// braider:hash:def456", "// Last comment"},
			want:     "def456",
		},
		{
			name:     "hash at end",
			comments: []string{"// First comment", "// Second comment", "// braider:hash:xyz789"},
			want:     "xyz789",
		},
		{
			name:     "flexible whitespace - no spaces",
			comments: []string{"//braider:hash:compact"},
			want:     "compact",
		},
		{
			name:     "flexible whitespace - multiple spaces",
			comments: []string{"//   braider:hash:spaced"},
			want:     "spaced",
		},
		{
			name:     "wrong pattern - braider-hash",
			comments: []string{"// braider-hash:wrong"},
			want:     "",
		},
		{
			name:     "wrong pattern - braider:sha256",
			comments: []string{"// braider:sha256:wrong"},
			want:     "",
		},
		{
			name:     "valid hex hash with regex meta characters context",
			comments: []string{"// braider:hash:a1b2c3d4"},
			want:     "a1b2c3d4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc *ast.CommentGroup
			if tt.comments != nil {
				comments := make([]*ast.Comment, len(tt.comments))
				for i, text := range tt.comments {
					comments[i] = &ast.Comment{Text: text}
				}
				doc = &ast.CommentGroup{List: comments}
			}

			// Use unexported function via DetectExistingBootstrap path
			// We'll create a minimal test to exercise extractHashFromComments
			source := "package main\n\n"
			if doc != nil {
				for _, c := range doc.List {
					source += c.Text + "\n"
				}
			}
			source += `var dependency = func() struct{} { return struct{}{} }()`

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "", source, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse source: %v", err)
			}

			pass := &analysis.Pass{
				Files: []*ast.File{file},
			}

			bg := NewBootstrapGenerator()
			existing := bg.DetectExistingBootstrap(pass)

			if existing == nil && tt.want != "" {
				t.Fatal("Failed to detect existing bootstrap")
			}

			if existing != nil {
				// Check via CheckBootstrapCurrent which internally uses extractHashFromComments
				g := &graph.Graph{
					Nodes: map[string]*graph.Node{
						"test.Service": {
							TypeName:        "test.Service",
							ConstructorName: "NewService",
							IsField:         true,
						},
					},
				}

				// Create a graph with the expected hash
				expectedHash := ComputeGraphHash(g)

				isCurrent := bg.CheckBootstrapCurrent(pass, existing, g)

				switch tt.want {
				case "":
					// Should not be current if no hash found
					if isCurrent {
						t.Error("CheckBootstrapCurrent() should return false when no hash comment")
					}
				case expectedHash:
					// If extracted hash matches computed hash
					if !isCurrent {
						t.Error("CheckBootstrapCurrent() should return true for matching hash")
					}
				}
			}
		})
	}
}

func TestRewriteExpressionAliases(t *testing.T) {
	tests := []struct {
		name           string
		expressionText string
		exprPkgs       []string
		exprPkgNames   []string
		aliasMap       map[string]string
		want           string
	}{
		{
			name:           "no alias needed",
			expressionText: "os.Stdout",
			exprPkgs:       []string{"os"},
			exprPkgNames:   []string{"os"},
			aliasMap:       map[string]string{},
			want:           "os.Stdout",
		},
		{
			name:           "alias rewrite",
			expressionText: "os.Stdout",
			exprPkgs:       []string{"os"},
			exprPkgNames:   []string{"os"},
			aliasMap:       map[string]string{"os": "os2"},
			want:           "os2.Stdout",
		},
		{
			name:           "empty alias skipped",
			expressionText: "os.Stdout",
			exprPkgs:       []string{"os"},
			exprPkgNames:   []string{"os"},
			aliasMap:       map[string]string{"os": ""},
			want:           "os.Stdout",
		},
		{
			name:           "no matching pkg in aliasMap",
			expressionText: "config.DefaultConfig",
			exprPkgs:       []string{"example.com/config"},
			exprPkgNames:   []string{"config"},
			aliasMap:       map[string]string{"os": "os2"},
			want:           "config.DefaultConfig",
		},
		{
			name:           "exprPkgNames shorter than exprPkgs",
			expressionText: "os.Stdout",
			exprPkgs:       []string{"os", "extra"},
			exprPkgNames:   []string{"os"},
			aliasMap:       map[string]string{"os": "os2", "extra": "extra2"},
			want:           "os2.Stdout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteExpressionAliases(tt.expressionText, tt.exprPkgs, tt.exprPkgNames, tt.aliasMap)
			if got != tt.want {
				t.Errorf("rewriteExpressionAliases() = %q, want %q", got, tt.want)
			}
		})
	}
}
