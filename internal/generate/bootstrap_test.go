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
						IsField:         false, // Provide
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
				// Only Service should be a field (Inject)
				if !strings.Contains(bootstrap.DependencyVar, "service pkg.Service") {
					t.Error("missing service field")
				}
				// Repository should NOT be a field (Provide)
				if strings.Contains(bootstrap.DependencyVar, "repository pkg.Repository") {
					t.Error("repository should not be a field")
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
				IsField:         false,
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
}
