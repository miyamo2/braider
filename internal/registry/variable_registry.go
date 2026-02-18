package registry

import (
	"fmt"
	"go/types"
	"sort"
	"sync"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// VariableInfo contains information about a Variable annotation.
// These are pre-existing values that become local variable assignments in the bootstrap IIFE.
type VariableInfo struct {
	// TypeName is the fully qualified type name (e.g., "os.File")
	TypeName string
	// PackagePath is the import path of the package (e.g., "os")
	PackagePath string
	// PackageName is the actual package name from go/types.Package (e.g., "os")
	PackageName string
	// LocalName is the type name without package path (e.g., "File")
	LocalName string
	// ExpressionText is the formatted source text of the argument expression (e.g., "os.Stdout")
	ExpressionText string
	// ExpressionPkgs contains package paths referenced by the expression
	ExpressionPkgs []string
	// ExpressionPkgNames contains package names parallel to ExpressionPkgs (for collision detection)
	ExpressionPkgNames []string
	// IsQualified indicates whether the expression is already package-qualified (SelectorExpr)
	IsQualified bool
	// Dependencies is always empty for Variables (Variables have no dependencies)
	Dependencies []string
	// Implements contains fully qualified interface types the argument type implements
	Implements []string
	// RegisteredType is the interface type for Typed[I], argument type otherwise
	RegisteredType types.Type
	// Name is the variable name from Named[N] option, empty if unnamed
	Name string
	// OptionMetadata contains parsed option configuration from type parameters
	OptionMetadata detect.OptionMetadata
}

// GetTypeName returns the fully qualified type name of the variable.
// Implements the dependencyInfo interface for graph edge building.
func (i *VariableInfo) GetTypeName() string {
	return i.TypeName
}

// GetDependencies returns the dependencies of the variable.
// Always returns an empty slice since Variables have no dependencies.
// Implements the dependencyInfo interface for graph edge building.
func (i *VariableInfo) GetDependencies() []string {
	return i.Dependencies
}

// GetName returns the name of the variable from Named[N] option.
// Returns empty string if unnamed.
// Implements the dependencyInfo interface for graph edge building.
func (i *VariableInfo) GetName() string {
	return i.Name
}

// VariableRegistry stores all discovered Variable annotations globally.
// Thread-safe for potential parallel analyzer execution.
// Uses RWMutex to allow concurrent reads for improved performance.
type VariableRegistry struct {
	mu        sync.RWMutex
	variables map[string]map[string]*VariableInfo
}

var _ = annotation.Provide[provide.Default](NewVariableRegistry)

// NewVariableRegistry creates a new empty registry.
func NewVariableRegistry() *VariableRegistry {
	return &VariableRegistry{
		variables: make(map[string]map[string]*VariableInfo),
	}
}

// Register adds a variable to the registry.
// Returns an error if a duplicate (TypeName, Name) pair is detected with a non-empty name.
// If a variable with the same TypeName already exists and names don't conflict, it will be overwritten.
func (r *VariableRegistry) Register(info *VariableInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.variables[info.TypeName] == nil {
		r.variables[info.TypeName] = make(map[string]*VariableInfo)
	}
	if existing, ok := r.variables[info.TypeName][info.Name]; ok {
		if existing.Name != "" && existing.Name == info.Name {
			return fmt.Errorf(
				"duplicate named dependency: type %s with name %q already registered from %s",
				info.TypeName, info.Name, existing.PackagePath,
			)
		}
	}
	r.variables[info.TypeName][info.Name] = info
	return nil
}

// GetAll returns all registered variables.
// The returned slice is sorted alphabetically by TypeName, then by Name for deterministic output.
// Returns a copy of the slice to prevent external mutation.
func (r *VariableRegistry) GetAll() []*VariableInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	size := 0
	for _, inner := range r.variables {
		size += len(inner)
	}
	result := make([]*VariableInfo, 0, size)
	for _, inner := range r.variables {
		for _, info := range inner {
			result = append(result, info)
		}
	}

	// Sort alphabetically by TypeName, then by Name for deterministic output
	sort.Slice(
		result, func(i, j int) bool {
			if result[i].TypeName != result[j].TypeName {
				return result[i].TypeName < result[j].TypeName
			}
			return result[i].Name < result[j].Name
		},
	)

	return result
}

// GetByName retrieves a named variable by fully qualified type name and name.
// Returns (info, true) if found with matching name, (nil, false) otherwise.
// This supports named dependency lookup for Variable[variable.Named[N]] annotations.
func (r *VariableRegistry) GetByName(typeName, name string) (*VariableInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inner, ok := r.variables[typeName]
	if !ok {
		return nil, false
	}
	info, exists := inner[name]
	return info, exists
}

// GetNamesByType returns a sorted slice of non-empty names registered for the given type name.
// Returns nil if no entries exist for the type or no entries have non-empty names.
func (r *VariableRegistry) GetNamesByType(typeName string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inner, ok := r.variables[typeName]
	if !ok {
		return nil
	}
	var names []string
	for name := range inner {
		if name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	sort.Strings(names)
	return names
}
