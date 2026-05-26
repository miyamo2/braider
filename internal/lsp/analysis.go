package lsp

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	annotationProvidePkg    = "github.com/miyamo2/braider/pkg/annotation"
	annotationProvideIdent  = "Provide"
	annotationInjectIdent   = "Injectable"
	annotationVariableIdent = "Variable"
)

// loadPackageForFile loads the Go package that contains the given file.
// overlay allows the caller to supply unsaved in-memory file content.
func loadPackageForFile(fset *token.FileSet, filename string, overlay map[string][]byte) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps,
		Fset:    fset,
		Overlay: overlay,
		Dir:     filepath.Dir(filename),
	}
	return packages.Load(cfg, ".")
}

// typeContext describes where the cursor sits relative to a braider annotation.
type typeContext int

const (
	contextNone     typeContext = iota
	contextProvide             // inside annotation.Provide[T](...)
	contextInject              // inside annotation.Injectable[T]
	contextVariable            // inside annotation.Variable[T](...)
)

// cursorContext finds the braider annotation context at the given offset in
// the AST, together with the existing type argument text if any.
type cursorContext struct {
	Kind       typeContext
	TypeArgPos token.Pos // position of the type argument node (or 0)
}

// findCursorContext walks the AST for the given file and returns the annotation
// context at the byte offset corresponding to (line, char) (both 0-based).
func findCursorContext(fset *token.FileSet, f *ast.File, line, char int) cursorContext {
	offset := lineCharToOffset(fset, f, line, char)
	if offset < 0 {
		return cursorContext{}
	}

	ctx := cursorContext{}
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		start := fset.Position(n.Pos()).Offset
		end := fset.Position(n.End()).Offset
		if offset < start || offset > end {
			return false
		}
		switch expr := n.(type) {
		case *ast.IndexExpr:
			// Generic: Provide[T] or Injectable[T]
			if name := selectorOrIdent(expr.X); name != "" {
				kind := annotationKind(name)
				if kind != contextNone {
					if offset >= fset.Position(expr.Index.Pos()).Offset &&
						offset <= fset.Position(expr.Index.End()).Offset {
						ctx = cursorContext{Kind: kind, TypeArgPos: expr.Index.Pos()}
					}
				}
			}
		case *ast.IndexListExpr:
			// Multi-type-param: Provide[T1, T2, ...]
			if name := selectorOrIdent(expr.X); name != "" {
				kind := annotationKind(name)
				if kind != contextNone {
					for _, idx := range expr.Indices {
						if offset >= fset.Position(idx.Pos()).Offset &&
							offset <= fset.Position(idx.End()).Offset {
							ctx = cursorContext{Kind: kind, TypeArgPos: idx.Pos()}
						}
					}
				}
			}
		}
		return true
	})
	return ctx
}

// lineCharToOffset converts 0-based (line, char) to byte offset in the file.
// Returns -1 on error.
func lineCharToOffset(fset *token.FileSet, f *ast.File, line, char int) int {
	file := fset.File(f.Pos())
	if file == nil {
		return -1
	}
	targetLine := line + 1 // token.File uses 1-based lines
	if targetLine < 1 || targetLine > file.LineCount() {
		return -1
	}
	lineStart := file.LineStart(targetLine)
	return file.Offset(lineStart) + char
}

// selectorOrIdent returns the local name from either a *ast.SelectorExpr or
// *ast.Ident (e.g., "annotation.Provide" → "Provide", "Provide" → "Provide").
func selectorOrIdent(x ast.Expr) string {
	switch v := x.(type) {
	case *ast.SelectorExpr:
		return v.Sel.Name
	case *ast.Ident:
		return v.Name
	}
	return ""
}

// annotationKind maps a local identifier name to a typeContext.
func annotationKind(name string) typeContext {
	switch name {
	case annotationProvideIdent:
		return contextProvide
	case annotationInjectIdent:
		return contextInject
	case annotationVariableIdent:
		return contextVariable
	}
	return contextNone
}

