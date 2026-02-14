package detect

import (
	"go/ast"
	"go/types"
	"unicode"

	"golang.org/x/tools/go/analysis"
)

// FieldInfo contains analyzed information about a struct field.
type FieldInfo struct {
	Name        string     // Field name (or generated name for anonymous)
	TypeExpr    ast.Expr   // Original type expression from AST
	Type        types.Type // Resolved type from type checker
	IsExported  bool       // Whether the field is exported
	IsPointer   bool       // Whether the type is a pointer
	IsInterface bool       // Whether the type is an interface
}

// FieldAnalyzer analyzes struct fields for constructor generation.
type FieldAnalyzer interface {
	// AnalyzeFields extracts injectable fields from a struct.
	// Excludes the embedded annotation.Injectable field from results.
	AnalyzeFields(pass *analysis.Pass, st *ast.StructType, injectField *ast.Field) []FieldInfo
}

// fieldAnalyzer is the default implementation of FieldAnalyzer.
type fieldAnalyzer struct{}

// NewFieldAnalyzer creates a new FieldAnalyzer instance.
func NewFieldAnalyzer() FieldAnalyzer {
	return &fieldAnalyzer{}
}

// AnalyzeFields extracts injectable fields from a struct.
func (a *fieldAnalyzer) AnalyzeFields(pass *analysis.Pass, st *ast.StructType, injectField *ast.Field) []FieldInfo {
	if st.Fields == nil {
		return nil
	}

	var fields []FieldInfo

	for _, field := range st.Fields.List {
		// Skip the inject field
		if field == injectField {
			continue
		}

		// Skip embedded fields (no names)
		if len(field.Names) == 0 {
			continue
		}

		// Process each name in the field (handles "a, b int" syntax)
		for _, name := range field.Names {
			info := a.analyzeField(pass, name.Name, field.Type)
			fields = append(fields, info)
		}
	}

	return fields
}

// analyzeField analyzes a single field and returns its FieldInfo.
func (a *fieldAnalyzer) analyzeField(pass *analysis.Pass, name string, typeExpr ast.Expr) FieldInfo {
	info := FieldInfo{
		Name:       name,
		TypeExpr:   typeExpr,
		IsExported: isExported(name),
	}

	// Get the resolved type from type checker
	if pass.TypesInfo != nil {
		info.Type = pass.TypesInfo.TypeOf(typeExpr)
		if info.Type != nil {
			info.IsPointer = isPointerType(info.Type)
			info.IsInterface = isInterfaceType(info.Type)
		}
	}

	// Fallback: analyze AST if type info is not available
	if info.Type == nil {
		info.IsPointer = isPointerAST(typeExpr)
		info.IsInterface = false // Cannot determine from AST alone
	}

	return info
}

// isExported returns true if the name is exported (starts with uppercase).
func isExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}

// isPointerType returns true if the type is a pointer type.
func isPointerType(t types.Type) bool {
	_, ok := t.(*types.Pointer)
	return ok
}

// isInterfaceType returns true if the type is an interface type.
func isInterfaceType(t types.Type) bool {
	// Handle named types
	if named, ok := t.(*types.Named); ok {
		_, ok := named.Underlying().(*types.Interface)
		return ok
	}

	// Handle direct interface types
	_, ok := t.(*types.Interface)
	return ok
}

// isPointerAST checks if the expression is a pointer type using AST.
func isPointerAST(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}
