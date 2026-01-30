package graph

import (
	"fmt"
	"sort"
	"strings"
)

// TopologicalSorter provides topological ordering with cycle detection.
type TopologicalSorter struct{}

// NewTopologicalSorter creates a new topological sorter.
func NewTopologicalSorter() *TopologicalSorter {
	return &TopologicalSorter{}
}

// Sort orders nodes topologically using Kahn's algorithm.
// Returns ordered node list or error with cycle path if cycle detected.
func (s *TopologicalSorter) Sort(graph *Graph) ([]string, error) {
	// Handle empty graph
	if len(graph.Nodes) == 0 {
		return []string{}, nil
	}

	// Create a working copy of in-degrees to avoid modifying the original graph
	inDegrees := make(map[string]int)
	for typeName, node := range graph.Nodes {
		inDegrees[typeName] = node.InDegree
	}

	// Build reverse edges: dependents[node] = nodes that depend on this node
	// graph.Edges[A] = [B] means "A depends on B"
	// We need reverse: when B is processed, decrement A's in-degree
	dependents := make(map[string][]string)
	for from, tos := range graph.Edges {
		for _, to := range tos {
			dependents[to] = append(dependents[to], from)
		}
	}

	// Initialize queue with all zero in-degree nodes (alphabetically sorted)
	queue := s.findZeroInDegreeNodes(inDegrees)

	// Result list
	result := []string{}

	// Process nodes using Kahn's algorithm
	for len(queue) > 0 {
		// Dequeue (always sorted alphabetically for determinism)
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// For each node that depends on current
		if deps, ok := dependents[current]; ok {
			for _, dependent := range deps {
				// Decrement in-degree of dependent
				inDegrees[dependent]--

				// If in-degree becomes zero, add to queue (maintaining sorted order)
				if inDegrees[dependent] == 0 {
					queue = s.insertSorted(queue, dependent)
				}
			}
		}
	}

	// Check if all nodes were processed (cycle detection)
	if len(result) != len(graph.Nodes) {
		// Reconstruct cycle path
		cycle := s.findCycle(graph, inDegrees)
		return nil, &CycleError{Cycle: cycle}
	}

	return result, nil
}

// findZeroInDegreeNodes returns all nodes with zero in-degree, sorted alphabetically.
func (s *TopologicalSorter) findZeroInDegreeNodes(inDegrees map[string]int) []string {
	zeroNodes := []string{}
	for typeName, degree := range inDegrees {
		if degree == 0 {
			zeroNodes = append(zeroNodes, typeName)
		}
	}
	sort.Strings(zeroNodes)
	return zeroNodes
}

// insertSorted inserts a node into a sorted slice maintaining alphabetical order.
func (s *TopologicalSorter) insertSorted(slice []string, node string) []string {
	// Find insertion position
	pos := sort.SearchStrings(slice, node)
	// Insert at position
	slice = append(slice, "")
	copy(slice[pos+1:], slice[pos:])
	slice[pos] = node
	return slice
}

// findCycle reconstructs a cycle path from remaining nodes with non-zero in-degree.
func (s *TopologicalSorter) findCycle(graph *Graph, inDegrees map[string]int) []string {
	// Find a node that's part of the cycle (non-zero in-degree)
	var startNode string
	for typeName, degree := range inDegrees {
		if degree > 0 {
			startNode = typeName
			break
		}
	}

	if startNode == "" {
		// This shouldn't happen if caller detected cycle correctly
		return []string{}
	}

	// Use DFS to find the cycle path starting from this node
	visited := make(map[string]bool)
	path := []string{}
	cycle := s.dfsFindCycle(graph, startNode, visited, path, inDegrees)

	return cycle
}

// dfsFindCycle performs DFS to find a cycle path.
// Only traverses nodes with non-zero in-degree (part of the cycle).
func (s *TopologicalSorter) dfsFindCycle(
	graph *Graph,
	current string,
	visited map[string]bool,
	path []string,
	inDegrees map[string]int,
) []string {
	// Add current to path
	path = append(path, current)
	visited[current] = true

	// Build path index for O(1) lookup
	pathIndex := make(map[string]int, len(path))
	for i, node := range path {
		pathIndex[node] = i
	}

	// Check current node's dependencies
	if edges, ok := graph.Edges[current]; ok {
		for _, dep := range edges {
			// Only follow edges to nodes in the cycle (non-zero in-degree)
			if inDegrees[dep] == 0 {
				continue
			}

			// Check if we've found a cycle (dep is in current path)
			if cycleStart, inPath := pathIndex[dep]; inPath {
				// Extract the cycle from path and add dep to complete the cycle
				cycle := append(path[cycleStart:], dep)
				return cycle
			}

			// If not visited, continue DFS
			if !visited[dep] {
				result := s.dfsFindCycle(graph, dep, visited, path, inDegrees)
				if len(result) > 0 {
					return result
				}
			}
		}
	}

	return []string{}
}

// CycleError represents a circular dependency error.
type CycleError struct {
	Cycle []string // Cycle path, e.g., ["A", "B", "C", "A"]
}

// Error returns the error message.
func (e *CycleError) Error() string {
	return fmt.Sprintf("circular dependency detected: %s", strings.Join(e.Cycle, " -> "))
}
