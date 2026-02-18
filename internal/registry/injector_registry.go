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

// InjectorInfo contains information about an Inject struct.
// These are constructor generation targets that become fields in the dependency struct.
type InjectorInfo struct {
	// TypeName is the fully qualified type name (e.g., "example.com/service.UserService")
	TypeName string
	// PackagePath is the import path of the package (e.g., "example.com/service")
	PackagePath string
	// PackageName is the actual package name from go/types.Package (e.g., "service")
	PackageName string
	// LocalName is the type name without package path (e.g., "UserService")
	LocalName string
	// ConstructorName is the constructor function name (e.g., "NewUserService")
	ConstructorName string
	// Dependencies contains fully qualified types of constructor parameters
	Dependencies []string
	// Implements contains fully qualified interface types this struct implements
	Implements []string
	// IsPending indicates whether the constructor is being generated in the current pass (true)
	// or already exists on disk (false). Enables single-pass constructor and bootstrap generation.
	IsPending bool

	// NEW: Option-derived fields for annotation refinement feature
	// RegisteredType is the type to use for registration - interface type for Typed[I], concrete type otherwise
	RegisteredType types.Type
	// Name is the dependency name from Named[N] option, empty if unnamed
	Name string
	// OptionMetadata contains parsed option configuration from type parameters
	OptionMetadata detect.OptionMetadata
}

func (i *InjectorInfo) GetTypeName() string {
	return i.TypeName
}

func (i *InjectorInfo) GetDependencies() []string {
	return i.Dependencies
}

func (i *InjectorInfo) GetName() string {
	return i.Name
}

var _ = annotation.Provide[provide.Default](NewInjectorRegistry)

// InjectorRegistry stores all discovered injector structs globally.
// Thread-safe for potential parallel analyzer execution.
// Uses RWMutex to allow concurrent reads for improved performance.
type InjectorRegistry struct {
	mu        sync.RWMutex
	injectors map[string]map[string]*InjectorInfo
}

// NewInjectorRegistry creates a new empty registry.
func NewInjectorRegistry() *InjectorRegistry {
	return &InjectorRegistry{
		injectors: make(map[string]map[string]*InjectorInfo),
	}
}

// Register adds an injector struct to the registry.
// Returns an error if a duplicate (TypeName, Name) pair is detected with a non-empty name.
// If an injector with the same TypeName already exists and names don't conflict, it will be overwritten.
func (r *InjectorRegistry) Register(info *InjectorInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.injectors[info.TypeName] == nil {
		r.injectors[info.TypeName] = make(map[string]*InjectorInfo)
	}
	if existing, ok := r.injectors[info.TypeName][info.Name]; ok {
		if existing.Name != "" && existing.Name == info.Name {
			return fmt.Errorf(
				"duplicate named dependency: type %s with name %q already registered from %s",
				info.TypeName, info.Name, existing.PackagePath,
			)
		}
	}
	r.injectors[info.TypeName][info.Name] = info
	return nil
}

// GetAll returns all registered injector structs.
// The returned slice is sorted alphabetically by TypeName for deterministic output.
// Returns a copy of the slice to prevent external mutation.
func (r *InjectorRegistry) GetAll() []*InjectorInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*InjectorInfo, 0)
	for _, inner := range r.injectors {
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

// GetByName retrieves a named injector by fully qualified type name and name.
// Returns (info, true) if found with matching name, (nil, false) otherwise.
// This supports named dependency lookup for Injectable[inject.Named[N]] annotations.
func (r *InjectorRegistry) GetByName(typeName, name string) (*InjectorInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inner, ok := r.injectors[typeName]
	if !ok {
		return nil, false
	}
	info, exists := inner[name]
	return info, exists
}
