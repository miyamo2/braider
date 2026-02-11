package detect

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// VariableTypeName is the type name for the Variable annotation.
const VariableTypeName = "Variable"

// VariableCandidate represents a detected Variable annotation call.
type VariableCandidate struct {
	// CallExpr is the annotation.Variable[T](value) call expression
	CallExpr *ast.CallExpr
	// ArgumentExpr is the value argument expression (e.g., os.Stdout)
	ArgumentExpr ast.Expr
	// ArgumentType is the resolved type of the argument expression
	ArgumentType types.Type
	// TypeName is the fully qualified type name (e.g., "os.File")
	TypeName string
	// PackagePath is the import path of the argument type's package
	PackagePath string
	// ExpressionText is the formatted source text of the argument expression
	ExpressionText string
	// ExpressionPkgs contains package paths referenced by the expression (path -> name)
	ExpressionPkgs map[string]string
	// IsQualified indicates whether the expression is already package-qualified (SelectorExpr)
	IsQualified bool
	// Implements contains interface types the argument type implements
	Implements []string
}

// VariableCallDetector identifies annotation.Variable[T](value) call expressions.
type VariableCallDetector interface {
	// DetectVariables returns all annotation.Variable[T](value) calls in the package.
	DetectVariables(pass *analysis.Pass) []VariableCandidate
}

// variableCallDetector is the default implementation of VariableCallDetector.
type variableCallDetector struct{}

// NewVariableCallDetector creates a new VariableCallDetector instance.
func NewVariableCallDetector() VariableCallDetector {
	return &variableCallDetector{}
}

// DetectVariables returns all annotation.Variable[T](value) calls in the package.
func (d *variableCallDetector) DetectVariables(pass *analysis.Pass) []VariableCandidate {
	var candidates []VariableCandidate

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

		insp.Preorder(
			nodeFilter, func(n ast.Node) {
				genDecl := n.(*ast.GenDecl)
				candidates = d.processGenDecl(pass, genDecl, candidates)
			},
		)
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

// processGenDecl processes a GenDecl and looks for var _ = annotation.Variable[T](value) patterns.
func (d *variableCallDetector) processGenDecl(
	pass *analysis.Pass, genDecl *ast.GenDecl, candidates []VariableCandidate,
) []VariableCandidate {
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

			if !d.isVariableCall(pass, callExpr) {
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

// isVariableCall checks if a call expression is annotation.Variable[T](value).
func (d *variableCallDetector) isVariableCall(pass *analysis.Pass, callExpr *ast.CallExpr) bool {
	// The function part is annotation.Variable[T] which is an IndexExpr wrapping a SelectorExpr
	// Pattern: *ast.IndexExpr { X: *ast.SelectorExpr { Sel: "Variable" } }
	var selExpr *ast.SelectorExpr

	switch fun := callExpr.Fun.(type) {
	case *ast.IndexExpr:
		// annotation.Variable[T](value) - single type parameter
		sel, ok := fun.X.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		selExpr = sel
	case *ast.IndexListExpr:
		// annotation.Variable[T1, T2, ...](value) - multiple type parameters (future-proof)
		sel, ok := fun.X.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		selExpr = sel
	case *ast.SelectorExpr:
		// annotation.Variable(value) - no type parameter (non-generic)
		selExpr = fun
	default:
		return false
	}

	// Check selector name is "Variable"
	if selExpr.Sel.Name != VariableTypeName {
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

// extractCandidate extracts a VariableCandidate from a validated Variable[T](value) call.
func (d *variableCallDetector) extractCandidate(pass *analysis.Pass, callExpr *ast.CallExpr) *VariableCandidate {
	if len(callExpr.Args) == 0 {
		return nil
	}

	argExpr := callExpr.Args[0]

	// Get the type of the argument expression
	argType := pass.TypesInfo.TypeOf(argExpr)
	if argType == nil {
		return nil
	}

	// Extract the fully qualified type name
	typeName := d.extractTypeName(argType)

	// Extract the package path from the argument type
	packagePath := d.extractPackagePath(pass, argType)

	// Format the expression text using go/format.Node()
	expressionText := d.formatExpression(pass, argExpr)

	// Determine if expression is already package-qualified
	isQualified := d.isQualifiedExpr(argExpr)

	// Collect package paths referenced by the expression
	expressionPkgs := d.collectExpressionPkgs(pass, argExpr)

	// Detect implemented interfaces
	var implements []string
	implements = d.detectImplementedInterfacesFromType(pass, argType)

	return &VariableCandidate{
		CallExpr:       callExpr,
		ArgumentExpr:   argExpr,
		ArgumentType:   argType,
		TypeName:       typeName,
		PackagePath:    packagePath,
		ExpressionText: expressionText,
		ExpressionPkgs: expressionPkgs,
		IsQualified:    isQualified,
		Implements:     implements,
	}
}

// extractTypeName extracts the fully qualified type name from a type.
func (d *variableCallDetector) extractTypeName(t types.Type) string {
	// Dereference pointer type
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	if named, ok := t.(*types.Named); ok {
		obj := named.Obj()
		if pkg := obj.Pkg(); pkg != nil {
			return pkg.Path() + "." + obj.Name()
		}
		return obj.Name()
	}

	return t.String()
}

// extractPackagePath extracts the package path from a type.
func (d *variableCallDetector) extractPackagePath(pass *analysis.Pass, t types.Type) string {
	// Dereference pointer type
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	if named, ok := t.(*types.Named); ok {
		if pkg := named.Obj().Pkg(); pkg != nil {
			return pkg.Path()
		}
	}

	return pass.Pkg.Path()
}

// formatExpression formats an AST expression to canonical Go source text.
func (d *variableCallDetector) formatExpression(pass *analysis.Pass, expr ast.Expr) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, pass.Fset, expr); err != nil {
		return ""
	}
	return buf.String()
}

// isQualifiedExpr checks if the expression is already package-qualified (SelectorExpr at top level).
func (d *variableCallDetector) isQualifiedExpr(expr ast.Expr) bool {
	_, ok := expr.(*ast.SelectorExpr)
	return ok
}

// collectExpressionPkgs collects all package paths referenced by the expression.
// Returns a map of package path to package name.
func (d *variableCallDetector) collectExpressionPkgs(pass *analysis.Pass, expr ast.Expr) map[string]string {
	pkgs := make(map[string]string)

	ast.Inspect(expr, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		// Check if this identifier refers to a package-level object
		obj, exists := pass.TypesInfo.Uses[ident]
		if !exists {
			return true
		}

		// Check if the object is a package name (used as qualifier in selector expressions)
		if pkgName, ok := obj.(*types.PkgName); ok {
			pkg := pkgName.Imported()
			pkgs[pkg.Path()] = pkg.Name()
		}

		return true
	})

	return pkgs
}

// detectImplementedInterfacesFromType detects interfaces implemented by the given type.
func (d *variableCallDetector) detectImplementedInterfacesFromType(pass *analysis.Pass, argType types.Type) []string {
	// Dereference pointer to get the underlying named type
	baseType := argType
	if ptr, ok := baseType.(*types.Pointer); ok {
		baseType = ptr.Elem()
	}

	namedType, ok := baseType.(*types.Named)
	if !ok {
		return nil
	}

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
