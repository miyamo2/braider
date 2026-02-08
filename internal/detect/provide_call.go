package detect

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// ProviderCandidate represents a detected provider from annotation.Provide[T](fn) call.
type ProviderCandidate struct {
	// CallExpr is the annotation.Provide[T](fn) call expression
	CallExpr *ast.CallExpr
	// ProviderFunc is the fn argument expression
	ProviderFunc ast.Expr
	// ProviderFuncSig is fn's type signature
	ProviderFuncSig *types.Signature
	// ProviderFuncName is the function name (e.g., "NewUserRepository")
	ProviderFuncName string
	// ReturnType is the first return type of the provider function
	ReturnType types.Type
	// ReturnTypeName is the local name of the return type (e.g., "userRepository")
	ReturnTypeName string
	// PackagePath is the import path of the package
	PackagePath string
	// Implements contains interface types the return type implements
	Implements []string
}

// ProvideCallDetector identifies annotation.Provide[T](fn) call expressions.
type ProvideCallDetector interface {
	// DetectProviders returns all annotation.Provide[T](fn) calls in the package.
	DetectProviders(pass *analysis.Pass) []ProviderCandidate

	// DetectImplementedInterfaces returns interface types the given type implements.
	DetectImplementedInterfaces(pass *analysis.Pass, namedType *types.Named) []string
}

// provideCallDetector is the default implementation of ProvideCallDetector.
type provideCallDetector struct{}

// NewProvideCallDetector creates a new ProvideCallDetector instance.
func NewProvideCallDetector() ProvideCallDetector {
	return &provideCallDetector{}
}

// DetectProviders returns all annotation.Provide[T](fn) calls in the package.
func (d *provideCallDetector) DetectProviders(pass *analysis.Pass) []ProviderCandidate {
	var candidates []ProviderCandidate

	// Use inspector if available, otherwise iterate files manually
	var insp *inspector.Inspector
	if pass.ResultOf != nil {
		if result, ok := pass.ResultOf[inspect.Analyzer]; ok {
			insp = result.(*inspector.Inspector)
		}
	}

	if insp != nil {
		nodeFilter := []ast.Node{
			(*ast.GenDecl)(nil),
		}

		insp.Preorder(nodeFilter, func(n ast.Node) {
			genDecl := n.(*ast.GenDecl)
			candidates = d.processGenDecl(pass, genDecl, candidates)
		})
	} else {
		for _, file := range pass.Files {
			for _, decl := range file.Decls {
				if genDecl, ok := decl.(*ast.GenDecl); ok {
					candidates = d.processGenDecl(pass, genDecl, candidates)
				}
			}
		}
	}

	return candidates
}

// processGenDecl processes a GenDecl and looks for var _ = annotation.Provide[T](fn) patterns.
func (d *provideCallDetector) processGenDecl(pass *analysis.Pass, genDecl *ast.GenDecl, candidates []ProviderCandidate) []ProviderCandidate {
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for _, value := range valueSpec.Values {
			callExpr, ok := value.(*ast.CallExpr)
			if !ok {
				continue
			}

			if !d.isProvideCall(pass, callExpr) {
				continue
			}

			candidate := d.extractCandidate(pass, callExpr)
			if candidate != nil {
				candidates = append(candidates, *candidate)
			}
		}
	}
	return candidates
}

// isProvideCall checks if a call expression is annotation.Provide[T](fn).
func (d *provideCallDetector) isProvideCall(pass *analysis.Pass, callExpr *ast.CallExpr) bool {
	// The function part is annotation.Provide[T] which is an IndexExpr wrapping a SelectorExpr
	// Pattern: *ast.IndexExpr { X: *ast.SelectorExpr { Sel: "Provide" } }
	var selExpr *ast.SelectorExpr

	switch fun := callExpr.Fun.(type) {
	case *ast.IndexExpr:
		// annotation.Provide[T](fn) - single type parameter
		sel, ok := fun.X.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		selExpr = sel
	case *ast.IndexListExpr:
		// annotation.Provide[T1, T2, ...](fn) - multiple type parameters (future-proof)
		sel, ok := fun.X.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		selExpr = sel
	case *ast.SelectorExpr:
		// annotation.Provide(fn) - no type parameter (non-generic, shouldn't happen but handle)
		selExpr = fun
	default:
		return false
	}

	// Check selector name is "Provide"
	if selExpr.Sel.Name != ProvideTypeName {
		return false
	}

	// Check the call's return type is from the annotation package
	typ := pass.TypesInfo.TypeOf(callExpr)
	if typ == nil {
		return false
	}

	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil {
		return false
	}

	pkg := obj.Pkg()
	if pkg == nil {
		return false
	}

	return pkg.Path() == AnnotationPath
}

// extractCandidate extracts a ProviderCandidate from a validated Provide[T](fn) call.
func (d *provideCallDetector) extractCandidate(pass *analysis.Pass, callExpr *ast.CallExpr) *ProviderCandidate {
	if len(callExpr.Args) == 0 {
		return nil
	}

	providerFuncExpr := callExpr.Args[0]

	// Get the type of the provider function argument
	providerFuncType := pass.TypesInfo.TypeOf(providerFuncExpr)
	if providerFuncType == nil {
		return nil
	}

	sig, ok := providerFuncType.(*types.Signature)
	if !ok {
		return nil
	}

	// Extract function name
	funcName := d.extractFuncName(providerFuncExpr)

	// Extract return type
	var returnType types.Type
	var returnTypeName string
	if sig.Results() != nil && sig.Results().Len() > 0 {
		returnType = sig.Results().At(0).Type()
		returnTypeName = d.extractReturnTypeName(returnType)
	}

	// Detect implemented interfaces
	var implements []string
	if returnType != nil {
		implements = d.detectImplementedInterfacesFromType(pass, returnType)
	}

	return &ProviderCandidate{
		CallExpr:         callExpr,
		ProviderFunc:     providerFuncExpr,
		ProviderFuncSig:  sig,
		ProviderFuncName: funcName,
		ReturnType:       returnType,
		ReturnTypeName:   returnTypeName,
		PackagePath:      pass.Pkg.Path(),
		Implements:       implements,
	}
}

// extractFuncName extracts the function name from a function expression.
func (d *provideCallDetector) extractFuncName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	default:
		return ""
	}
}

// extractReturnTypeName extracts the local type name from a return type.
func (d *provideCallDetector) extractReturnTypeName(t types.Type) string {
	// Dereference pointer type
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name()
	}

	return ""
}

// detectImplementedInterfacesFromType detects interfaces implemented by the given type.
func (d *provideCallDetector) detectImplementedInterfacesFromType(pass *analysis.Pass, returnType types.Type) []string {
	// Dereference pointer to get the underlying named type
	baseType := returnType
	if ptr, ok := baseType.(*types.Pointer); ok {
		baseType = ptr.Elem()
	}

	namedType, ok := baseType.(*types.Named)
	if !ok {
		return nil
	}

	return d.DetectImplementedInterfaces(pass, namedType)
}

// DetectImplementedInterfaces returns interface types the given named type implements.
func (d *provideCallDetector) DetectImplementedInterfaces(pass *analysis.Pass, namedType *types.Named) []string {
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
