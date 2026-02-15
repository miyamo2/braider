package detect

import (
	"debug/buildinfo"
	"go/types"
	"os"
	"sync"

	"golang.org/x/tools/go/analysis"
)

var (
	resolvedModulePathOnce sync.Once
	resolvedModulePath     string
)

// resolveModulePath uses debug/buildinfo to determine the module path
// of the running binary, enabling support for forked repositories.
func resolveModulePath() string {
	resolvedModulePathOnce.Do(func() {
		exe, err := os.Executable()
		if err != nil {
			return
		}
		info, err := buildinfo.ReadFile(exe)
		if err != nil {
			return
		}
		resolvedModulePath = info.Main.Path
	})
	return resolvedModulePath
}

// internalAnnotationPkgPath returns the dynamically resolved package path
// for internal/annotation based on the current module path.
func internalAnnotationPkgPath() string {
	modPath := resolveModulePath()
	if modPath == "" {
		return ""
	}
	return modPath + "/internal/annotation"
}

// MarkerInterfaces holds resolved marker interfaces from internal/annotation.
// These are used with types.Implements to identify annotation types.
type MarkerInterfaces struct {
	Injectable            *types.Interface
	Provider              *types.Interface
	Variable              *types.Interface
	InjectableDefault     *types.Interface
	InjectableTyped       *types.Interface
	InjectableNamed       *types.Interface
	InjectableWithoutCtor *types.Interface
	ProviderDefault       *types.Interface
	ProviderTyped         *types.Interface
	ProviderNamed         *types.Interface
	VariableDefault       *types.Interface
	VariableTyped         *types.Interface
	VariableNamed         *types.Interface
}

// markerResolver resolves and caches marker interfaces from internal/annotation.
// The cache is keyed by the *types.Package pointer identity of the internal/annotation
// package, ensuring correctness across different type systems (e.g., separate test runs).
type markerResolver struct {
	mu      sync.Mutex
	pkg     *types.Package   // the internal/annotation package pointer used for cache validation
	markers *MarkerInterfaces // cached result
}

var globalMarkerResolver = &markerResolver{}

// resolveMarkers attempts to find the internal/annotation package in the import graph
// and extract all marker interfaces. Returns nil if the package is not reachable.
// Thread-safe; result is cached per internal/annotation package identity.
func resolveMarkers(pass *analysis.Pass) *MarkerInterfaces {
	return globalMarkerResolver.resolve(pass)
}

func (r *markerResolver) resolve(pass *analysis.Pass) *MarkerInterfaces {
	annPkg := findInternalAnnotationPkg(pass.Pkg)
	if annPkg == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Cache hit: same internal/annotation package object
	if r.pkg == annPkg {
		return r.markers
	}

	// Cache miss: resolve and update
	r.markers = &MarkerInterfaces{
		Injectable:            lookupMarkerInterface(annPkg, "Injectable"),
		Provider:              lookupMarkerInterface(annPkg, "Provider"),
		Variable:              lookupMarkerInterface(annPkg, "Variable"),
		InjectableDefault:     lookupMarkerInterface(annPkg, "InjectableDefault"),
		InjectableTyped:       lookupMarkerInterface(annPkg, "InjectableTyped"),
		InjectableNamed:       lookupMarkerInterface(annPkg, "InjectableNamed"),
		InjectableWithoutCtor: lookupMarkerInterface(annPkg, "InjectableWithoutConstructor"),
		ProviderDefault:       lookupMarkerInterface(annPkg, "ProviderDefault"),
		ProviderTyped:         lookupMarkerInterface(annPkg, "ProviderTyped"),
		ProviderNamed:         lookupMarkerInterface(annPkg, "ProviderNamed"),
		VariableDefault:       lookupMarkerInterface(annPkg, "VariableDefault"),
		VariableTyped:         lookupMarkerInterface(annPkg, "VariableTyped"),
		VariableNamed:         lookupMarkerInterface(annPkg, "VariableNamed"),
	}
	r.pkg = annPkg
	return r.markers
}

// findInternalAnnotationPkg walks the import graph (depth-limited to 2 levels)
// to find the internal/annotation package using the dynamically resolved path.
func findInternalAnnotationPkg(pkg *types.Package) *types.Package {
	if pkg == nil {
		return nil
	}

	targetPath := internalAnnotationPkgPath()
	if targetPath == "" {
		return nil
	}

	for _, imp := range pkg.Imports() {
		if imp.Path() == targetPath {
			return imp
		}
		for _, transitive := range imp.Imports() {
			if transitive.Path() == targetPath {
				return transitive
			}
		}
	}
	return nil
}

// lookupMarkerInterface looks up a named interface type from a package's scope.
func lookupMarkerInterface(pkg *types.Package, name string) *types.Interface {
	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		return nil
	}
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return nil
	}
	iface, ok := tn.Type().Underlying().(*types.Interface)
	if !ok {
		return nil
	}
	return iface
}
