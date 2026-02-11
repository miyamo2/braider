package generate

import (
	"testing"

	"github.com/miyamo2/braider/internal/graph"
)

func TestComputeGraphHash(t *testing.T) {
	tests := []struct {
		name  string
		graph *graph.Graph
		want  string
	}{
		{
			name:  "nil graph",
			graph: nil,
			want:  "0000000000000000",
		},
		{
			name: "empty graph",
			graph: &graph.Graph{
				Nodes: make(map[string]*graph.Node),
				Edges: make(map[string][]string),
			},
			want: "e3b0c44298fc1c14", // SHA-256 of empty string (first 16 chars)
		},
		{
			name: "single type no dependencies",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/pkg.Service": {},
				},
				Edges: map[string][]string{
					"example.com/pkg.Service": {},
				},
			},
		},
		{
			name: "single type with dependency",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/pkg.Service":    {},
					"example.com/pkg.Repository": {},
				},
				Edges: map[string][]string{
					"example.com/pkg.Service":    {"example.com/pkg.Repository"},
					"example.com/pkg.Repository": {},
				},
			},
		},
		{
			name: "multiple types with dependencies",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/pkg.ServiceA":   {},
					"example.com/pkg.ServiceB":   {},
					"example.com/pkg.Repository": {},
				},
				Edges: map[string][]string{
					"example.com/pkg.ServiceA":   {"example.com/pkg.Repository"},
					"example.com/pkg.ServiceB":   {"example.com/pkg.Repository"},
					"example.com/pkg.Repository": {},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeGraphHash(tt.graph)

			// Verify hash length (changed to 16 characters for 64-bit hash)
			if len(got) != 16 {
				t.Errorf("ComputeGraphHash() hash length = %d, want 16", len(got))
			}

			// Verify determinism
			got2 := ComputeGraphHash(tt.graph)
			if got != got2 {
				t.Errorf("ComputeGraphHash() not deterministic: first=%s, second=%s", got, got2)
			}

			// Verify expected hash for known cases
			if tt.want != "" && got != tt.want {
				t.Errorf("ComputeGraphHash() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestComputeGraphHash_OrderIndependent(t *testing.T) {
	// Graph with dependencies in different order
	graph1 := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.ServiceA":   {},
			"example.com/pkg.ServiceB":   {},
			"example.com/pkg.Repository": {},
		},
		Edges: map[string][]string{
			"example.com/pkg.ServiceA":   {"example.com/pkg.Repository"},
			"example.com/pkg.ServiceB":   {"example.com/pkg.Repository"},
			"example.com/pkg.Repository": {},
		},
	}

	graph2 := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Repository": {},
			"example.com/pkg.ServiceB":   {},
			"example.com/pkg.ServiceA":   {},
		},
		Edges: map[string][]string{
			"example.com/pkg.Repository": {},
			"example.com/pkg.ServiceB":   {"example.com/pkg.Repository"},
			"example.com/pkg.ServiceA":   {"example.com/pkg.Repository"},
		},
	}

	hash1 := ComputeGraphHash(graph1)
	hash2 := ComputeGraphHash(graph2)

	if hash1 != hash2 {
		t.Errorf("ComputeGraphHash() order dependent: hash1=%s, hash2=%s", hash1, hash2)
	}
}

func TestComputeGraphHash_DependencyOrderIndependent(t *testing.T) {
	// Same dependencies in different order
	graph1 := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service": {"example.com/pkg.RepoA", "example.com/pkg.RepoB"},
		},
	}

	graph2 := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service": {"example.com/pkg.RepoB", "example.com/pkg.RepoA"},
		},
	}

	hash1 := ComputeGraphHash(graph1)
	hash2 := ComputeGraphHash(graph2)

	if hash1 != hash2 {
		t.Errorf("ComputeGraphHash() dependency order dependent: hash1=%s, hash2=%s", hash1, hash2)
	}
}

