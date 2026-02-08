package detect

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/miyamo2/braider/internal/loader"
	"golang.org/x/tools/go/analysis"
)

// NamerValidator validates Namer interface implementations.
// Ensures that Namer.Name() methods return hardcoded string literals.
type NamerValidator interface {
	// ExtractName validates that the given Namer type's Name() method returns
	// a hardcoded string literal and returns the extracted name.
	// Returns error if Name() method not found, returns non-literal, or AST traversal fails.
	ExtractName(pass *analysis.Pass, namerType types.Type) (string, error)
}

// namerValidatorImpl implements NamerValidator.
type namerValidatorImpl struct {
	loader loader.PackageLoader
}

// NewNamerValidator creates a new NamerValidator instance.
// loader can be nil for same-package validation only.
func NewNamerValidator(loader loader.PackageLoader) NamerValidator {
	return &namerValidatorImpl{
		loader: loader,
	}
}

// ExtractName validates and extracts the name from a Namer type.
func (v *namerValidatorImpl) ExtractName(pass *analysis.Pass, namerType types.Type) (string, error) {
	// Find Name() method
	method := findNameMethod(namerType)
	if method == nil {
		return "", fmt.Errorf("type %s does not have Name() string method", namerType)
	}

	// Validate method signature
	sig, ok := method.Type().(*types.Signature)
	if !ok {
		return "", fmt.Errorf("Name method has invalid type")
	}

	// Check return type is string
	if sig.Results() == nil || sig.Results().Len() != 1 {
		return "", fmt.Errorf("Name() method must return exactly one value")
	}

	retType := sig.Results().At(0).Type()
	if !types.Identical(retType, types.Typ[types.String]) {
		return "", fmt.Errorf("Name() method must return string, got %s", retType)
	}

	// Find method declaration in AST
	methodDecl, err := v.findMethodDecl(pass, method, namerType)
	if err != nil {
		return "", err
	}

	// Validate return statement contains literal
	name, err := v.validateLiteralReturn(methodDecl)
	if err != nil {
		return "", err
	}

	return name, nil
}

// findNameMethod finds the Name() method in the type's method set.
func findNameMethod(typ types.Type) *types.Func {
	// Get method set for the type
	ms := types.NewMethodSet(typ)
	for i := 0; i < ms.Len(); i++ {
		obj := ms.At(i).Obj()
		if fn, ok := obj.(*types.Func); ok && fn.Name() == "Name" {
			return fn
		}
	}
	return nil
}

// findMethodDecl finds the AST declaration for a method.
func (v *namerValidatorImpl) findMethodDecl(
	pass *analysis.Pass, method *types.Func, namerType types.Type,
) (*ast.FuncDecl, error) {
	// Check if method is in the current package
	if method.Pkg() == pass.Pkg {
		// Search in current package files
		for _, file := range pass.Files {
			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}

				// Check if this is the Name method
				if funcDecl.Name.Name == "Name" && funcDecl.Recv != nil {
					// Verify receiver type matches
					if v.matchesReceiverType(pass, funcDecl.Recv, namerType) {
						return funcDecl, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("Name() method declaration not found in current package")
	}

	// Method is in external package - use loader
	if v.loader == nil {
		return nil, fmt.Errorf(
			"cannot validate external Namer in package %s: no package loader available. Define Namer in same package as Injectable annotation",
			method.Pkg().Path(),
		)
	}

	pkg, err := v.loader.LoadPackage(method.Pkg().Path())
	if err != nil {
		return nil, fmt.Errorf(
			"cannot validate external Namer in package %s: %w. Define Namer in same package as Injectable annotation",
			method.Pkg().Path(),
			err,
		)
	}

	// Search in external package files
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if funcDecl.Name.Name == "Name" && funcDecl.Recv != nil {
				// For external packages, we can't easily verify the exact receiver type
				// Just check it's a method named Name
				return funcDecl, nil
			}
		}
	}

	return nil, fmt.Errorf("Name() method declaration not found in package %s", method.Pkg().Path())
}

// matchesReceiverType checks if a receiver matches the namer type.
func (v *namerValidatorImpl) matchesReceiverType(pass *analysis.Pass, recv *ast.FieldList, namerType types.Type) bool {
	if recv == nil || len(recv.List) == 0 {
		return false
	}

	recvField := recv.List[0]
	recvTypeExpr := recvField.Type

	// Handle pointer receiver
	if starExpr, ok := recvTypeExpr.(*ast.StarExpr); ok {
		recvTypeExpr = starExpr.X
	}

	// Get receiver type from type checker
	recvType := pass.TypesInfo.TypeOf(recvTypeExpr)
	if recvType == nil {
		return false
	}

	// Compare with namer type
	return types.Identical(recvType, namerType)
}

// validateLiteralReturn validates that the method returns a string literal.
func (v *namerValidatorImpl) validateLiteralReturn(methodDecl *ast.FuncDecl) (string, error) {
	if methodDecl.Body == nil {
		return "", fmt.Errorf("Name() method has no body")
	}

	// Find return statement
	var returnStmt *ast.ReturnStmt
	ast.Inspect(
		methodDecl.Body, func(n ast.Node) bool {
			if ret, ok := n.(*ast.ReturnStmt); ok {
				returnStmt = ret
				return false // Stop searching after first return
			}
			return true
		},
	)

	if returnStmt == nil {
		return "", fmt.Errorf("Name() method has no return statement")
	}

	if len(returnStmt.Results) != 1 {
		return "", fmt.Errorf("Name() method must return exactly one value")
	}

	// Check if return value is a string literal
	result := returnStmt.Results[0]
	basicLit, ok := result.(*ast.BasicLit)
	if !ok {
		return "", fmt.Errorf("Name() must return hardcoded string literal, found %T", result)
	}

	if basicLit.Kind != token.STRING {
		return "", fmt.Errorf("Name() must return hardcoded string literal, found %s", basicLit.Kind)
	}

	// Strip quotes from string literal
	name := basicLit.Value
	if len(name) >= 2 && name[0] == '"' && name[len(name)-1] == '"' {
		name = name[1 : len(name)-1]
	}

	if name == "" {
		return "", fmt.Errorf("Name() must return non-empty string literal")
	}

	return name, nil
}
