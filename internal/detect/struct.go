package detect

import (
	"go/ast"
	"go/types"
	"unicode"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// ConstructorCandidate represents a struct requiring constructor generation.
type ConstructorCandidate struct {
	TypeSpec            *ast.TypeSpec   // The struct type specification
	StructType          *ast.StructType // The struct type node
	GenDecl             *ast.GenDecl    // Parent declaration (for positioning)
	InjectField         *ast.Field      // The embedded annotation.Injectable field
	ExistingConstructor *ast.FuncDecl   // Existing constructor to replace (nil if none)
}

// StructDetector identifies structs requiring constructor generation.
type StructDetector interface {
	// DetectCandidates returns all structs with annotation.Injectable embedding.
	DetectCandidates(pass *analysis.Pass) []ConstructorCandidate

	// FindExistingConstructor returns the existing constructor function declaration if found.
	// Returns nil if no existing constructor exists.
	FindExistingConstructor(pass *analysis.Pass, structName string) *ast.FuncDecl
}

// structDetector is the default implementation of StructDetector.
type structDetector struct {
	injectDetector InjectDetector
}

// NewStructDetector creates a new StructDetector instance.
func NewStructDetector(injectDetector InjectDetector) StructDetector {
	return &structDetector{
		injectDetector: injectDetector,
	}
}

// DetectCandidates returns all structs with annotation.Injectable embedding.
func (d *structDetector) DetectCandidates(pass *analysis.Pass) []ConstructorCandidate {
	var candidates []ConstructorCandidate

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
func (d *structDetector) processGenDecl(pass *analysis.Pass, genDecl *ast.GenDecl, candidates []ConstructorCandidate) []ConstructorCandidate {
	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		// Check for annotation.Injectable embedding
		injectField := d.injectDetector.FindInjectField(pass, structType)
		if injectField == nil {
			continue
		}

		// Find existing constructor
		existingCtor := d.FindExistingConstructor(pass, typeSpec.Name.Name)

		candidate := ConstructorCandidate{
			TypeSpec:            typeSpec,
			StructType:          structType,
			GenDecl:             genDecl,
			InjectField:         injectField,
			ExistingConstructor: existingCtor,
		}
		candidates = append(candidates, candidate)
	}

	return candidates
}

// FindExistingConstructor finds an existing New<StructName> function in the package.
// Detection criteria:
// 1. Function name matches "New" + StructName (case-sensitive)
// 2. Function is defined in the same package as the struct
// 3. Function returns *StructName (pointer to the struct type)
func (d *structDetector) FindExistingConstructor(pass *analysis.Pass, structName string) *ast.FuncDecl {
	runes := []rune(structName)
	runes[0] = unicode.ToUpper(runes[0])
	expectedName := "New" + string(runes)

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
func (d *structDetector) returnsPointerToStruct(pass *analysis.Pass, fn *ast.FuncDecl, structName string) bool {
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return false
	}

	// Check the first return type (we only care about single return for constructors)
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
func (d *structDetector) isPointerToStructAST(expr ast.Expr, structName string) bool {
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
