package graph

import (
	"github.com/miyamo2/braider/internal/registry"
	"golang.org/x/tools/go/analysis"
)

// Graph represents the dependency graph.
type Graph struct {
	Nodes map[string]*Node    // Keyed by fully qualified type name
	Edges map[string][]string // Dependencies: from -> []to
}

// Node represents an injectable type in the graph.
type Node struct {
	TypeName        string   // Fully qualified type name
	PackagePath     string   // Import path
	LocalName       string   // Type name without package
	ConstructorName string   // New<TypeName>
	Dependencies    []string // Types this depends on
	InDegree        int      // For Kahn's algorithm
	IsInject        bool     // True for Inject structs (fields), false for Provide structs (local vars)
}

// DependencyGraphBuilder builds the dependency graph for injectable types.
type DependencyGraphBuilder struct {
	interfaceRegistry *InterfaceRegistry
}

// NewDependencyGraphBuilder creates a new dependency graph builder.
func NewDependencyGraphBuilder() *DependencyGraphBuilder {
	return &DependencyGraphBuilder{}
}

// BuildGraph constructs the dependency graph from registered providers and injectors.
// Injectables are retrieved from GlobalProviderRegistry and GlobalInjectorRegistry.
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
	b.interfaceRegistry = NewInterfaceRegistry()
	if err := b.interfaceRegistry.Build(pass, providers, injectors); err != nil {
		return nil, err
	}

	// Add provider nodes (IsInject = false)
	for _, provider := range providers {
		node := &Node{
			TypeName:        provider.TypeName,
			PackagePath:     provider.PackagePath,
			LocalName:       provider.LocalName,
			ConstructorName: provider.ConstructorName,
			Dependencies:    []string{},
			InDegree:        0,
			IsInject:        false, // Providers are local variables in IIFE
		}
		graph.Nodes[provider.TypeName] = node
	}

	// Add injector nodes (IsInject = true)
	for _, injector := range injectors {
		node := &Node{
			TypeName:        injector.TypeName,
			PackagePath:     injector.PackagePath,
			LocalName:       injector.LocalName,
			ConstructorName: injector.ConstructorName,
			Dependencies:    []string{},
			InDegree:        0,
			IsInject:        true, // Injectors are fields in dependency struct
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
		edges := []string{}
		for _, dep := range provider.Dependencies {
			// Resolve dependency (may be interface type)
			resolvedDep, err := b.resolveDependency(dep)
			if err != nil {
				return err
			}
			edges = append(edges, resolvedDep)

			// Update node dependencies
			node := graph.Nodes[provider.TypeName]
			node.Dependencies = append(node.Dependencies, resolvedDep)

			// Increment in-degree of dependency node
			if depNode, ok := graph.Nodes[resolvedDep]; ok {
				depNode.InDegree++
			}
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
		edges := []string{}
		for _, dep := range injector.Dependencies {
			// Resolve dependency (may be interface type)
			resolvedDep, err := b.resolveDependency(dep)
			if err != nil {
				return err
			}
			edges = append(edges, resolvedDep)

			// Update node dependencies
			node := graph.Nodes[injector.TypeName]
			node.Dependencies = append(node.Dependencies, resolvedDep)

			// Increment in-degree of dependency node
			if depNode, ok := graph.Nodes[resolvedDep]; ok {
				depNode.InDegree++
			}
		}
		graph.Edges[injector.TypeName] = edges
	}
	return nil
}

// resolveDependency resolves a dependency type name, handling interface types.
// If the dependency is a concrete type already registered in the graph, return as-is.
// If the dependency is an interface type, resolve to implementing injectable struct.
func (b *DependencyGraphBuilder) resolveDependency(typeName string) (string, error) {
	// Try to resolve as interface first
	if impl, err := b.interfaceRegistry.Resolve(typeName); err == nil {
		return impl, nil
	}

	// If not found in interface registry, assume it's a concrete type
	// (will be validated later during graph construction)
	return typeName, nil
}
