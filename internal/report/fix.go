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
