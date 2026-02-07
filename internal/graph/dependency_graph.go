package graph

import (
	"fmt"
	"go/types"

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
	TypeName        string      // Fully qualified type name
	PackagePath     string      // Import path
	PackageName     string      // Actual package name from go/types.Package
	PackageAlias    string      // Alias when name collision occurs (empty = no alias)
	LocalName       string      // Type name without package
	ConstructorName string      // New<TypeName>
	Dependencies    []string    // Types this depends on
	InDegree        int         // Number of types that depend on this node (for Kahn's algorithm)
	IsField         bool        // True for Inject structs (dependency struct fields), false for Provide structs (local variables only)
	RegisteredType  types.Type  // Interface type for Typed[I], concrete type otherwise (nil = use concrete type)
	Name            string      // Dependency name from Named[N], empty if unnamed
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

// makeNodeKey creates a composite key for a dependency.
// For named dependencies, returns "TypeName#Name".
// For unnamed dependencies, returns "TypeName".
func makeNodeKey(typeName, name string) string {
	if name != "" {
		return typeName + "#" + name
	}
	return typeName
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
		nodeKey := makeNodeKey(provider.TypeName, provider.Name)
		node := &Node{
			TypeName:        provider.TypeName,
			PackagePath:     provider.PackagePath,
			PackageName:     provider.PackageName,
			LocalName:       provider.LocalName,
			ConstructorName: provider.ConstructorName,
			Dependencies:    []string{},
			InDegree:        0,
			IsField:         false, // Providers are local variables in IIFE
			RegisteredType:  provider.RegisteredType,
			Name:            provider.Name,
		}
		graph.Nodes[nodeKey] = node
	}

	// Add injector nodes (IsField = true)
	for _, injector := range injectors {
		nodeKey := makeNodeKey(injector.TypeName, injector.Name)
		node := &Node{
			TypeName:        injector.TypeName,
			PackagePath:     injector.PackagePath,
			PackageName:     injector.PackageName,
			LocalName:       injector.LocalName,
			ConstructorName: injector.ConstructorName,
			Dependencies:    []string{},
			InDegree:        0,
			IsField:         true, // Injectors are fields in dependency struct
			RegisteredType:  injector.RegisteredType,
			Name:            injector.Name,
		}
		graph.Nodes[nodeKey] = node
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
	return buildEdges(graph, providers, b)
}

// buildEdgesFromInjectors builds edges from injector dependencies.
func (b *DependencyGraphBuilder) buildEdgesFromInjectors(
	graph *Graph,
	injectors []*registry.InjectorInfo,
) error {
	return buildEdges(graph, injectors, b)
}

type dependencyInfo interface {
	GetTypeName() string
	GetDependencies() []string
	GetName() string
}

func buildEdges[T dependencyInfo](
	graph *Graph,
	infos []T,
	builder *DependencyGraphBuilder,
) error {
	for _, info := range infos {
		// Use composite key for the node
		nodeKey := makeNodeKey(info.GetTypeName(), info.GetName())
		node := graph.Nodes[nodeKey]
		edges := make([]string, 0, len(info.GetDependencies()))
		for _, dep := range info.GetDependencies() {
			// Resolve dependency (may be interface type or named dependency)
			resolvedDep, err := builder.resolveDependency(graph, dep)
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
		graph.Edges[nodeKey] = edges
	}
	return nil
}

// resolveDependency resolves a dependency type name, handling interface types and named dependencies.
// If the dependency is a composite key (TypeName#Name), return as-is if found in graph.
// If the dependency is a concrete type already registered in the graph, return as-is.
// If the dependency is an interface type, resolve to implementing injectable struct.
// Returns error if the dependency cannot be resolved.
func (b *DependencyGraphBuilder) resolveDependency(graph *Graph, typeName string) (string, error) {
	// Check if it's a composite key (named dependency) already in the graph
	if graph.Nodes[typeName] != nil {
		return typeName, nil
	}

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

	// Neither interface nor concrete type found
	// Return UnresolvedInterfaceError if it was an interface lookup failure
	return "", &UnresolvableTypeError{TypeName: typeName}
}
