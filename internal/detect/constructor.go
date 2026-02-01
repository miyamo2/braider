package detect

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// ConstructorAnalyzer extracts dependency information from constructors.
type ConstructorAnalyzer interface {
	// ExtractDependencies returns fully qualified type names of constructor parameters.
	ExtractDependencies(pass *analysis.Pass, ctor *ast.FuncDecl) []string
}

// constructorAnalyzer is the default implementation of ConstructorAnalyzer.
type constructorAnalyzer struct{}

// NewConstructorAnalyzer creates a new ConstructorAnalyzer instance.
func NewConstructorAnalyzer() ConstructorAnalyzer {
	return &constructorAnalyzer{}
}

// ExtractDependencies returns fully qualified type names of constructor parameters.
func (a *constructorAnalyzer) ExtractDependencies(pass *analysis.Pass, ctor *ast.FuncDecl) []string {
	if ctor == nil || ctor.Type.Params == nil {
		return nil
	}

	var deps []string

	for _, param := range ctor.Type.Params.List {
		t := pass.TypesInfo.TypeOf(param.Type)
		if t == nil {
			continue
		}

		fqn := a.fullyQualifiedTypeName(t)
		if fqn != "" {
			// Handle multiple names in a single param (e.g., "a, b int")
			count := len(param.Names)
			if count == 0 {
				count = 1
			}
			for i := 0; i < count; i++ {
				deps = append(deps, fqn)
			}
		}
	}

	return deps
}

// fullyQualifiedTypeName returns the fully qualified name of a type.
func (a *constructorAnalyzer) fullyQualifiedTypeName(t types.Type) string {
	switch typ := t.(type) {
	case *types.Pointer:
		elem := a.fullyQualifiedTypeName(typ.Elem())
		if elem == "" {
			return ""
		}
		return "*" + elem
	case *types.Named:
		obj := typ.Obj()
		if obj.Pkg() != nil {
			return obj.Pkg().Path() + "." + obj.Name()
		}
		return obj.Name()
	case *types.Basic:
		return typ.Name()
	case *types.Interface:
		// For named interfaces, get the qualified name
		// For anonymous interfaces, return empty
		return ""
	case *types.Slice:
		elem := a.fullyQualifiedTypeName(typ.Elem())
		if elem == "" {
			return ""
		}
		return "[]" + elem
	case *types.Array:
		elem := a.fullyQualifiedTypeName(typ.Elem())
		if elem == "" {
			return ""
		}
		return "[]" + elem // Simplified: treat as slice for dependency purposes
	case *types.Map:
		key := a.fullyQualifiedTypeName(typ.Key())
		val := a.fullyQualifiedTypeName(typ.Elem())
		if key == "" || val == "" {
			return ""
		}
		return "map[" + key + "]" + val
	case *types.Chan:
		elem := a.fullyQualifiedTypeName(typ.Elem())
		if elem == "" {
			return ""
		}
		return "chan " + elem
	default:
		return ""
	}
}