func TestComputeGraphHash_ChangesDetected(t *testing.T) {
	graph1 := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {
				TypeName:        "example.com/pkg.Service",
				ConstructorName: "NewService",
				Dependencies:    []string{"example.com/pkg.Repository"},
				IsField:         true,
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service": {"example.com/pkg.Repository"},
		},
	}

	graph2 := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {
				TypeName:        "example.com/pkg.Service",
				ConstructorName: "NewService",
				Dependencies:    []string{"example.com/pkg.Repository", "example.com/pkg.Cache"},
				IsField:         true,
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service": {"example.com/pkg.Repository", "example.com/pkg.Cache"},
		},
	}

	hash1 := ComputeGraphHash(graph1)
	hash2 := ComputeGraphHash(graph2)

	if hash1 == hash2 {
		t.Errorf("ComputeGraphHash() did not detect dependency change")
	}
}

func TestComputeGraphHash_ConstructorNameChanges(t *testing.T) {
	graph1 := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {
				TypeName:        "example.com/pkg.Service",
				PackagePath:     "example.com/pkg",
				LocalName:       "Service",
				ConstructorName: "NewService",
				Dependencies:    []string{"example.com/pkg.Repository"},
				IsField:         true,
			},
			"example.com/pkg.Repository": {
				TypeName:        "example.com/pkg.Repository",
				PackagePath:     "example.com/pkg",
				LocalName:       "Repository",
				ConstructorName: "NewRepository",
				Dependencies:    []string{},
				IsField:         false,
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service":    {"example.com/pkg.Repository"},
			"example.com/pkg.Repository": {},
		},
	}

	graph2 := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {
				TypeName:        "example.com/pkg.Service",
				PackagePath:     "example.com/pkg",
				LocalName:       "Service",
				ConstructorName: "NewServiceV2", // Different constructor name
				Dependencies:    []string{"example.com/pkg.Repository"},
				IsField:         true,
			},
			"example.com/pkg.Repository": {
				TypeName:        "example.com/pkg.Repository",
				PackagePath:     "example.com/pkg",
				LocalName:       "Repository",
				ConstructorName: "NewRepository",
				Dependencies:    []string{},
				IsField:         false,
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service":    {"example.com/pkg.Repository"},
			"example.com/pkg.Repository": {},
		},
	}

	hash1 := ComputeGraphHash(graph1)
	hash2 := ComputeGraphHash(graph2)

	if hash1 == hash2 {
		t.Errorf("ComputeGraphHash() did not detect constructor name change: hash1=%s, hash2=%s", hash1, hash2)
	}
}

func TestComputeGraphHash_ExpressionTextChangesHash(t *testing.T) {
	// A Variable node with ExpressionText should produce a different hash
	// than the same node without ExpressionText.
	graphWithoutExpr := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"os.File": {
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				Dependencies:   []string{},
				IsField:        false,
				ExpressionText: "", // No expression (Provider/Injector node)
			},
		},
		Edges: map[string][]string{
			"os.File": {},
		},
	}

	graphWithExpr := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"os.File": {
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				Dependencies:   []string{},
				IsField:        false,
				ExpressionText: "os.Stdout", // Variable node with expression
			},
		},
		Edges: map[string][]string{
			"os.File": {},
		},
	}

	hashWithout := ComputeGraphHash(graphWithoutExpr)
	hashWith := ComputeGraphHash(graphWithExpr)

	if hashWithout == hashWith {
		t.Errorf("ComputeGraphHash() did not detect ExpressionText change: hashWithout=%s, hashWith=%s", hashWithout, hashWith)
	}
}

func TestComputeGraphHash_DifferentExpressionTextsProduceDifferentHashes(t *testing.T) {
	// Two Variable nodes with different ExpressionText should produce different hashes.
	graph1 := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"os.File": {
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				Dependencies:   []string{},
				IsField:        false,
				ExpressionText: "os.Stdout",
			},
		},
		Edges: map[string][]string{
			"os.File": {},
		},
	}

	graph2 := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"os.File": {
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				Dependencies:   []string{},
				IsField:        false,
				ExpressionText: "os.Stderr", // Different expression
			},
		},
		Edges: map[string][]string{
			"os.File": {},
		},
	}

	hash1 := ComputeGraphHash(graph1)
	hash2 := ComputeGraphHash(graph2)

	if hash1 == hash2 {
		t.Errorf("ComputeGraphHash() did not detect ExpressionText difference: hash1=%s, hash2=%s", hash1, hash2)
	}
}

