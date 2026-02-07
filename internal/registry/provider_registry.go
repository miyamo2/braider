// Package registry provides global registries for storing discovered
// provider and injector structs across packages during analysis.
// These registries enable cross-package dependency discovery for
// bootstrap code generation.
package registry

import (
	"go/types"
	"sort"
	"sync"

	"github.com/miyamo2/braider/internal/detect"
)

// ProviderInfo contains information about a Provide struct.
// These are dependency providers that become local variables in the bootstrap IIFE.
type ProviderInfo struct {
	// TypeName is the fully qualified type name (e.g., "example.com/repo.UserRepository")
	TypeName string
	// PackagePath is the import path of the package (e.g., "example.com/repo")
	PackagePath string
	// PackageName is the actual package name from go/types.Package (e.g., "repo")
	PackageName string
	// LocalName is the type name without package path (e.g., "UserRepository")
	LocalName string
	// ConstructorName is the constructor function name (e.g., "NewUserRepository")
	ConstructorName string
	// Dependencies contains fully qualified types of constructor parameters
	Dependencies []string
	// Implements contains fully qualified interface types this struct implements
	Implements []string
	// IsPending indicates whether the constructor is being generated in the current pass (true)
	// or already exists on disk (false). Typically false for Provide structs as they require
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

// ProviderRegistry stores all discovered provider structs globally.
// Thread-safe for potential parallel analyzer execution.
// Uses RWMutex to allow concurrent reads for improved performance.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]*ProviderInfo
}

// NewProviderRegistry creates a new empty registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]*ProviderInfo),
	}
}

// Register adds a provider struct to the registry.
// If a provider with the same TypeName already exists, it will be overwritten.
func (r *ProviderRegistry) Register(info *ProviderInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[info.TypeName] = info
}

// GetAll returns all registered provider structs.
// The returned slice is sorted alphabetically by TypeName for deterministic output.
// Returns a copy of the slice to prevent external mutation.
func (r *ProviderRegistry) GetAll() []*ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ProviderInfo, 0, len(r.providers))
	for _, info := range r.providers {
		result = append(result, info)
	}

	// Sort alphabetically by TypeName for deterministic output
	sort.Slice(
		result, func(i, j int) bool {
			return result[i].TypeName < result[j].TypeName
		},
	)

	return result
}

// Get retrieves a provider by fully qualified type name.
// Returns nil if not found.
func (r *ProviderRegistry) Get(typeName string) *ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[typeName]
}

// GetByName retrieves a named provider by fully qualified type name and name.
// Returns (info, true) if found with matching name, (nil, false) otherwise.
// This supports named dependency lookup for Provider[provide.Named[N]] annotations.
func (r *ProviderRegistry) GetByName(typeName, name string) (*ProviderInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.providers[typeName]
	if !exists {
		return nil, false
	}

	// Check if the name matches
	if info.Name != name {
		return nil, false
	}

	return info, true
}
