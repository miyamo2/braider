package detect

import (
	"debug/buildinfo"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"golang.org/x/tools/go/packages"
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
// Uses packages.Load to load the internal/annotation package via the source directory
// obtained from runtime.Caller, combined with the module path from debug/buildinfo.
type markerResolver struct {
	once    sync.Once
	markers *MarkerInterfaces
}

var globalMarkerResolver = &markerResolver{}

// resolveMarkers returns the cached marker interfaces from internal/annotation.
// Thread-safe; result is computed once per binary lifetime.
func resolveMarkers() *MarkerInterfaces {
	return globalMarkerResolver.resolve()
}

func (r *markerResolver) resolve() *MarkerInterfaces {
	r.once.Do(func() {
		modPath := resolveModulePath()
		if modPath == "" {
			return
		}

		annPkg := loadInternalAnnotationPkg(modPath)
		if annPkg == nil {
			return
		}

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
	})
	return r.markers
}

// loadInternalAnnotationPkg loads the internal/annotation package using
// packages.Load with the source directory obtained from runtime.Caller.
// This works regardless of the analyzed module's identity because it loads
// from the braider module's own source tree (local build or module cache).
func loadInternalAnnotationPkg(modulePath string) *types.Package {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil
	}
	thisDir := filepath.Dir(thisFile)

	cfg := &packages.Config{
		Dir:  thisDir,
		Mode: packages.NeedTypes | packages.NeedName,
	}
	pkgs, err := packages.Load(cfg, "../annotation")
	if err != nil || len(pkgs) == 0 {
		return nil
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil
	}

	// Verify the loaded package path matches the expected path
	expectedPath := modulePath + "/internal/annotation"
	if pkg.Types == nil || pkg.Types.Path() != expectedPath {
		return nil
	}

	return pkg.Types
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

// resetMarkerResolverForTest resets the global marker resolver state.
// This is intended for use in tests only.
func resetMarkerResolverForTest() {
	globalMarkerResolver = &markerResolver{}
}
