// Package report provides diagnostic reporting capabilities for braider analyzer.
package report

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"golang.org/x/tools/go/analysis"
)

// SuggestedFixBuilder constructs SuggestedFix for diagnostics.
type SuggestedFixBuilder interface {
	// BuildConstructorFix creates a SuggestedFix for constructor insertion or replacement.
	// If candidate.ExistingConstructor is non-nil, builds a replacement TextEdit.
	// Otherwise, builds an insertion TextEdit after the struct definition.
	BuildConstructorFix(
		pass *analysis.Pass,
		candidate detect.ConstructorCandidate,
		constructor *generate.GeneratedConstructor,
	) analysis.SuggestedFix

	// BuildBootstrapFix creates a SuggestedFix for inserting bootstrap code.
	// Inserts dependency variable and optionally adds reference in main.
	BuildBootstrapFix(
		pass *analysis.Pass,
		app *detect.AppAnnotation,
		bootstrap *generate.GeneratedBootstrap,
		mainFunc *ast.FuncDecl,
	) analysis.SuggestedFix

	// BuildBootstrapReplacementFix creates a SuggestedFix for replacing existing bootstrap code.
	// Replaces existing dependency variable and updates main reference if needed.
	BuildBootstrapReplacementFix(
		pass *analysis.Pass,
		existing *ast.GenDecl,
		bootstrap *generate.GeneratedBootstrap,
		mainFunc *ast.FuncDecl,
	) analysis.SuggestedFix
}

// suggestedFixBuilder is the default implementation of SuggestedFixBuilder.
type suggestedFixBuilder struct{}

// NewSuggestedFixBuilder creates a new SuggestedFixBuilder instance.
func NewSuggestedFixBuilder() SuggestedFixBuilder {
	return &suggestedFixBuilder{}
}

// BuildConstructorFix creates a SuggestedFix for constructor insertion or replacement.
func (b *suggestedFixBuilder) BuildConstructorFix(
	pass *analysis.Pass,
	candidate detect.ConstructorCandidate,
	constructor *generate.GeneratedConstructor,
) analysis.SuggestedFix {
	if candidate.ExistingConstructor != nil {
		return b.buildReplacementFix(pass, candidate, constructor)
	}
	return b.buildInsertionFix(pass, candidate, constructor)
}

// buildInsertionFix creates a SuggestedFix for inserting a new constructor.
func (b *suggestedFixBuilder) buildInsertionFix(
	pass *analysis.Pass,
	candidate detect.ConstructorCandidate,
	constructor *generate.GeneratedConstructor,
) analysis.SuggestedFix {
	// Calculate insertion position after struct definition
	insertPos := candidate.GenDecl.End()

	// Prepend blank line separator
	newText := "\n\n" + constructor.Code

	return analysis.SuggestedFix{
		Message: fmt.Sprintf("generate constructor for %s", constructor.StructName),
		TextEdits: []analysis.TextEdit{
			{
				Pos:     insertPos,
				End:     insertPos, // Same as Pos for insertion
				NewText: []byte(newText),
			},
		},
	}
}

// buildReplacementFix creates a SuggestedFix for replacing an existing constructor.
func (b *suggestedFixBuilder) buildReplacementFix(
	pass *analysis.Pass,
	candidate detect.ConstructorCandidate,
	constructor *generate.GeneratedConstructor,
) analysis.SuggestedFix {
	fn := candidate.ExistingConstructor

	// Calculate replacement range
	start, end := b.calculateReplacementRange(fn)

	return analysis.SuggestedFix{
		Message: fmt.Sprintf("regenerate constructor for %s", constructor.StructName),
		TextEdits: []analysis.TextEdit{
			{
				Pos:     start,
				End:     end,
				NewText: []byte(constructor.Code),
			},
		},
	}
}

// calculateReplacementRange returns the byte range for replacing an existing constructor.
func (b *suggestedFixBuilder) calculateReplacementRange(fn *ast.FuncDecl) (start, end token.Pos) {
	// Include doc comment if present
	if fn.Doc != nil {
		start = fn.Doc.Pos()
	} else {
		start = fn.Pos()
	}
	end = fn.End()
	return start, end
}

