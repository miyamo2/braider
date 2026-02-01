package graph

import (
	"fmt"

	"github.com/miyamo2/braider/internal/registry"
	"golang.org/x/tools/go/analysis"
)

// Graph represents the dependency graph.
//
// Structure:
//   - Nodes: Map of fully qualified type names to Node structs
//   - Edges: Adjacency list representing dependencies
//
// Edge direction: from -> []to
//   - "from" depends on "to" (from's constructor requires to as parameter)
//   - Example: If UserService depends on UserRepository,
//     Edges["UserService"] = ["UserRepository"]
//
// InDegree interpretation (reverse edges):
//   - InDegree counts how many types depend on this node
//   - Used by Kahn's algorithm for topological sorting
//   - InDegree == 0 means no dependencies, can be initialized first
//   - As nodes are processed, InDegree is decremented for dependents
type Graph struct {
	Nodes map[string]*Node    // Keyed by fully qualified type name
	Edges map[string][]string // Dependencies: from -> []to
}

// Node represents an injectable type in the graph.
type Node struct {
	TypeName        string   // Fully qualified type name
	PackagePath     string   // Import path
	PackageName     string   // Actual package name from go/types.Package
	PackageAlias    string   // Alias when name collision occurs (empty = no alias)
	LocalName       string   // Type name without package
	ConstructorName string   // New<TypeName>
	Dependencies    []string // Types this depends on
	InDegree        int      // Number of types that depend on this node (for Kahn's algorithm)
	IsField         bool     // True for Inject structs (dependency struct fields), false for Provide structs (local variables only)
}

// UnresolvableTypeError represents a dependency type that cannot be resolved.
type UnresolvableTypeError struct {
	TypeName string
}

func (e *UnresolvableTypeError) Error() string {
	return fmt.Sprintf("unresolvable dependency type: %s", e.TypeName)
}

// DependencyGraphBuilder builds the dependency graph for injectable types.
type DependencyGraphBuilder struct {
	interfaceRegistry *InterfaceRegistry
}

// NewDependencyGraphBuilder creates a new dependency graph builder.
func NewDependencyGraphBuilder() *DependencyGraphBuilder {
	return &DependencyGraphBuilder{
		interfaceRegistry: NewInterfaceRegistry(),
	}
}

// BuildGraph constructs the dependency graph from registered providers and injectors.
// Injectables are retrieved from GlobalProviderRegistry and GlobalInjectorRegistry.
//
// This method executes sequentially and is NOT safe for concurrent calls on the same graph.
// Each analyzer run should create its own graph instance.
//
// Returns error if:
// - An interface parameter has no injectable implementation
// - Multiple injectable structs implement the same required interface
func (b *DependencyGraphBuilder) BuildGraph(
	pass *analysis.Pass,
	providers []*registry.ProviderInfo,
	injectors []*registry.InjectorInfo,
) (*Graph, error) {
	// Initialize graph
	graph := &Graph{
		Nodes: make(map[string]*Node),
		Edges: make(map[string][]string),
	}

	// Build interface registry for dependency resolution
	// Clear existing registry for reuse (avoid repeated allocations)
	b.interfaceRegistry.Clear()
	if err := b.interfaceRegistry.Build(pass, providers, injectors); err != nil {
		return nil, err
	}

	// Add provider nodes (IsField = false)
	for _, provider := range providers {
		node := &Node{
			TypeName:        provider.TypeName,
			PackagePath:     provider.PackagePath,
			PackageName:     provider.PackageName,
			LocalName:       provider.LocalName,
			ConstructorName: provider.ConstructorName,
			Dependencies:    []string{},
			InDegree:        0,
			IsField:         false, // Providers are local variables in IIFE
		}
		graph.Nodes[provider.TypeName] = node
	}

	// Add injector nodes (IsField = true)
	for _, injector := range injectors {
		node := &Node{
			TypeName:        injector.TypeName,
			PackagePath:     injector.PackagePath,
			PackageName:     injector.PackageName,
			LocalName:       injector.LocalName,
			ConstructorName: injector.ConstructorName,
			Dependencies:    []string{},
			InDegree:        0,
			IsField:         true, // Injectors are fields in dependency struct
		}
		graph.Nodes[injector.TypeName] = node
	}

	// Build edges from dependencies
	if err := b.buildEdgesFromProviders(graph, providers); err != nil {
		return nil, err
	}
	if err := b.buildEdgesFromInjectors(graph, injectors); err != nil {
		return nil, err
	}

	return graph, nil
}

// buildEdgesFromProviders builds edges from provider dependencies.
func (b *DependencyGraphBuilder) buildEdgesFromProviders(
	graph *Graph,
	providers []*registry.ProviderInfo,
) error {
	for _, provider := range providers {
		node := graph.Nodes[provider.TypeName]
		edges := make([]string, 0, len(provider.Dependencies))
		for _, dep := range provider.Dependencies {
			// Resolve dependency (may be interface type)
			resolvedDep, err := b.resolveDependency(graph, dep)
			if err != nil {
				return err
			}
			edges = append(edges, resolvedDep)

			// Update node dependencies
			node.Dependencies = append(node.Dependencies, resolvedDep)

			// Increment in-degree of the current node (it depends on resolvedDep)
			// For Kahn's algorithm: nodes with InDegree=0 are constructed first
			// Since this node depends on resolvedDep, it must wait until resolvedDep is constructed
			node.InDegree++
		}
		graph.Edges[provider.TypeName] = edges
	}
	return nil
}

// buildEdgesFromInjectors builds edges from injector dependencies.
func (b *DependencyGraphBuilder) buildEdgesFromInjectors(
	graph *Graph,
	injectors []*registry.InjectorInfo,
) error {
	for _, injector := range injectors {
		node := graph.Nodes[injector.TypeName]
		edges := make([]string, 0, len(injector.Dependencies))
		for _, dep := range injector.Dependencies {
			// Resolve dependency (may be interface type)
			resolvedDep, err := b.resolveDependency(graph, dep)
			if err != nil {
				return err
			}
			edges = append(edges, resolvedDep)

			// Update node dependencies
			node.Dependencies = append(node.Dependencies, resolvedDep)

			// Increment in-degree of the current node (it depends on resolvedDep)
			// For Kahn's algorithm: nodes with InDegree=0 are constructed first
			// Since this node depends on resolvedDep, it must wait until resolvedDep is constructed
			node.InDegree++
		}
		graph.Edges[injector.TypeName] = edges
	}
	return nil
}

// resolveDependency resolves a dependency type name, handling interface types.
// If the dependency is a concrete type already registered in the graph, return as-is.
// If the dependency is an interface type, resolve to implementing injectable struct.
// Returns error if the dependency cannot be resolved.
func (b *DependencyGraphBuilder) resolveDependency(graph *Graph, typeName string) (string, error) {
	// Try to resolve as interface first
	if impl, err := b.interfaceRegistry.Resolve(typeName); err == nil {
		return impl, nil
	} else {
		// Check if error is AmbiguousImplementationError - this is a real error
		if _, ok := err.(*AmbiguousImplementationError); ok {
			return "", err
		}
		// UnresolvedInterfaceError is not necessarily an error - the type may be concrete
		// Continue to check if it's a concrete type in the graph
	}

	// Check if it's a concrete type in the graph
	if graph.Nodes[typeName] != nil {
		return typeName, nil
	}

	// Neither interface nor concrete type found
	// Return UnresolvedInterfaceError if it was an interface lookup failure
	return "", &UnresolvableTypeError{TypeName: typeName}
}
