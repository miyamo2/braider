package detect

import (
	"debug/buildinfo"
	"fmt"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
	"golang.org/x/tools/go/packages"
)

var _ = annotation.Provide[provide.Default](MustResolveMarkers)

var (
	resolvedModulePathOnce sync.Once
	resolvedModulePath     string
	resolvedModulePathErr  error
)

// resolveModulePath uses debug/buildinfo to determine the module path
// of the running binary, enabling support for forked repositories.
func resolveModulePath() (string, error) {
	resolvedModulePathOnce.Do(
		func() {
			exe, err := os.Executable()
			if err != nil {
				resolvedModulePathErr = fmt.Errorf("failed to locate braider executable: %w", err)
				return
			}
			info, err := buildinfo.ReadFile(exe)
			if err != nil {
				resolvedModulePathErr = fmt.Errorf("failed to read build info from %s: %w", exe, err)
				return
			}
			resolvedModulePath = info.Main.Path
		},
	)
	return resolvedModulePath, resolvedModulePathErr
}

// MarkerInterfaces holds resolved marker interfaces from internal/annotation.
// These are used with types.Implements to identify annotation types.
type MarkerInterfaces struct {
	App                   *types.Interface
	AppDefault            *types.Interface
	AppContainer          *types.Interface
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
	resolvedMarkersErr error
)

// MustResolveMarkers loads the internal/annotation package via packages.Load and
// returns the resolved marker interfaces. Panics if resolution fails.
// Thread-safe; result is computed once and cached.
func MustResolveMarkers() *MarkerInterfaces {
	v, err := ResolveMarkers()
	if err != nil {
		panic(err)
	}
	return v
}

// ResolveMarkers loads the internal/annotation package via packages.Load and
// returns the resolved marker interfaces. Returns an error if resolution fails.
// Thread-safe; result is computed once and cached.
func ResolveMarkers() (*MarkerInterfaces, error) {
	resolveMarkersOnce.Do(
		func() {
			modPath, err := resolveModulePath()
			if err != nil {
				resolvedMarkersErr = fmt.Errorf("failed to resolve module path: %w", err)
				return
			}
			if modPath == "" {
				resolvedMarkersErr = fmt.Errorf("module path is empty: binary may not contain module information")
				return
			}

			annPkg, err := loadInternalAnnotationPkg(modPath)
			if err != nil {
				resolvedMarkersErr = fmt.Errorf("failed to load annotation package: %w", err)
				return
			}

			resolvedMarkers = &MarkerInterfaces{
				App:                   lookupMarkerInterface(annPkg, "App"),
				AppDefault:            lookupMarkerInterface(annPkg, "AppDefault"),
				AppContainer:          lookupMarkerInterface(annPkg, "AppContainer"),
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

			// Validate that all marker interfaces were successfully resolved.
			missing := map[string]bool{
				"App":                   resolvedMarkers.App == nil,
				"AppDefault":            resolvedMarkers.AppDefault == nil,
				"AppContainer":          resolvedMarkers.AppContainer == nil,
				"Injectable":            resolvedMarkers.Injectable == nil,
				"Provider":              resolvedMarkers.Provider == nil,
				"Variable":              resolvedMarkers.Variable == nil,
				"InjectableDefault":     resolvedMarkers.InjectableDefault == nil,
				"InjectableTyped":       resolvedMarkers.InjectableTyped == nil,
				"InjectableNamed":       resolvedMarkers.InjectableNamed == nil,
				"InjectableWithoutCtor": resolvedMarkers.InjectableWithoutCtor == nil,
				"ProviderDefault":       resolvedMarkers.ProviderDefault == nil,
				"ProviderTyped":         resolvedMarkers.ProviderTyped == nil,
				"ProviderNamed":         resolvedMarkers.ProviderNamed == nil,
				"VariableDefault":       resolvedMarkers.VariableDefault == nil,
				"VariableTyped":         resolvedMarkers.VariableTyped == nil,
				"VariableNamed":         resolvedMarkers.VariableNamed == nil,
			}
			names := slices.Sorted(maps.Keys(missing))
			var missingNames []string
			for _, name := range names {
				if missing[name] {
					missingNames = append(missingNames, name)
				}
			}
			if len(missingNames) > 0 {
				resolvedMarkers = nil
				resolvedMarkersErr = fmt.Errorf("failed to resolve marker interfaces: %v", missingNames)
				return
			}
		},
	)
	return resolvedMarkers, resolvedMarkersErr
}

// loadInternalAnnotationPkg loads the internal/annotation package using
// packages.Load with the source directory obtained from runtime.Caller.
// This works regardless of the analyzed module's identity because it loads
// from the braider module's own source tree (local build or module cache).
func loadInternalAnnotationPkg(modulePath string) (*types.Package, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("failed to determine source file location via runtime.Caller")
	}
	thisDir := filepath.Dir(thisFile)

	cfg := &packages.Config{
		Dir:  thisDir,
		Mode: packages.NeedTypes | packages.NeedName,
	}
	pkgs, err := packages.Load(cfg, "../annotation")
	if err != nil {
		return nil, fmt.Errorf("failed to load annotation package from %s: %w", thisDir, err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("annotation package not found in %s", thisDir)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("annotation package has errors: %v", pkg.Errors[0])
	}

	// Verify the loaded package path matches the expected path
	expectedPath := modulePath + "/internal/annotation"
	if pkg.Types == nil {
		return nil, fmt.Errorf("annotation package types not loaded")
	}
	if pkg.Types.Path() != expectedPath {
		return nil, fmt.Errorf("loaded package path %q does not match expected %q", pkg.Types.Path(), expectedPath)
	}

	return pkg.Types, nil
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
