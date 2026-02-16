package detect

import (
	"debug/buildinfo"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
	"golang.org/x/tools/go/packages"
)

var _ = annotation.Provide[provide.Default](ResolveMarkers)

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

var (
	resolveMarkersOnce sync.Once
	resolvedMarkers    *MarkerInterfaces
)

// ResolveMarkers loads the internal/annotation package via packages.Load and
// returns the resolved marker interfaces. Returns nil if resolution fails.
// Thread-safe; result is computed once and cached.
func ResolveMarkers() *MarkerInterfaces {
	resolveMarkersOnce.Do(func() {
		modPath := resolveModulePath()
		if modPath == "" {
			return
		}

		annPkg := loadInternalAnnotationPkg(modPath)
		if annPkg == nil {
			return
		}

		resolvedMarkers = &MarkerInterfaces{
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
	return resolvedMarkers
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
