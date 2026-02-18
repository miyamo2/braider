// Package graph provides dependency graph construction and analysis
// for bootstrap code generation.
package graph

import (
	"fmt"
	"strings"

	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
	"golang.org/x/tools/go/analysis"
)

// InterfaceRegistry maps interface types to their implementing injectable structs.
// It supports both provider (annotation.Provide) and injector (annotation.Injectable) structs.
type InterfaceRegistry struct {
	// interfaces maps interface type name to list of implementing type names
	interfaces map[string][]string
}

var _ = annotation.Provide[provide.Default](NewInterfaceRegistry)

// NewInterfaceRegistry creates a new empty interface registry.
func NewInterfaceRegistry() *InterfaceRegistry {
	return &InterfaceRegistry{
		interfaces: make(map[string][]string),
	}
}

// Clear clears the registry for reuse.
func (r *InterfaceRegistry) Clear() {
	for k := range r.interfaces {
		delete(r.interfaces, k)
	}
}

// Build constructs the registry from all registered providers, injectors, and variables.
// It uses the Implements field from ProviderInfo, InjectorInfo, and VariableInfo which is
// populated by DependencyAnalyzer using go/types.Implements().
func (r *InterfaceRegistry) Build(
	pass *analysis.Pass,
	providers []*registry.ProviderInfo,
	injectors []*registry.InjectorInfo,
	variables []*registry.VariableInfo,
) error {
	// Process providers
	for _, provider := range providers {
		for _, iface := range provider.Implements {
			r.interfaces[iface] = append(r.interfaces[iface], provider.TypeName)
		}
	}

	// Process injectors
	for _, injector := range injectors {
		for _, iface := range injector.Implements {
			r.interfaces[iface] = append(r.interfaces[iface], injector.TypeName)
		}
	}

	// Process variables (required for Typed[I] resolution)
	for _, variable := range variables {
		for _, iface := range variable.Implements {
			r.interfaces[iface] = append(r.interfaces[iface], variable.TypeName)
		}
	}

	return nil
}

// Resolve finds the injectable struct implementing the given interface.
// Returns the fully qualified type name of the implementation.
// Returns AmbiguousImplementationError if multiple implementations exist.
// Returns UnresolvedInterfaceError if no implementation found.
func (r *InterfaceRegistry) Resolve(ifaceType string) (string, error) {
	impls, ok := r.interfaces[ifaceType]
	if !ok || len(impls) == 0 {
		return "", &UnresolvedInterfaceError{
			InterfaceType: ifaceType,
		}
	}

	if len(impls) > 1 {
		return "", &AmbiguousImplementationError{
			InterfaceType:   ifaceType,
			Implementations: impls,
		}
	}

	return impls[0], nil
}

// AmbiguousImplementationError indicates multiple structs implement an interface.
type AmbiguousImplementationError struct {
	InterfaceType   string   // Fully qualified interface type name
	Implementations []string // List of implementing types
}

func (e *AmbiguousImplementationError) Error() string {
	return fmt.Sprintf(
		"multiple injectable structs implement interface %s: %s",
		e.InterfaceType,
		strings.Join(e.Implementations, ", "),
	)
}

// UnresolvedInterfaceError indicates no injectable struct implements a required interface.
type UnresolvedInterfaceError struct {
	InterfaceType string // Fully qualified interface type name
}

func (e *UnresolvedInterfaceError) Error() string {
	return fmt.Sprintf(
		"no injectable struct implements interface %s; add annotation.Provide or annotation.Injectable to an implementing struct or change parameter to concrete type",
		e.InterfaceType,
	)
}
