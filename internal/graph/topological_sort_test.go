package graph

import (
	"strings"
	"testing"
)

// TestTopologicalSort_Sort tests the topological sorting algorithm.
func TestTopologicalSort_Sort(t *testing.T) {
	tests := []struct {
		name      string
		graph     *Graph
		wantOrder []string
		wantErr   bool
		errMsg    string // Expected error message substring
	}{
		{
			name: "empty graph",
			graph: &Graph{
				Nodes: map[string]*Node{},
				Edges: map[string][]string{},
			},
			wantOrder: []string{},
			wantErr:   false,
		},
		{
			name: "single node with no dependencies",
			graph: &Graph{
				Nodes: map[string]*Node{
					"example.com/repo.UserRepository": {
						TypeName:        "example.com/repo.UserRepository",
						PackagePath:     "example.com/repo",
						LocalName:       "UserRepository",
						ConstructorName: "NewUserRepository",
						Dependencies:    []string{},
						InDegree:        0,
					},
				},
				Edges: map[string][]string{
					"example.com/repo.UserRepository": {},
				},
			},
			wantOrder: []string{"example.com/repo.UserRepository"},
			wantErr:   false,
		},
		{
			name: "linear dependency chain",
			graph: &Graph{
				Nodes: map[string]*Node{
					"example.com/repo.UserRepository": {
						TypeName:        "example.com/repo.UserRepository",
						PackagePath:     "example.com/repo",
						LocalName:       "UserRepository",
						ConstructorName: "NewUserRepository",
						Dependencies:    []string{},
						InDegree:        0,
					},
					"example.com/service.UserService": {
						TypeName:        "example.com/service.UserService",
						PackagePath:     "example.com/service",
						LocalName:       "UserService",
						ConstructorName: "NewUserService",
						Dependencies:    []string{"example.com/repo.UserRepository"},
						InDegree:        1,
					},
					"example.com/handler.UserHandler": {
						TypeName:        "example.com/handler.UserHandler",
						PackagePath:     "example.com/handler",
						LocalName:       "UserHandler",
						ConstructorName: "NewUserHandler",
						Dependencies:    []string{"example.com/service.UserService"},
						InDegree:        1,
					},
				},
				Edges: map[string][]string{
					"example.com/repo.UserRepository": {},
					"example.com/service.UserService": {"example.com/repo.UserRepository"},
					"example.com/handler.UserHandler": {"example.com/service.UserService"},
				},
			},
			wantOrder: []string{
				"example.com/repo.UserRepository",
				"example.com/service.UserService",
				"example.com/handler.UserHandler",
			},
			wantErr: false,
		},
		{
			name: "multiple dependencies with deterministic ordering",
			graph: &Graph{
				Nodes: map[string]*Node{
					"example.com/repo.UserRepository": {
						TypeName:        "example.com/repo.UserRepository",
						PackagePath:     "example.com/repo",
						LocalName:       "UserRepository",
						ConstructorName: "NewUserRepository",
						Dependencies:    []string{},
						InDegree:        0,
					},
					"example.com/repo.OrderRepository": {
						TypeName:        "example.com/repo.OrderRepository",
						PackagePath:     "example.com/repo",
						LocalName:       "OrderRepository",
						ConstructorName: "NewOrderRepository",
						Dependencies:    []string{},
						InDegree:        0,
					},
					"example.com/service.OrderService": {
						TypeName:        "example.com/service.OrderService",
						PackagePath:     "example.com/service",
						LocalName:       "OrderService",
						ConstructorName: "NewOrderService",
						Dependencies: []string{
							"example.com/repo.UserRepository",
							"example.com/repo.OrderRepository",
						},
						InDegree: 2,
					},
				},
				Edges: map[string][]string{
					"example.com/repo.UserRepository":  {},
					"example.com/repo.OrderRepository": {},
					"example.com/service.OrderService": {
						"example.com/repo.UserRepository",
						"example.com/repo.OrderRepository",
					},
				},
			},
			wantOrder: []string{
				"example.com/repo.OrderRepository", // Alphabetically first
				"example.com/repo.UserRepository",
				"example.com/service.OrderService",
			},
			wantErr: false,
		},
		{
			name: "diamond dependency pattern",
			graph: &Graph{
				Nodes: map[string]*Node{
					"example.com/A": {
						TypeName:        "example.com/A",
						PackagePath:     "example.com",
						LocalName:       "A",
						ConstructorName: "NewA",
						Dependencies:    []string{},
						InDegree:        0,
					},
					"example.com/B": {
						TypeName:        "example.com/B",
						PackagePath:     "example.com",
						LocalName:       "B",
						ConstructorName: "NewB",
						Dependencies:    []string{"example.com/A"},
						InDegree:        1,
					},
					"example.com/C": {
						TypeName:        "example.com/C",
						PackagePath:     "example.com",
						LocalName:       "C",
						ConstructorName: "NewC",
						Dependencies:    []string{"example.com/A"},
						InDegree:        1,
					},
					"example.com/D": {
						TypeName:        "example.com/D",
						PackagePath:     "example.com",
						LocalName:       "D",
						ConstructorName: "NewD",
						Dependencies:    []string{"example.com/B", "example.com/C"},
						InDegree:        2,
					},
				},
				Edges: map[string][]string{
					"example.com/A": {},
					"example.com/B": {"example.com/A"},
					"example.com/C": {"example.com/A"},
					"example.com/D": {"example.com/B", "example.com/C"},
				},
			},
			wantOrder: []string{
				"example.com/A",
				"example.com/B", // B before C alphabetically
				"example.com/C",
				"example.com/D",
			},
			wantErr: false,
		},
		{
			name: "simple circular dependency",
			graph: &Graph{
				Nodes: map[string]*Node{
					"example.com/service.ServiceA": {
						TypeName:        "example.com/service.ServiceA",
						PackagePath:     "example.com/service",
						LocalName:       "ServiceA",
						ConstructorName: "NewServiceA",
						Dependencies:    []string{"example.com/service.ServiceB"},
						InDegree:        1,
					},
					"example.com/service.ServiceB": {
						TypeName:        "example.com/service.ServiceB",
						PackagePath:     "example.com/service",
						LocalName:       "ServiceB",
						ConstructorName: "NewServiceB",
						Dependencies:    []string{"example.com/service.ServiceA"},
						InDegree:        1,
					},
				},
				Edges: map[string][]string{
					"example.com/service.ServiceA": {"example.com/service.ServiceB"},
					"example.com/service.ServiceB": {"example.com/service.ServiceA"},
				},
			},
			wantOrder: nil,
			wantErr:   true,
			errMsg:    "circular dependency detected",
		},
		{
			name: "three-node circular dependency",
			graph: &Graph{
				Nodes: map[string]*Node{
					"example.com/A": {
						TypeName:        "example.com/A",
						PackagePath:     "example.com",
						LocalName:       "A",
						ConstructorName: "NewA",
						Dependencies:    []string{"example.com/B"},
						InDegree:        1,
					},
					"example.com/B": {
						TypeName:        "example.com/B",
						PackagePath:     "example.com",
						LocalName:       "B",
						ConstructorName: "NewB",
						Dependencies:    []string{"example.com/C"},
						InDegree:        1,
					},
					"example.com/C": {
						TypeName:        "example.com/C",
						PackagePath:     "example.com",
						LocalName:       "C",
						ConstructorName: "NewC",
						Dependencies:    []string{"example.com/A"},
						InDegree:        1,
					},
				},
				Edges: map[string][]string{
					"example.com/A": {"example.com/B"},
					"example.com/B": {"example.com/C"},
					"example.com/C": {"example.com/A"},
				},
			},
			wantOrder: nil,
			wantErr:   true,
			errMsg:    "circular dependency detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorter := NewTopologicalSorter()
			gotOrder, err := sorter.Sort(tt.graph)

			if (err != nil) != tt.wantErr {
				t.Errorf("Sort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("Sort() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Sort() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if len(gotOrder) != len(tt.wantOrder) {
				t.Errorf("Sort() returned %d items, want %d", len(gotOrder), len(tt.wantOrder))
				return
			}

			for i, want := range tt.wantOrder {
				if gotOrder[i] != want {
					t.Errorf("Sort()[%d] = %s, want %s", i, gotOrder[i], want)
				}
			}
		})
	}
}

// TestTopologicalSort_CycleError tests the CycleError structure and message.
func TestTopologicalSort_CycleError(t *testing.T) {
	tests := []struct {
		name      string
		graph     *Graph
		wantCycle []string // Expected cycle path elements (order may vary)
	}{
		{
			name: "two-node cycle",
			graph: &Graph{
				Nodes: map[string]*Node{
					"example.com/A": {
						TypeName:     "example.com/A",
						Dependencies: []string{"example.com/B"},
						InDegree:     1,
					},
					"example.com/B": {
						TypeName:     "example.com/B",
						Dependencies: []string{"example.com/A"},
						InDegree:     1,
					},
				},
				Edges: map[string][]string{
					"example.com/A": {"example.com/B"},
					"example.com/B": {"example.com/A"},
				},
			},
			wantCycle: []string{"example.com/A", "example.com/B"},
		},
		{
			name: "three-node cycle",
			graph: &Graph{
				Nodes: map[string]*Node{
					"example.com/A": {
						TypeName:     "example.com/A",
						Dependencies: []string{"example.com/B"},
						InDegree:     1,
					},
					"example.com/B": {
						TypeName:     "example.com/B",
						Dependencies: []string{"example.com/C"},
						InDegree:     1,
					},
					"example.com/C": {
						TypeName:     "example.com/C",
						Dependencies: []string{"example.com/A"},
						InDegree:     1,
					},
				},
				Edges: map[string][]string{
					"example.com/A": {"example.com/B"},
					"example.com/B": {"example.com/C"},
					"example.com/C": {"example.com/A"},
				},
			},
			wantCycle: []string{"example.com/A", "example.com/B", "example.com/C"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorter := NewTopologicalSorter()
			_, err := sorter.Sort(tt.graph)

			if err == nil {
				t.Fatalf("Sort() expected CycleError but got nil")
			}

			cycleErr, ok := err.(*CycleError)
			if !ok {
				t.Fatalf("Sort() error is not *CycleError, got %T", err)
			}

			// Check that cycle path contains all expected nodes
			cycleSet := make(map[string]bool)
			for _, node := range cycleErr.Cycle {
				cycleSet[node] = true
			}

			for _, expectedNode := range tt.wantCycle {
				if !cycleSet[expectedNode] {
					t.Errorf("CycleError.Cycle missing expected node %s", expectedNode)
				}
			}

			// Check error message format
			errMsg := cycleErr.Error()
			if !strings.Contains(errMsg, "circular dependency detected") {
				t.Errorf("CycleError.Error() = %q, want to contain 'circular dependency detected'", errMsg)
			}
			if !strings.Contains(errMsg, " -> ") {
				t.Errorf("CycleError.Error() = %q, want to contain ' -> ' separator", errMsg)
			}
		})
	}
}

// TestTopologicalSort_AlphabeticalTieBreaking tests deterministic ordering.
func TestTopologicalSort_AlphabeticalTieBreaking(t *testing.T) {
	// Create a graph with multiple valid orderings
	// All three nodes have no dependencies, so any order is valid
	// But we expect alphabetical order for determinism
	graph := &Graph{
		Nodes: map[string]*Node{
			"example.com/Z": {
				TypeName:     "example.com/Z",
				Dependencies: []string{},
				InDegree:     0,
			},
			"example.com/A": {
				TypeName:     "example.com/A",
				Dependencies: []string{},
				InDegree:     0,
			},
			"example.com/M": {
				TypeName:     "example.com/M",
				Dependencies: []string{},
				InDegree:     0,
			},
		},
		Edges: map[string][]string{
			"example.com/Z": {},
			"example.com/A": {},
			"example.com/M": {},
		},
	}

	sorter := NewTopologicalSorter()
	gotOrder, err := sorter.Sort(graph)

	if err != nil {
		t.Fatalf("Sort() unexpected error = %v", err)
	}

	expectedOrder := []string{
		"example.com/A",
		"example.com/M",
		"example.com/Z",
	}

	if len(gotOrder) != len(expectedOrder) {
		t.Fatalf("Sort() returned %d items, want %d", len(gotOrder), len(expectedOrder))
	}

	for i, want := range expectedOrder {
		if gotOrder[i] != want {
			t.Errorf("Sort()[%d] = %s, want %s (alphabetical order)", i, gotOrder[i], want)
		}
	}
}
