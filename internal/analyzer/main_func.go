package analyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// findMainFunction returns the top-level "func main" declaration in pass, or nil
// if none exists. Used by both DependencyAnalyzeRunner (to classify a package
// as a main package) and AppAnalyzeRunner (to position inferred-entry-point
// diagnostics and synthesize an inferred AppAnnotation).
func findMainFunction(pass *analysis.Pass) *ast.FuncDecl {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if fn.Recv == nil && fn.Name != nil && fn.Name.Name == "main" {
					return fn
				}
			}
		}
	}
	return nil
}

// findFileForFunc returns the *ast.File in pass containing fn, or nil if none does.
func findFileForFunc(pass *analysis.Pass, fn *ast.FuncDecl) *ast.File {
	if fn == nil {
		return nil
	}
	pos := fn.Pos()
	for _, file := range pass.Files {
		if file.Pos() <= pos && pos <= file.End() {
			return file
		}
	}
	return nil
}
