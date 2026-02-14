package detect

import (
	"go/ast"
	"go/types"
	"reflect"
	"strconv"
	"unicode"

	"golang.org/x/tools/go/analysis"
)

// FieldInfo contains analyzed information about a struct field.
type FieldInfo struct {
	Name            string     // Field name (or generated name for anonymous)
	TypeExpr        ast.Expr   // Original type expression from AST
	Type            types.Type // Resolved type from type checker
	IsExported      bool       // Whether the field is exported
	IsPointer       bool       // Whether the type is a pointer
	IsInterface     bool       // Whether the type is an interface
	NamedDependency string     // Named dependency from braider:"name" tag (empty if not tagged)
	Excluded        bool       // True if field has braider:"-" tag (excluded from DI)
	InvalidTag      bool       // True if field has braider:"" tag (empty value, invalid)
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

		// Parse struct tag for braider key (shared across all names in same field declaration)
		tagInfo := a.parseStructTag(field)

		// Process each name in the field (handles "a, b int" syntax)
		for _, name := range field.Names {
			info := a.analyzeField(pass, name.Name, field.Type)
			info.NamedDependency = tagInfo.namedDependency
			info.Excluded = tagInfo.excluded
			info.InvalidTag = tagInfo.invalidTag
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

// structTagInfo holds parsed braider struct tag metadata.
type structTagInfo struct {
	namedDependency string // Named dependency value (empty if not tagged or excluded)
	excluded        bool   // True if braider:"-"
	invalidTag      bool   // True if braider:"" (empty value)
}

// parseStructTag parses the braider struct tag from an ast.Field.
// It extracts the raw tag string from the field's Tag literal, strips surrounding
// backticks via strconv.Unquote, and uses reflect.StructTag.Lookup("braider") to
// find the braider key value.
func (a *fieldAnalyzer) parseStructTag(field *ast.Field) structTagInfo {
	if field.Tag == nil {
		return structTagInfo{}
	}

	// field.Tag.Value includes surrounding backticks or quotes (e.g., `json:"x" braider:"name"`)
	rawTag, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		// Non-standard tag format; treat as untagged
		return structTagInfo{}
	}

	value, ok := reflect.StructTag(rawTag).Lookup("braider")
	if !ok {
		// No braider key present; treat as standard dependency
		return structTagInfo{}
	}

	// Empty value: braider:""
	if value == "" {
		return structTagInfo{invalidTag: true}
	}

	// Exclusion: braider:"-"
	if value == "-" {
		return structTagInfo{excluded: true}
	}

	// Named dependency: braider:"name"
	return structTagInfo{namedDependency: value}
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
