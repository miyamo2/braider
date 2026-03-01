package generate

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"slices"
	"sort"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/graph"
)

// hashLength is the number of hex characters to use for the hash (64-bit).
const hashLength = 16

// ComputeGraphHash computes a deterministic hash from the dependency graph.
// It generates a 16-character hex string (64-bit hash) based on sorted type names,
// constructor names, IsField flags, ExpressionText (for Variable nodes),
// ConstructorPkgPath (conditional: only when it differs from PackagePath), and their dependencies.
// Using 64-bit reduces collision probability compared to 32-bit hashes.
//
// The hash is computed from node fields (ConstructorName, Dependencies, ExpressionText) which must
// be properly initialized. Empty strings and nil slices are handled gracefully.
// When ExpressionText is empty (Provider/Injector nodes), no additional bytes are written,
// preserving existing hash values for projects that do not use Variable annotations.
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

		// ExpressionText (for Variable nodes — affects generated code)
		// When ExpressionText is empty (Provider/Injector nodes), no bytes are written,
		// preserving existing hash values for projects that do not use Variables.
		if node.ExpressionText != "" {
			h.Write([]byte(node.ExpressionText))
			h.Write([]byte{0})
		}

		// ConstructorPkgPath (for cross-package Provide nodes — affects generated code)
		// Only included when it differs from PackagePath (i.e., the constructor
		// function lives in a different package than the return type).
		// When equal or empty, no bytes are written, preserving existing hash values.
		if node.ConstructorPkgPath != "" && node.ConstructorPkgPath != node.PackagePath {
			h.Write([]byte(node.ConstructorPkgPath))
			h.Write([]byte{0})
		}

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

// ComputeContainerHash computes a deterministic hash from the dependency graph
// and optional container field definitions. When containerDef is nil, it behaves
// identically to ComputeGraphHash.
func ComputeContainerHash(g *graph.Graph, containerDef *detect.ContainerDefinition) string {
	if g == nil {
		return "0000000000000000"
	}

	// Collect all type names and sort them for determinism
	types := slices.Sorted(maps.Keys(g.Nodes))

	// Build a deterministic string representation (same as ComputeGraphHash)
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

		// ExpressionText (for Variable nodes — affects generated code)
		if node.ExpressionText != "" {
			h.Write([]byte(node.ExpressionText))
			h.Write([]byte{0})
		}

		// ConstructorPkgPath (for cross-package Provide nodes — affects generated code)
		// Only included when it differs from PackagePath (i.e., the constructor
		// function lives in a different package than the return type).
		// When equal or empty, no bytes are written, preserving existing hash values.
		if node.ConstructorPkgPath != "" && node.ConstructorPkgPath != node.PackagePath {
			h.Write([]byte(node.ConstructorPkgPath))
			h.Write([]byte{0})
		}

		// Add dependencies in sorted order
		sortedDeps := make([]string, len(node.Dependencies))
		copy(sortedDeps, node.Dependencies)
		sort.Strings(sortedDeps)

		for _, dep := range sortedDeps {
			h.Write([]byte(dep))
			h.Write([]byte{0}) // separator
		}
	}

	// Append container field data if present
	if containerDef != nil {
		for _, field := range containerDef.Fields {
			h.Write([]byte(field.Name))
			h.Write([]byte{0})
			h.Write([]byte(field.TypeString))
			h.Write([]byte{0})
			h.Write([]byte(field.Tag))
			h.Write([]byte{0})
		}
	}

	// Return first hashLength hex characters (64-bit)
	fullHash := hex.EncodeToString(h.Sum(nil))
	if len(fullHash) >= hashLength {
		return fullHash[:hashLength]
	}
	return fullHash
}
