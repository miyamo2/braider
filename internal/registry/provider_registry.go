// Package registry provides global registries for storing discovered
// provider and injector structs across packages during analysis.
// These registries enable cross-package dependency discovery for
// bootstrap code generation.
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

// ProviderInfo contains information about a Provide annotation.
// These are dependency providers that become fields in the bootstrap dependency struct.
type ProviderInfo struct {
	// TypeName is the fully qualified type name (e.g., "example.com/repo.UserRepository")
	TypeName string
	// PackagePath is the import path of the return type's package (e.g., "example.com/repo")
	PackagePath string
	// PackageName is the actual package name of the return type from go/types.Package (e.g., "repo")
	PackageName string
	// LocalName is the type name without package path (e.g., "UserRepository")
	LocalName string
	// ConstructorName is the constructor function name (e.g., "NewUserRepository")
	ConstructorName string
	// ConstructorPkgPath is the import path of the package where the constructor function is defined.
	// This may differ from PackagePath when the constructor returns a type from a different package.
	ConstructorPkgPath string
	// ConstructorPkgName is the package name where the constructor function is defined.
	ConstructorPkgName string
	// Dependencies contains fully qualified types of constructor parameters
	Dependencies []string
	// Implements contains fully qualified interface types this struct implements
	Implements []string
	// IsPending indicates whether the constructor is being generated in the current pass (true)
	// or already exists on disk (false). Typically false for Provide annotations as they require
	// existing constructors, but included for consistency with InjectorInfo.
	IsPending bool

	// NEW: Option-derived fields for annotation refinement feature
	// RegisteredType is the type to use for registration - interface type for Typed[I], return type otherwise
	RegisteredType types.Type
	// Name is the provider name from Named[N] option, empty if unnamed
	Name string
	// OptionMetadata contains parsed option configuration from type parameters
	OptionMetadata detect.OptionMetadata
}

func (i *ProviderInfo) GetTypeName() string {
	return i.TypeName
}

func (i *ProviderInfo) GetDependencies() []string {
	return i.Dependencies
}

func (i *ProviderInfo) GetName() string {
	return i.Name
}

var _ = annotation.Provide[provide.Default](NewProviderRegistry)

// ProviderRegistry stores all discovered provider structs globally.
// Thread-safe for potential parallel analyzer execution.
// Uses RWMutex to allow concurrent reads for improved performance.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]map[string]*ProviderInfo
}

// NewProviderRegistry creates a new empty registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]map[string]*ProviderInfo),
	}
}

// Register adds a provider struct to the registry.
// Returns an error if a duplicate (TypeName, Name) pair is detected with a non-empty name.
// If a provider with the same TypeName already exists and names don't conflict, it will be overwritten.
func (r *ProviderRegistry) Register(info *ProviderInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.providers[info.TypeName] == nil {
		r.providers[info.TypeName] = make(map[string]*ProviderInfo)
	}
	if existing, ok := r.providers[info.TypeName][info.Name]; ok {
		if existing.Name != "" && existing.Name == info.Name {
			return fmt.Errorf(
				"duplicate named dependency: type %s with name %q already registered from %s",
				info.TypeName, info.Name, existing.PackagePath,
			)
		}
	}
	r.providers[info.TypeName][info.Name] = info
	return nil
}

// GetAll returns all registered provider structs.
// The returned slice is sorted alphabetically by TypeName for deterministic output.
// Returns a copy of the slice to prevent external mutation.
func (r *ProviderRegistry) GetAll() []*ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	size := 0
	for _, inner := range r.providers {
		size += len(inner)
	}
	result := make([]*ProviderInfo, 0, size)
	for _, inner := range r.providers {
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

// GetByName retrieves a named provider by fully qualified type name and name.
// Returns (info, true) if found with matching name, (nil, false) otherwise.
// This supports named dependency lookup for Provider[provide.Named[N]] annotations.
func (r *ProviderRegistry) GetByName(typeName, name string) (*ProviderInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inner, ok := r.providers[typeName]
	if !ok {
		return nil, false
	}
	info, exists := inner[name]
	return info, exists
}
