// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package free defines utilities for computing the free variables of
// a syntax tree without type information. This is inherently
// heuristic because of the T{f: x} ambiguity, in which f may or may
// not be a lexical reference depending on whether T is a struct type.
package free

import (
	"go/ast"
	"go/token"
)

// Names computes an approximation to the set of free names of the AST
// at node n based solely on syntax.
//
// In the absence of composite literals, the set of free names is exact. Composite
// literals introduce an ambiguity that can only be resolved with type information:
// whether F is a field name or a value in `T{F: ...}`.
// If includeComplitIdents is true, this function conservatively assumes
// T is not a struct type, so freeishNames overapproximates: the resulting
// set may contain spurious entries that are not free lexical references
// but are references to struct fields.
// If includeComplitIdents is false, this function assumes that T *is*
// a struct type, so freeishNames underapproximates: the resulting set
// may omit names that are free lexical references.
func Names(n ast.Node, includeComplitIdents bool) map[string]bool {
	v := &freeVisitor{
		free:                 make(map[string]bool),
		includeComplitIdents: includeComplitIdents,
	}
	v.openScope()
	ast.Walk(v, n)
	v.closeScope()
	if v.scope != nil {
		panic("unbalanced scopes")
	}
	return v.free
}

// A freeVisitor holds state for a free-name analysis.
type freeVisitor struct {
	scope                *scope          // the current innermost scope
	free                 map[string]bool // free names seen so far
	includeComplitIdents bool            // include identifier key in composite literals
}

// scope contains all the names defined in a lexical scope.
type scope struct {
	names map[string]bool
	outer *scope
}

func (s *scope) defined(name string) bool {
	for ; s != nil; s = s.outer {
		if s.names[name] {
			return true
		}
	}
	return false
}

func (v *freeVisitor) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {

	// Expressions.
	case *ast.Ident:
		v.use(n)

	case *ast.FuncLit:
		v.openScope()
		defer v.closeScope()
		v.walkFuncType(nil, n.Type)
		v.walkBody(n.Body)

	case *ast.SelectorExpr:
		v.walk(n.X)
		// Skip n.Sel: it cannot be free.

	case *ast.StructType:
		v.openScope()
		defer v.closeScope()
		v.walkFieldList(n.Fields)

	case *ast.FuncType:
		v.openScope()
		defer v.closeScope()
		v.walkFuncType(nil, n)

	case *ast.CompositeLit:
		v.walk(n.Type)
		for _, e := range n.Elts {
			if kv, _ := e.(*ast.KeyValueExpr); kv != nil {
				if ident, _ := kv.Key.(*ast.Ident); ident != nil {
					if v.includeComplitIdents {
						v.use(ident)
					}
				} else {
					v.walk(kv.Key)
				}
				v.walk(kv.Value)
			} else {
				v.walk(e)
			}
		}

	case *ast.InterfaceType:
		v.openScope()
		defer v.closeScope()
		v.walkFieldList(n.Methods)

	// Statements
	case *ast.AssignStmt:
		walkSlice(v, n.Rhs)
		if n.Tok == token.DEFINE {
			v.shortVarDecl(n.Lhs)
		} else {
			walkSlice(v, n.Lhs)
		}

	case *ast.LabeledStmt:
		v.walk(n.Stmt)

	case *ast.BranchStmt:
		// Ignore labels.

	case *ast.BlockStmt:
		v.openScope()
		defer v.closeScope()
		walkSlice(v, n.List)

	case *ast.IfStmt:
		v.openScope()
		defer v.closeScope()
		v.walk(n.Init)
		v.walk(n.Cond)
		v.walk(n.Body)
		v.walk(n.Else)

	case *ast.CaseClause:
		walkSlice(v, n.List)
		v.openScope()
		defer v.closeScope()
		walkSlice(v, n.Body)

	case *ast.SwitchStmt:
		v.openScope()
		defer v.closeScope()
		v.walk(n.Init)
		v.walk(n.Tag)
		v.walkBody(n.Body)

	case *ast.TypeSwitchStmt:
		v.openScope()
		defer v.closeScope()
		if n.Init != nil {
			v.walk(n.Init)
		}
		v.walk(n.Assign)
		v.walkBody(n.Body)

	case *ast.CommClause:
		v.openScope()
		defer v.closeScope()
		v.walk(n.Comm)
		walkSlice(v, n.Body)

	case *ast.SelectStmt:
		v.walkBody(n.Body)

	case *ast.ForStmt:
		v.openScope()
		defer v.closeScope()
		v.walk(n.Init)
		v.walk(n.Cond)
		v.walk(n.Post)
		v.walk(n.Body)

	case *ast.RangeStmt:
		v.openScope()
		defer v.closeScope()
		v.walk(n.X)
		var lhs []ast.Expr
		if n.Key != nil {
			lhs = append(lhs, n.Key)
		}
		if n.Value != nil {
			lhs = append(lhs, n.Value)
		}
		if len(lhs) > 0 {
			if n.Tok == token.DEFINE {
				v.shortVarDecl(lhs)
			} else {
				walkSlice(v, lhs)
			}
		}
		v.walk(n.Body)

	// Declarations
	case *ast.GenDecl:
		switch n.Tok {
		case token.CONST, token.VAR:
			for _, spec := range n.Specs {
				spec := spec.(*ast.ValueSpec)
				walkSlice(v, spec.Values)
				v.walk(spec.Type)
				v.declare(spec.Names...)
			}

		case token.TYPE:
			for _, spec := range n.Specs {
				spec := spec.(*ast.TypeSpec)
				v.declare(spec.Name)
				if spec.TypeParams != nil {
					v.openScope()
					defer v.closeScope()
					v.walkTypeParams(spec.TypeParams)
				}
				v.walk(spec.Type)
			}

		case token.IMPORT:
			panic("encountered import declaration in free analysis")
		}

	case *ast.FuncDecl:
		if n.Recv == nil && n.Name.Name != "init" {
			v.declare(n.Name)
		}
		v.openScope()
		defer v.closeScope()
		v.walkTypeParams(n.Type.TypeParams)
		v.walkFuncType(n.Recv, n.Type)
		v.walkBody(n.Body)

	default:
		return v
	}

	return nil
}