// BuildBootstrapFix creates a SuggestedFix for inserting bootstrap code.
func (b *suggestedFixBuilder) BuildBootstrapFix(
	pass *analysis.Pass,
	app *detect.AppAnnotation,
	bootstrap *generate.GeneratedBootstrap,
	mainFunc *ast.FuncDecl,
) analysis.SuggestedFix {
	var edits []analysis.TextEdit

	// Find insertion point for dependency variable
	// Insert after package declaration, before first function
	insertPos := b.findBootstrapInsertionPoint(pass)

	// Add dependency variable
	dependencyText := "\n\n" + bootstrap.DependencyVar + "\n"
	edits = append(edits, analysis.TextEdit{
		Pos:     insertPos,
		End:     insertPos,
		NewText: []byte(dependencyText),
	})

	// Add main reference if dependency is not referenced and _ = dependency doesn't already exist
	if mainFunc != nil && !generate.IsDependencyReferenced(mainFunc) && !generate.HasBlankDependencyAssignment(mainFunc) {
		mainRefPos := b.findMainReferenceInsertionPoint(mainFunc)
		mainRefText := "\t" + bootstrap.MainReference + "\n"
		edits = append(edits, analysis.TextEdit{
			Pos:     mainRefPos,
			End:     mainRefPos,
			NewText: []byte(mainRefText),
		})
	}

	return analysis.SuggestedFix{
		Message:   "generate bootstrap code",
		TextEdits: edits,
	}
}

// BuildBootstrapReplacementFix creates a SuggestedFix for replacing existing bootstrap code.
func (b *suggestedFixBuilder) BuildBootstrapReplacementFix(
	pass *analysis.Pass,
	existing *ast.GenDecl,
	bootstrap *generate.GeneratedBootstrap,
	mainFunc *ast.FuncDecl,
) analysis.SuggestedFix {
	var edits []analysis.TextEdit

	// Replace existing dependency variable
	start, end := b.calculateBootstrapReplacementRange(existing)
	edits = append(edits, analysis.TextEdit{
		Pos:     start,
		End:     end,
		NewText: []byte(bootstrap.DependencyVar),
	})

	// Update main reference if needed (only if not referenced and _ = dependency doesn't exist)
	if mainFunc != nil && !generate.IsDependencyReferenced(mainFunc) && !generate.HasBlankDependencyAssignment(mainFunc) {
		mainRefPos := b.findMainReferenceInsertionPoint(mainFunc)
		mainRefText := "\t" + bootstrap.MainReference + "\n"
		edits = append(edits, analysis.TextEdit{
			Pos:     mainRefPos,
			End:     mainRefPos,
			NewText: []byte(mainRefText),
		})
	}

	return analysis.SuggestedFix{
		Message:   "update bootstrap code",
		TextEdits: edits,
	}
}

// findBootstrapInsertionPoint finds the position to insert bootstrap code.
// Returns position after package declaration, before first function.
func (b *suggestedFixBuilder) findBootstrapInsertionPoint(pass *analysis.Pass) token.Pos {
	for _, file := range pass.Files {
		// Find first function declaration
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				return fn.Pos()
			}
		}
		// If no function found, insert at end of file
		return file.End()
	}
	return token.NoPos
}

// findMainReferenceInsertionPoint finds the position to insert dependency reference in main.
// Returns position at start of main function body.
func (b *suggestedFixBuilder) findMainReferenceInsertionPoint(mainFunc *ast.FuncDecl) token.Pos {
	if mainFunc.Body != nil && len(mainFunc.Body.List) > 0 {
		// Insert before first statement
		return mainFunc.Body.List[0].Pos()
	}
	// Empty body: insert after opening brace
	return mainFunc.Body.Lbrace + 1
}

// calculateBootstrapReplacementRange returns the byte range for replacing existing bootstrap.
func (b *suggestedFixBuilder) calculateBootstrapReplacementRange(genDecl *ast.GenDecl) (start, end token.Pos) {
	// Include doc comment if present
	if genDecl.Doc != nil {
		start = genDecl.Doc.Pos()
	} else {
		start = genDecl.Pos()
	}
	end = genDecl.End()
	return start, end
}

// findExistingBlankDependencyAssignment finds an existing "_ = dependency" statement in main function.
// Returns the assignment statement node if found, nil otherwise.
//
// TODO: This method is reserved for future use in BuildBootstrapReplacementFix
// to properly handle updating bootstrap code when "_ = dependency" already exists.
func (b *suggestedFixBuilder) findExistingBlankDependencyAssignment(mainFunc *ast.FuncDecl) *ast.AssignStmt {
	if mainFunc == nil || mainFunc.Body == nil {
		return nil
	}

	var found *ast.AssignStmt
	ast.Inspect(mainFunc.Body, func(n ast.Node) bool {
		if found != nil {
			return false // Already found, stop searching
		}

		assignStmt, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		// Check for "_ = dependency" pattern
		if len(assignStmt.Lhs) == 1 && len(assignStmt.Rhs) == 1 {
			if lhsIdent, ok := assignStmt.Lhs[0].(*ast.Ident); ok {
				if lhsIdent.Name == "_" {
					if rhsIdent, ok := assignStmt.Rhs[0].(*ast.Ident); ok {
						if rhsIdent.Name == "dependency" {
							found = assignStmt
							return false
						}
					}
				}
			}
		}

		return true
	})

	return found
}