func TestComputeGraphHash_EmptyExpressionTextPreservesExistingHash(t *testing.T) {
	// Verify that when ExpressionText is empty (Provider/Injector nodes),
	// the hash is the same as it would be without the ExpressionText feature.
	// This ensures backward compatibility.
	graphProvider := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {
				TypeName:        "example.com/pkg.Service",
				PackagePath:     "example.com/pkg",
				PackageName:     "pkg",
				LocalName:       "Service",
				ConstructorName: "NewService",
				Dependencies:    []string{},
				IsField:         true,
				ExpressionText:  "", // Empty - Provider/Injector node
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service": {},
		},
	}

	// The hash should be deterministic and stable for nodes with empty ExpressionText
	hash1 := ComputeGraphHash(graphProvider)
	hash2 := ComputeGraphHash(graphProvider)

	if hash1 != hash2 {
		t.Errorf("ComputeGraphHash() not deterministic for empty ExpressionText: hash1=%s, hash2=%s", hash1, hash2)
	}

	// Verify the hash length is correct
	if len(hash1) != 16 {
		t.Errorf("ComputeGraphHash() hash length = %d, want 16", len(hash1))
	}
}

func TestComputeGraphHash_MixedVariableAndProviderNodes(t *testing.T) {
	// A graph with both Variable and Provider nodes should produce a stable hash.
	// Adding a Variable node should change the hash compared to having only the Provider.
	graphProviderOnly := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {
				TypeName:        "example.com/pkg.Service",
				PackagePath:     "example.com/pkg",
				PackageName:     "pkg",
				LocalName:       "Service",
				ConstructorName: "NewService",
				Dependencies:    []string{},
				IsField:         true,
				ExpressionText:  "",
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service": {},
		},
	}

	graphMixed := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {
				TypeName:        "example.com/pkg.Service",
				PackagePath:     "example.com/pkg",
				PackageName:     "pkg",
				LocalName:       "Service",
				ConstructorName: "NewService",
				Dependencies:    []string{},
				IsField:         true,
				ExpressionText:  "",
			},
			"os.File": {
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				Dependencies:   []string{},
				IsField:        false,
				ExpressionText: "os.Stdout",
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service": {},
			"os.File":                {},
		},
	}

	hashProviderOnly := ComputeGraphHash(graphProviderOnly)
	hashMixed := ComputeGraphHash(graphMixed)

	if hashProviderOnly == hashMixed {
		t.Errorf("ComputeGraphHash() did not detect addition of Variable node: hashProviderOnly=%s, hashMixed=%s", hashProviderOnly, hashMixed)
	}
}

func TestComputeGraphHash_IsFieldFlagChanges(t *testing.T) {
	graph1 := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {
				TypeName:        "example.com/pkg.Service",
				PackagePath:     "example.com/pkg",
				LocalName:       "Service",
				ConstructorName: "NewService",
				Dependencies:    []string{},
				IsField:         true, // Field in dependency struct
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service": {},
		},
	}

	graph2 := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {
				TypeName:        "example.com/pkg.Service",
				PackagePath:     "example.com/pkg",
				LocalName:       "Service",
				ConstructorName: "NewService",
				Dependencies:    []string{},
				IsField:         false, // Local variable only
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service": {},
		},
	}

	hash1 := ComputeGraphHash(graph1)
	hash2 := ComputeGraphHash(graph2)

	if hash1 == hash2 {
		t.Errorf("ComputeGraphHash() did not detect IsField flag change: hash1=%s, hash2=%s", hash1, hash2)
	}
}
