package detect

import (
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// findImplementedInterfaces returns interface types the given named type implements.
// It checks both imported packages and the current package scope.
func findImplementedInterfaces(pass *analysis.Pass, namedType *types.Named) []string {
	var interfaces []string

	if pass.TypesInfo == nil {
		return interfaces
	}

	ptrType := types.NewPointer(namedType)

	// Iterate through all imported packages and check interfaces
	for _, pkg := range pass.Pkg.Imports() {
		scope := pkg.Scope()
		for _, name := range scope.Names() {
			scopeObj := scope.Lookup(name)
			if scopeObj == nil {
				continue
			}

			if _, ok := scopeObj.(*types.TypeName); !ok {
				continue
			}

			named, ok := scopeObj.Type().(*types.Named)
			if !ok {
				continue
			}

			iface, ok := named.Underlying().(*types.Interface)
			if !ok {
				continue
			}

			if types.Implements(ptrType, iface) || types.Implements(namedType, iface) {
				interfaces = append(interfaces, pkg.Path()+"."+name)
			}
		}
	}

	// Also check interfaces in current package
	scope := pass.Pkg.Scope()
	for _, name := range scope.Names() {
		scopeObj := scope.Lookup(name)
		if scopeObj == nil {
			continue
		}

		if _, ok := scopeObj.(*types.TypeName); !ok {
			continue
		}

		named, ok := scopeObj.Type().(*types.Named)
		if !ok {
			continue
		}

		iface, ok := named.Underlying().(*types.Interface)
		if !ok {
			continue
		}

		if types.Implements(ptrType, iface) || types.Implements(namedType, iface) {
			interfaces = append(interfaces, pass.Pkg.Path()+"."+name)
		}
	}

	return interfaces
}

// findImplementedInterfacesFromType dereferences pointer types and delegates
// to findImplementedInterfaces.
func findImplementedInterfacesFromType(pass *analysis.Pass, t types.Type) []string {
	baseType := t
	if ptr, ok := baseType.(*types.Pointer); ok {
		baseType = ptr.Elem()
	}

	namedType, ok := baseType.(*types.Named)
	if !ok {
		return nil
	}

	return findImplementedInterfaces(pass, namedType)
}