func (v *freeVisitor) openScope() {
	v.scope = &scope{map[string]bool{}, v.scope}
}

func (v *freeVisitor) closeScope() {
	v.scope = v.scope.outer
}

func (v *freeVisitor) walk(n ast.Node) {
	if n != nil {
		ast.Walk(v, n)
	}
}

func (v *freeVisitor) walkFuncType(recv *ast.FieldList, typ *ast.FuncType) {
	v.walkRecvFieldType(recv)
	v.walkFieldTypes(typ.Params)
	v.walkFieldTypes(typ.Results)

	v.declareFieldNames(recv)
	v.declareFieldNames(typ.Params)
	v.declareFieldNames(typ.Results)
}

func (v *freeVisitor) walkRecvFieldType(list *ast.FieldList) {
	if list == nil {
		return
	}
	for _, f := range list.List {
		typ := f.Type
		if ptr, ok := typ.(*ast.StarExpr); ok {
			typ = ptr.X
		}

		var (
			base    ast.Expr
			indices []ast.Expr
		)
		switch typ := typ.(type) {
		case *ast.IndexExpr:
			base, indices = typ.X, []ast.Expr{typ.Index}
		case *ast.IndexListExpr:
			base, indices = typ.X, typ.Indices
		default:
			base = typ
		}
		for _, expr := range indices {
			if id, ok := expr.(*ast.Ident); ok {
				v.declare(id)
			}
		}
		v.walk(base)
	}
}

func (v *freeVisitor) walkTypeParams(list *ast.FieldList) {
	v.declareFieldNames(list)
	v.walkFieldTypes(list)
}

func (v *freeVisitor) walkBody(body *ast.BlockStmt) {
	if body == nil {
		return
	}
	walkSlice(v, body.List)
}

func (v *freeVisitor) walkFieldList(list *ast.FieldList) {
	if list == nil {
		return
	}
	v.walkFieldTypes(list)
	v.declareFieldNames(list)
}

func (v *freeVisitor) shortVarDecl(lhs []ast.Expr) {
	for _, x := range lhs {
		if id, ok := x.(*ast.Ident); ok {
			v.declare(id)
		}
	}
}

func walkSlice[S ~[]E, E ast.Node](r *freeVisitor, list S) {
	for _, e := range list {
		r.walk(e)
	}
}

func (v *freeVisitor) walkFieldTypes(list *ast.FieldList) {
	if list != nil {
		for _, f := range list.List {
			v.walk(f.Type)
		}
	}
}

func (v *freeVisitor) declareFieldNames(list *ast.FieldList) {
	if list != nil {
		for _, f := range list.List {
			v.declare(f.Names...)
		}
	}
}

func (v *freeVisitor) use(ident *ast.Ident) {
	if s := ident.Name; s != "_" && !v.scope.defined(s) {
		v.free[s] = true
	}
}

func (v *freeVisitor) declare(idents ...*ast.Ident) {
	for _, id := range idents {
		if id.Name != "_" {
			v.scope.names[id.Name] = true
		}
	}
}
