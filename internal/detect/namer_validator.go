package detect

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"unicode"

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

	// Extract the expected type name from namerType for receiver matching.
	// Unwrap pointer if present.
	expectedType := namerType
	if ptr, ok := expectedType.(*types.Pointer); ok {
		expectedType = ptr.Elem()
	}
	var expectedTypeName string
	if named, ok := expectedType.(*types.Named); ok {
		expectedTypeName = named.Obj().Name()
	}

	// Search in external package files, verifying receiver type name
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if funcDecl.Name.Name != "Name" || funcDecl.Recv == nil {
				continue
			}

			// Match receiver type name against expected type name
			if expectedTypeName != "" && len(funcDecl.Recv.List) > 0 {
				recvExpr := funcDecl.Recv.List[0].Type
				if star, ok := recvExpr.(*ast.StarExpr); ok {
					recvExpr = star.X
				}
				if ident, ok := recvExpr.(*ast.Ident); ok && ident.Name == expectedTypeName {
					return funcDecl, nil
				}
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

// validateLiteralReturn validates that all return statements in the method return the same string literal.
func (v *namerValidatorImpl) validateLiteralReturn(methodDecl *ast.FuncDecl) (string, error) {
	if methodDecl.Body == nil {
		return "", fmt.Errorf("Name() method has no body")
	}

	// Collect all return statements
	var returnStmts []*ast.ReturnStmt
	ast.Inspect(
		methodDecl.Body, func(n ast.Node) bool {
			if ret, ok := n.(*ast.ReturnStmt); ok {
				returnStmts = append(returnStmts, ret)
			}
			return true
		},
	)

	if len(returnStmts) == 0 {
		return "", fmt.Errorf("Name() method has no return statement")
	}

	if len(returnStmts) > 1 {
		return "", fmt.Errorf("Name() method must have exactly one return statement, found %d", len(returnStmts))
	}

	// Validate the single return statement returns a string literal
	if len(returnStmts[0].Results) != 1 {
		return "", fmt.Errorf("Name() method must return exactly one value")
	}

	// Check if return value is a string literal
	result := returnStmts[0].Results[0]
	basicLit, ok := result.(*ast.BasicLit)
	if !ok {
		return "", fmt.Errorf("Name() must return hardcoded string literal, found %T", result)
	}

	if basicLit.Kind != token.STRING {
		return "", fmt.Errorf("Name() must return hardcoded string literal, found %s", basicLit.Kind)
	}

	// Unquote the string literal (handles double quotes, backticks, and escape sequences)
	unquoted, err := unquoteStringLiteral(basicLit.Value)
	if err != nil {
		return "", fmt.Errorf("Name() has invalid string literal %s: %w", basicLit.Value, err)
	}

	if unquoted == "" {
		return "", fmt.Errorf("Name() must return non-empty string literal")
	}

	// Validate that the name is a valid Go identifier
	if !isValidGoIdentifier(unquoted) {
		return "", fmt.Errorf("Name() must return a valid Go identifier, got %q", unquoted)
	}

	return unquoted, nil
}

// unquoteStringLiteral unquotes a Go string literal (double-quoted or backtick).
func unquoteStringLiteral(lit string) (string, error) {
	if len(lit) >= 2 && lit[0] == '`' && lit[len(lit)-1] == '`' {
		// Raw string literal: strip backticks, no escape processing
		return lit[1 : len(lit)-1], nil
	}
	return strconv.Unquote(lit)
}

// isValidGoIdentifier checks if a string is a valid Go identifier.
func isValidGoIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
		} else {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				return false
			}
		}
	}
	return true
}
