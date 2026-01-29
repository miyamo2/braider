package registry

import (
	"sort"
	"sync"
)

// GlobalInjectorRegistry is the singleton instance used by all analyzers.
// DependencyAnalyzer registers discovered injectors; AppAnalyzer retrieves
// them for bootstrap generation.
var GlobalInjectorRegistry = NewInjectorRegistry()

// InjectorInfo contains information about an Inject struct.
// These are constructor generation targets that become fields in the dependency struct.
type InjectorInfo struct {
	// TypeName is the fully qualified type name (e.g., "example.com/service.UserService")
	TypeName string
	// PackagePath is the import path of the package (e.g., "example.com/service")
	PackagePath string
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
}

// InjectorRegistry stores all discovered injector structs globally.
// Thread-safe for potential parallel analyzer execution.
type InjectorRegistry struct {
	mu        sync.Mutex
	injectors map[string]*InjectorInfo
}

// NewInjectorRegistry creates a new empty registry.
func NewInjectorRegistry() *InjectorRegistry {
	return &InjectorRegistry{
		injectors: make(map[string]*InjectorInfo),
	}
}

// Register adds an injector struct to the registry.
// If an injector with the same TypeName already exists, it will be overwritten.
func (r *InjectorRegistry) Register(info *InjectorInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.injectors[info.TypeName] = info
}

// GetAll returns all registered injector structs.
// The returned slice is sorted alphabetically by TypeName for deterministic output.
// Returns a copy of the slice to prevent external mutation.
func (r *InjectorRegistry) GetAll() []*InjectorInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]*InjectorInfo, 0, len(r.injectors))
	for _, info := range r.injectors {
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

// Get retrieves an injector by fully qualified type name.
// Returns nil if not found.
func (r *InjectorRegistry) Get(typeName string) *InjectorInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.injectors[typeName]
}

// Clear removes all entries. Used for testing.
func (r *InjectorRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.injectors = make(map[string]*InjectorInfo)
}
