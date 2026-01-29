package detect

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// ProviderCandidate represents a struct with annotation.Provide embedding.
type ProviderCandidate struct {
	TypeSpec            *ast.TypeSpec   // The struct type specification
	StructType          *ast.StructType // The struct type node
	GenDecl             *ast.GenDecl    // Parent declaration (for positioning)
	ProvideField        *ast.Field      // The embedded annotation.Provide field
	ExistingConstructor *ast.FuncDecl   // Existing constructor (nil if none)
	PackagePath         string          // Import path of the package
	Implements          []string        // Interface types this struct implements
}

// ProvideStructDetector identifies structs with annotation.Provide embedding.
type ProvideStructDetector interface {
	// DetectProviders returns all structs with annotation.Provide embedding.
	DetectProviders(pass *analysis.Pass) []ProviderCandidate

	// FindExistingConstructor returns the existing constructor function declaration if found.
	// Returns nil if no existing constructor exists.
	FindExistingConstructor(pass *analysis.Pass, structName string) *ast.FuncDecl

	// DetectImplementedInterfaces returns interface types the struct implements.
	DetectImplementedInterfaces(pass *analysis.Pass, typeSpec *ast.TypeSpec) []string
}

// provideStructDetector is the default implementation of ProvideStructDetector.
type provideStructDetector struct {
	provideDetector ProvideDetector
}

// NewProvideStructDetector creates a new ProvideStructDetector instance.
func NewProvideStructDetector(provideDetector ProvideDetector) ProvideStructDetector {
	return &provideStructDetector{
		provideDetector: provideDetector,
	}
}

// DetectProviders returns all structs with annotation.Provide embedding.
func (d *provideStructDetector) DetectProviders(pass *analysis.Pass) []ProviderCandidate {
	var candidates []ProviderCandidate

	// Use inspector if available, otherwise iterate files manually
	var insp *inspector.Inspector
	if pass.ResultOf != nil {
		if result, ok := pass.ResultOf[inspect.Analyzer]; ok {
			insp = result.(*inspector.Inspector)
		}
	}

	if insp != nil {
		// Use inspector for efficient traversal
		nodeFilter := []ast.Node{
			(*ast.GenDecl)(nil),
		}

		insp.Preorder(nodeFilter, func(n ast.Node) {
			genDecl := n.(*ast.GenDecl)
			candidates = d.processGenDecl(pass, genDecl, candidates)
		})
	} else {
		// Fallback: iterate files manually
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

// processGenDecl processes a GenDecl node and adds candidates if found.
func (d *provideStructDetector) processGenDecl(pass *analysis.Pass, genDecl *ast.GenDecl, candidates []ProviderCandidate) []ProviderCandidate {
	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		// Check for annotation.Provide embedding
		provideField := d.provideDetector.FindProvideField(pass, structType)
		if provideField == nil {
			continue
		}

		// Find existing constructor
		existingCtor := d.FindExistingConstructor(pass, typeSpec.Name.Name)

		// Detect implemented interfaces
		implementedIfaces := d.DetectImplementedInterfaces(pass, typeSpec)

		candidate := ProviderCandidate{
			TypeSpec:            typeSpec,
			StructType:          structType,
			GenDecl:             genDecl,
			ProvideField:        provideField,
			ExistingConstructor: existingCtor,
			PackagePath:         pass.Pkg.Path(),
			Implements:          implementedIfaces,
		}
		candidates = append(candidates, candidate)
	}

	return candidates
}

// FindExistingConstructor finds an existing New<StructName> function in the package.
func (d *provideStructDetector) FindExistingConstructor(pass *analysis.Pass, structName string) *ast.FuncDecl {
	expectedName := "New" + structName

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			// Check function name
			if fn.Name.Name != expectedName {
				continue
			}

			// Must be a function, not a method
			if fn.Recv != nil {
				continue
			}

			// Check return type
			if !d.returnsPointerToStruct(pass, fn, structName) {
				continue
			}

			return fn
		}
	}

	return nil
}

// returnsPointerToStruct checks if the function returns a pointer to the specified struct.
func (d *provideStructDetector) returnsPointerToStruct(pass *analysis.Pass, fn *ast.FuncDecl, structName string) bool {
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return false
	}

	// Check the first return type
	result := fn.Type.Results.List[0]

	// Fallback to AST if TypesInfo is not available
	if pass.TypesInfo == nil {
		return d.isPointerToStructAST(result.Type, structName)
	}

	// Get the type from the type checker
	t := pass.TypesInfo.TypeOf(result.Type)
	if t == nil {
		// Fallback: parse AST directly
		return d.isPointerToStructAST(result.Type, structName)
	}

	// Check if it's a pointer to the struct
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}

	// Get the element type
	elem := ptr.Elem()
	named, ok := elem.(*types.Named)
	if !ok {
		return false
	}

	return named.Obj().Name() == structName
}

// isPointerToStructAST checks if the expression is a pointer to the struct using AST.
func (d *provideStructDetector) isPointerToStructAST(expr ast.Expr, structName string) bool {
	starExpr, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}

	ident, ok := starExpr.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == structName
}

// DetectImplementedInterfaces returns interface types the struct implements.
func (d *provideStructDetector) DetectImplementedInterfaces(pass *analysis.Pass, typeSpec *ast.TypeSpec) []string {
	var interfaces []string

	// Return empty if TypesInfo is not available
	if pass.TypesInfo == nil {
		return interfaces
	}

	// Get the type object from definitions
	obj := pass.TypesInfo.Defs[typeSpec.Name]
	if obj == nil {
		return interfaces
	}

	namedType, ok := obj.Type().(*types.Named)
	if !ok {
		return interfaces
	}

	// Get pointer type for method receiver check
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

			// Check if struct implements interface (pointer or value receiver)
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