// collectExportedTypes returns all exported named types from the given package
// and its directly imported dependencies (one level deep for cross-package Provide).
func collectExportedTypes(pkg *packages.Package) []exportedType {
	seen := make(map[string]bool)
	var result []exportedType

	addFromScope := func(scope *types.Scope, pkgPath, pkgName string) {
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if obj == nil || !obj.Exported() {
				continue
			}
			tn, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}
			key := pkgPath + "." + name
			if seen[key] {
				continue
			}
			seen[key] = true
			result = append(result, exportedType{
				LocalName:   name,
				PackagePath: pkgPath,
				PackageName: pkgName,
				IsInterface: types.IsInterface(tn.Type()),
			})
		}
	}

	if pkg.Types != nil {
		addFromScope(pkg.Types.Scope(), pkg.PkgPath, pkg.Name)
	}

	for _, imp := range pkg.Imports {
		if imp.Types != nil {
			addFromScope(imp.Types.Scope(), imp.PkgPath, imp.Name)
		}
	}

	return result
}

// exportedType is a discovered exported type.
type exportedType struct {
	LocalName   string
	PackagePath string
	PackageName string
	IsInterface bool
}

// QualifiedName returns the display form of the type (e.g., "pkg.MyType" for
// cross-package types, "MyType" for same-package types).
func (e exportedType) QualifiedName(currentPkgPath string) string {
	if e.PackagePath == currentPkgPath {
		return e.LocalName
	}
	return e.PackageName + "." + e.LocalName
}

// constructorAtPosition finds the constructor function declaration whose name
// or return type spans the given (line, char) position.
// Returns (funcDecl, returnTypeName, ok).
func constructorAtPosition(fset *token.FileSet, f *ast.File, info *types.Info, line, char int) (*ast.FuncDecl, string, bool) {
	offset := lineCharToOffset(fset, f, line, char)
	if offset < 0 {
		return nil, "", false
	}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		// Must be exported, no receiver (package-level function)
		if fn.Name == nil || !fn.Name.IsExported() || fn.Recv != nil {
			continue
		}
		start := fset.Position(fn.Pos()).Offset
		end := fset.Position(fn.End()).Offset
		if offset < start || offset > end {
			continue
		}
		// Must return exactly one named type (or *Named)
		if fn.Type.Results == nil || len(fn.Type.Results.List) < 1 {
			continue
		}
		retExpr := fn.Type.Results.List[0].Type
		retType := info.TypeOf(retExpr)
		if retType == nil {
			continue
		}
		rt := retType
		if ptr, ok2 := rt.(*types.Pointer); ok2 {
			rt = ptr.Elem()
		}
		named, ok2 := rt.(*types.Named)
		if !ok2 {
			continue
		}
		localName := named.Obj().Name()
		return fn, localName, true
	}
	return nil, "", false
}

// resolvedBinding describes which provider/injector/variable wins for a type.
type resolvedBinding struct {
	Kind          string // "provide", "inject", or "variable"
	ConstructorFn string
	PackagePath   string
	TypeName      string
	Name          string // named binding name, if any
}

// typeAtPosition finds the type name under the cursor for hover purposes.
func typeAtPosition(fset *token.FileSet, f *ast.File, info *types.Info, line, char int) (types.Type, bool) {
	offset := lineCharToOffset(fset, f, line, char)
	if offset < 0 {
		return nil, false
	}

	var found types.Type
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil || found != nil {
			return false
		}
		start := fset.Position(n.Pos()).Offset
		end := fset.Position(n.End()).Offset
		if offset < start || offset > end {
			return false
		}
		if ident, ok := n.(*ast.Ident); ok {
			if t, exists := info.Uses[ident]; exists {
				if _, isType := t.(*types.TypeName); isType {
					found = t.Type()
				}
			}
		}
		return found == nil
	})
	return found, found != nil
}

// uriToFilePath converts an LSP file:// URI to a local filesystem path.
func uriToFilePath(uri string) string {
	path := strings.TrimPrefix(uri, "file://")
	// On Windows, URIs look like file:///C:/...; strip the leading slash.
	if len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return path
}

// filePathToURI converts a local path to an LSP file:// URI.
func filePathToURI(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return "file://" + path
}
