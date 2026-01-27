// Package detect provides detection capabilities for braider analyzer.
package detect

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// InjectAnnotationPath is the import path for the annotation package.
const InjectAnnotationPath = "github.com/miyamo2/braider/pkg/annotation"

// InjectTypeName is the type name for the Inject annotation.
const InjectTypeName = "Inject"

// InjectDetector detects annotation.Inject embedding in structs.
type InjectDetector interface {
	// HasInjectAnnotation checks if a struct embeds annotation.Inject.
	// Returns true if the struct has an embedded annotation.Inject field.
	HasInjectAnnotation(pass *analysis.Pass, st *ast.StructType) bool

	// FindInjectField returns the embedded Inject field if present.
	// Returns nil if no Inject embedding is found.
	FindInjectField(pass *analysis.Pass, st *ast.StructType) *ast.Field
}

// injectDetector is the default implementation of InjectDetector.
type injectDetector struct{}

// NewInjectDetector creates a new InjectDetector instance.
func NewInjectDetector() InjectDetector {
	return &injectDetector{}
}

// HasInjectAnnotation checks if a struct embeds annotation.Inject.
func (d *injectDetector) HasInjectAnnotation(pass *analysis.Pass, st *ast.StructType) bool {
	return d.FindInjectField(pass, st) != nil
}

// FindInjectField returns the embedded Inject field if present.
func (d *injectDetector) FindInjectField(pass *analysis.Pass, st *ast.StructType) *ast.Field {
	if st.Fields == nil {
		return nil
	}

	for _, field := range st.Fields.List {
		// Check if it's an embedded field (no name)
		if len(field.Names) != 0 {
			continue
		}

		// Get the type of the field
		if !d.isInjectType(pass, field.Type) {
			continue
		}

		return field
	}

	return nil
}

// isInjectType checks if the given expression is the annotation.Inject type.
func (d *injectDetector) isInjectType(pass *analysis.Pass, expr ast.Expr) bool {
	// Use the type checker to get the type
	tv, ok := pass.TypesInfo.Types[expr]
	if !ok {
		// Try using TypeOf for expressions not in Types map
		t := pass.TypesInfo.TypeOf(expr)
		if t == nil {
			return false
		}
		return d.isNamedInjectType(t)
	}

	return d.isNamedInjectType(tv.Type)
}

// isNamedInjectType checks if the type is the annotation.Inject named type.
func (d *injectDetector) isNamedInjectType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil {
		return false
	}

	// Check type name
	if obj.Name() != InjectTypeName {
		return false
	}

	// Check package path
	pkg := obj.Pkg()
	if pkg == nil {
		return false
	}

	return pkg.Path() == InjectAnnotationPath
}
