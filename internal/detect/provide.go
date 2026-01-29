// Package detect provides detection capabilities for braider analyzer.
package detect

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// ProvideTypeName is the type name for the Provide annotation.
const ProvideTypeName = "Provide"

// ProvideDetector detects annotation.Provide embedding in structs.
type ProvideDetector interface {
	// HasProvideAnnotation checks if a struct embeds annotation.Provide.
	// Returns true if the struct has an embedded annotation.Provide field.
	HasProvideAnnotation(pass *analysis.Pass, st *ast.StructType) bool

	// FindProvideField returns the embedded Provide field if present.
	// Returns nil if no Provide embedding is found.
	FindProvideField(pass *analysis.Pass, st *ast.StructType) *ast.Field
}

// provideDetector is the default implementation of ProvideDetector.
type provideDetector struct{}

// NewProvideDetector creates a new ProvideDetector instance.
func NewProvideDetector() ProvideDetector {
	return &provideDetector{}
}

// HasProvideAnnotation checks if a struct embeds annotation.Provide.
func (d *provideDetector) HasProvideAnnotation(pass *analysis.Pass, st *ast.StructType) bool {
	return d.FindProvideField(pass, st) != nil
}

// FindProvideField returns the embedded Provide field if present.
func (d *provideDetector) FindProvideField(pass *analysis.Pass, st *ast.StructType) *ast.Field {
	if st.Fields == nil {
		return nil
	}

	for _, field := range st.Fields.List {
		// Check if it's an embedded field (no name)
		if len(field.Names) != 0 {
			continue
		}

		// Get the type of the field
		if !d.isProvideType(pass, field.Type) {
			continue
		}

		return field
	}

	return nil
}

// isProvideType checks if the given expression is the annotation.Provide type.
func (d *provideDetector) isProvideType(pass *analysis.Pass, expr ast.Expr) bool {
	// Use the type checker to get the type
	tv, ok := pass.TypesInfo.Types[expr]
	if !ok {
		// Try using TypeOf for expressions not in Types map
		t := pass.TypesInfo.TypeOf(expr)
		if t == nil {
			return false
		}
		return d.isNamedProvideType(t)
	}

	return d.isNamedProvideType(tv.Type)
}

// isNamedProvideType checks if the type is the annotation.Provide named type.
func (d *provideDetector) isNamedProvideType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil {
		return false
	}

	// Check type name
	if obj.Name() != ProvideTypeName {
		return false
	}

	// Check package path
	pkg := obj.Pkg()
	if pkg == nil {
		return false
	}

	return pkg.Path() == AnnotationPath
}
