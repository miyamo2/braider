package detect

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AppFuncName is the function name for the App annotation.
const AppFuncName = "App"

// AppAnnotation represents a detected annotation.App call.
type AppAnnotation struct {
	CallExpr *ast.CallExpr // The App(main) call expression
	GenDecl  *ast.GenDecl  // The var declaration containing the call
	MainFunc *ast.Ident    // The main function identifier argument
	Pos      token.Pos     // Position for diagnostics
	File     *ast.File     // The file containing this annotation
}

// AppDetector detects annotation.App calls in packages.
type AppDetector interface {
	// DetectAppAnnotations finds all annotation.App calls in the package.
	// Returns a slice of AppAnnotation for all detected calls.
	DetectAppAnnotations(pass *analysis.Pass) []*AppAnnotation

	// ValidateAppAnnotations validates all detected App annotations.
	// Multiple App annotations are allowed (e.g., in multi-main packages).
	// Each annotation must reference the main function.
	// Returns error if any annotation references a non-main function.
	ValidateAppAnnotations(pass *analysis.Pass, apps []*AppAnnotation) error

	// DeduplicateAppsByFile returns the first App annotation from each file.
	// If multiple App annotations exist in the same file, only the first is returned.
	DeduplicateAppsByFile(apps []*AppAnnotation) []*AppAnnotation
}

// AppValidationErrorType represents types of App validation errors.
type AppValidationErrorType int

const (
	// NonMainReference indicates App references a non-main function.
	NonMainReference AppValidationErrorType = iota
)

// AppValidationError represents validation errors for App annotations.
type AppValidationError struct {
	Type      AppValidationErrorType
	Positions []token.Pos
	FuncName  string // For non-main reference error
}

func (e *AppValidationError) Error() string {
	switch e.Type {
	case NonMainReference:
		return fmt.Sprintf("annotation.App must reference main function, got %s", e.FuncName)
	default:
		return "invalid App annotation"
	}
}

// appDetector is the default implementation of AppDetector.
type appDetector struct {
	annotation.Injectable[inject.Typed[AppDetector]]
}

// DetectAppAnnotations finds all annotation.App calls in the package.
func (d *appDetector) DetectAppAnnotations(pass *analysis.Pass) []*AppAnnotation {
	var apps []*AppAnnotation

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

		insp.Preorder(
			nodeFilter, func(n ast.Node) {
				genDecl := n.(*ast.GenDecl)
				// Find the file containing this declaration
				file := d.findFileForNode(pass, genDecl)
				apps = d.processGenDecl(pass, genDecl, apps, file)
			},
		)
	} else {
		// Fallback: iterate files manually
		for _, file := range pass.Files {
			for _, decl := range file.Decls {
				if genDecl, ok := decl.(*ast.GenDecl); ok {
					apps = d.processGenDecl(pass, genDecl, apps, file)
				}
			}
		}
	}

	return apps
}

// processGenDecl processes a GenDecl node and adds AppAnnotation if found.
func (d *appDetector) processGenDecl(
	pass *analysis.Pass, genDecl *ast.GenDecl, apps []*AppAnnotation, file *ast.File,
) []*AppAnnotation {
	// Only process var declarations
	if genDecl.Tok != token.VAR {
		return apps
	}

	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		// Check for blank identifier assignment: var _ = annotation.App(main)
		if len(valueSpec.Names) != 1 || valueSpec.Names[0].Name != "_" {
			continue
		}

		if len(valueSpec.Values) != 1 {
			continue
		}

		callExpr, ok := valueSpec.Values[0].(*ast.CallExpr)
		if !ok {
			continue
		}

		if !d.isAppCall(pass, callExpr) {
			continue
		}

		app := &AppAnnotation{
			CallExpr: callExpr,
			GenDecl:  genDecl,
			Pos:      callExpr.Pos(),
			File:     file,
		}

		// Extract the argument (should be main function identifier)
		if len(callExpr.Args) == 1 {
			if ident, ok := callExpr.Args[0].(*ast.Ident); ok {
				app.MainFunc = ident
			}
		}

		apps = append(apps, app)
	}

	return apps
}

// findFileForNode finds the file containing the given node.
func (d *appDetector) findFileForNode(pass *analysis.Pass, node ast.Node) *ast.File {
	pos := node.Pos()
	for _, file := range pass.Files {
		if file.Pos() <= pos && pos <= file.End() {
			return file
		}
	}
	return nil
}

// isAppCall checks if the call expression is annotation.App.
func (d *appDetector) isAppCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	// Handle selector expression: annotation.App
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	if sel.Sel.Name != AppFuncName {
		return false
	}

	// Verify the package is the annotation package
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	// Use type checker to verify package path
	obj := pass.TypesInfo.Uses[ident]
	if obj == nil {
		return false
	}

	pkgName, ok := obj.(*types.PkgName)
	if !ok {
		return false
	}

	return pkgName.Imported().Path() == AnnotationPath
}

// ValidateAppAnnotations validates all detected App annotations.
func (d *appDetector) ValidateAppAnnotations(pass *analysis.Pass, apps []*AppAnnotation) error {
	if len(apps) == 0 {
		return nil // No App annotation, skip bootstrap generation
	}

	// Validate each App annotation independently (multiple Apps are now allowed)
	for _, app := range apps {
		// Validate that the argument is the main function
		if app.MainFunc == nil {
			return &AppValidationError{
				Type:      NonMainReference,
				Positions: []token.Pos{app.Pos},
				FuncName:  "<unknown>",
			}
		}

		// Verify the identifier resolves to the main function
		obj := pass.TypesInfo.Uses[app.MainFunc]
		if obj == nil {
			// Check Defs in case it's a forward reference
			obj = pass.TypesInfo.Defs[app.MainFunc]
		}

		if obj == nil {
			return &AppValidationError{
				Type:      NonMainReference,
				Positions: []token.Pos{app.Pos},
				FuncName:  app.MainFunc.Name,
			}
		}

		fn, ok := obj.(*types.Func)
		if !ok {
			return &AppValidationError{
				Type:      NonMainReference,
				Positions: []token.Pos{app.Pos},
				FuncName:  app.MainFunc.Name,
			}
		}

		if fn.Name() != "main" {
			return &AppValidationError{
				Type:      NonMainReference,
				Positions: []token.Pos{app.Pos},
				FuncName:  fn.Name(),
			}
		}
	}

	return nil
}

// DeduplicateAppsByFile returns the first App annotation from each file.
// If multiple App annotations exist in the same file, only the first is returned.
// The order of returned annotations matches the input order.
func (d *appDetector) DeduplicateAppsByFile(apps []*AppAnnotation) []*AppAnnotation {
	if len(apps) <= 1 {
		return apps
	}

	// Track first App per file
	fileToApp := make(map[*ast.File]*AppAnnotation)
	var result []*AppAnnotation

	for _, app := range apps {
		if app.File == nil {
			// Fallback: file not set, include app
			result = append(result, app)
			continue
		}

		if _, exists := fileToApp[app.File]; !exists {
			fileToApp[app.File] = app
			result = append(result, app)
		}
		// else: duplicate in same file, skip
	}

	return result
}
