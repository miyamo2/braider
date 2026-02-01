package generate

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"slices"
	"sort"

	"github.com/miyamo2/braider/internal/graph"
)

// hashLength is the number of hex characters to use for the hash (64-bit).
const hashLength = 16

// ComputeGraphHash computes a deterministic hash from the dependency graph.
// It generates a 16-character hex string (64-bit hash) based on sorted type names,
// constructor names, IsField flags, and their dependencies.
// Using 64-bit reduces collision probability compared to 32-bit hashes.
//
// The hash is computed from node fields (ConstructorName, Dependencies) which must
// be properly initialized. Empty strings and nil slices are handled gracefully.
func ComputeGraphHash(g *graph.Graph) string {
	if g == nil {
		return "0000000000000000"
	}

	// Collect all type names and sort them for determinism
	types := slices.Sorted(maps.Keys(g.Nodes))

	// Build a deterministic string representation
	h := sha256.New()
	for _, typeName := range types {
		node := g.Nodes[typeName]

		// Type name
		h.Write([]byte(typeName))
		h.Write([]byte{0}) // separator

		// Constructor name (affects generated code)
		h.Write([]byte(node.ConstructorName))
		h.Write([]byte{0})

		// IsField flag (affects field vs local variable placement)
		if node.IsField {
			h.Write([]byte{1})
		} else {
			h.Write([]byte{0})
		}
		h.Write([]byte{0})

		// Add dependencies in sorted order
		sortedDeps := make([]string, len(node.Dependencies))
		copy(sortedDeps, node.Dependencies)
		sort.Strings(sortedDeps)

		for _, dep := range sortedDeps {
			h.Write([]byte(dep))
			h.Write([]byte{0}) // separator
		}
	}

	// Return first hashLength hex characters (64-bit)
	fullHash := hex.EncodeToString(h.Sum(nil))
	if len(fullHash) >= hashLength {
		return fullHash[:hashLength]
	}
	return fullHash
}
