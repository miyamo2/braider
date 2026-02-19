package generate

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

// astIdent creates an *ast.Ident node.
func astIdent(name string) *ast.Ident {
	return &ast.Ident{Name: name}
}

// astSelector creates a selector expression (pkg.Name).
func astSelector(pkg, name string) *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X:   astIdent(pkg),
		Sel: astIdent(name),
	}
}

// astCommentGroup creates a CommentGroup from lines.
// Each line should include the "//" prefix (e.g., "// foo").
func astCommentGroup(lines ...string) *ast.CommentGroup {
	list := make([]*ast.Comment, len(lines))
	for i, line := range lines {
		list[i] = &ast.Comment{Text: line}
	}
	return &ast.CommentGroup{List: list}
}

// astStructType creates an anonymous struct type.
func astStructType(fields ...*ast.Field) *ast.StructType {
	return &ast.StructType{
		Fields: &ast.FieldList{List: fields},
	}
}

// astShortVar creates a short variable declaration (name := value).
func astShortVar(name string, value ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{astIdent(name)},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{value},
	}
}

// astBlankAssign creates a blank assignment (_ = value).
func astBlankAssign(value ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{astIdent("_")},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{value},
	}
}

// astFuncDecl creates a function declaration.
func astFuncDecl(
	doc *ast.CommentGroup, name string, params, results *ast.FieldList, body *ast.BlockStmt,
) *ast.FuncDecl {
	return &ast.FuncDecl{
		Doc:  doc,
		Name: astIdent(name),
		Type: &ast.FuncType{
			Params:  params,
			Results: results,
		},
		Body: body,
	}
}

// astFuncLit creates a function literal.
func astFuncLit(params, results *ast.FieldList, body *ast.BlockStmt) *ast.FuncLit {
	return &ast.FuncLit{
		Type: &ast.FuncType{
			Params:  params,
			Results: results,
		},
		Body: body,
	}
}

// astVarDecl creates a var declaration (var name = value) with optional doc comment.
func astVarDecl(doc *ast.CommentGroup, name string, value ast.Expr) *ast.GenDecl {
	return &ast.GenDecl{
		Doc: doc,
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{astIdent(name)},
				Values: []ast.Expr{value},
			},
		},
	}
}

// astImportDecl creates an import declaration with parenthesized form.
func astImportDecl(specs []ast.Spec) *ast.GenDecl {
	return &ast.GenDecl{
		Tok:    token.IMPORT,
		Lparen: 1, // Non-zero to force parenthesized form
		Specs:  specs,
	}
}

// astImportSpec creates an import spec with optional alias.
func astImportSpec(alias, path string) *ast.ImportSpec {
	spec := &ast.ImportSpec{
		Path: &ast.BasicLit{Kind: token.STRING, Value: `"` + path + `"`},
	}
	if alias != "" {
		spec.Name = astIdent(alias)
	}
	return spec
}

// parseExprString parses a Go expression string into an ast.Expr.
// Uses go/parser.ParseExpr and then clears all position information.
// Suitable for type expressions from types.TypeString output and simple expressions.
func parseExprString(s string) (ast.Expr, error) {
	expr, err := parser.ParseExpr(s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression %q: %w", s, err)
	}
	clearPositions(expr)
	return expr, nil
}

// clearPositions recursively clears all position information in an AST node.
// This prevents conflicts when embedding parsed nodes into a NoPos-based tree.
func clearPositions(node ast.Node) {
	ast.Inspect(
		node, func(n ast.Node) bool {
			if n == nil {
				return false
			}
			switch x := n.(type) {
			case *ast.Ident:
				x.NamePos = token.NoPos
			case *ast.BasicLit:
				x.ValuePos = token.NoPos
			case *ast.StarExpr:
				x.Star = token.NoPos
			case *ast.UnaryExpr:
				x.OpPos = token.NoPos
			case *ast.BinaryExpr:
				x.OpPos = token.NoPos
			case *ast.SelectorExpr:
				// Positions cleared via children
			case *ast.ArrayType:
				x.Lbrack = token.NoPos
			case *ast.MapType:
				x.Map = token.NoPos
			case *ast.ChanType:
				x.Begin = token.NoPos
				x.Arrow = token.NoPos
			case *ast.ParenExpr:
				x.Lparen = token.NoPos
				x.Rparen = token.NoPos
			case *ast.FuncType:
				x.Func = token.NoPos
			case *ast.StructType:
				x.Struct = token.NoPos
			case *ast.InterfaceType:
				x.Interface = token.NoPos
			case *ast.Ellipsis:
				x.Ellipsis = token.NoPos
			case *ast.IndexExpr:
				x.Lbrack = token.NoPos
				x.Rbrack = token.NoPos
			case *ast.FieldList:
				x.Opening = token.NoPos
				x.Closing = token.NoPos
			case *ast.Field:
				// Positions cleared via children
			case *ast.SliceExpr:
				x.Lbrack = token.NoPos
				x.Rbrack = token.NoPos
			case *ast.TypeAssertExpr:
				x.Lparen = token.NoPos
				x.Rparen = token.NoPos
			case *ast.CallExpr:
				x.Lparen = token.NoPos
				x.Rparen = token.NoPos
			case *ast.CompositeLit:
				x.Lbrace = token.NoPos
				x.Rbrace = token.NoPos
			case *ast.KeyValueExpr:
				x.Colon = token.NoPos
			}
			return true
		},
	)
}

// renderNode formats an AST node using format.Node and returns the string.
func renderNode(node ast.Node) (string, error) {
	var buf bytes.Buffer
	fset := token.NewFileSet()
	if err := format.Node(&buf, fset, node); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderDecl formats an AST declaration (FuncDecl, GenDecl) as Go source.
// It wraps the declaration in a dummy file, assigns synthetic line-based
// positions to produce multi-line layout for composite literals and proper
// comment placement, then uses format.Node. The package declaration prefix
// is stripped from the output.
func renderDecl(decl ast.Decl) (string, error) {
	file := &ast.File{
		Name:  astIdent("_"),
		Decls: []ast.Decl{decl},
	}

	// Attach doc comments to the file's comment list
	// so that format.Node can properly render them.
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Doc != nil {
			file.Comments = append(file.Comments, d.Doc)
		}
	case *ast.GenDecl:
		if d.Doc != nil {
			file.Comments = append(file.Comments, d.Doc)
		}
	}

	// Assign synthetic positions to force proper layout
	fset := assignPositions(file)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return "", err
	}

	// Strip "package _\n" prefix and leading blank lines
	s := buf.String()
	if idx := strings.Index(s, "\n\n"); idx >= 0 {
		return s[idx+2:], nil
	}
	return s, nil
}

// posAssigner provides incrementing line-based positions.
type posAssigner struct {
	tf   *token.File
	line int
	max  int
}

func newPosAssigner(fset *token.FileSet, nodeCount int) *posAssigner {
	lineWidth := 100
	maxLines := nodeCount*2 + 20
	size := maxLines * lineWidth
	tf := fset.AddFile("", -1, size)

	lines := make([]int, maxLines)
	for i := range lines {
		lines[i] = i * lineWidth
	}
	tf.SetLines(lines)

	return &posAssigner{tf: tf, line: 1, max: maxLines}
}

func (pa *posAssigner) next() token.Pos {
	if pa.line >= pa.max {
		return token.NoPos
	}
	p := pa.tf.LineStart(pa.line)
	pa.line++
	return p
}

// assignPositions walks the AST and assigns line-based positions to control
// format.Node's layout decisions. The key goals are:
// - Doc comments on separate lines immediately before the declaration
// - Function parameters on the same line (not multi-line)
// - Composite literal elements on separate lines (multi-line)
// - Struct type fields on separate lines (multi-line)
func assignPositions(file *ast.File) *token.FileSet {
	fset := token.NewFileSet()

	// Count nodes to estimate space needed
	var nodeCount int
	ast.Inspect(
		file, func(n ast.Node) bool {
			if n != nil {
				nodeCount++
			}
			return true
		},
	)

	pa := newPosAssigner(fset, nodeCount)

	// Package name on line 1
	file.Name.NamePos = pa.next()
	// Blank line between package and decls
	pa.next()

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			assignFuncDeclPositions(d, pa)
		case *ast.GenDecl:
			assignGenDeclPositions(d, pa)
		}
	}

	return fset
}

// assignFuncDeclPositions assigns positions for a function declaration,
// keeping parameters inline and composite literal elements multi-line.
func assignFuncDeclPositions(decl *ast.FuncDecl, pa *posAssigner) {
	// Doc comment: each line on its own line
	if decl.Doc != nil {
		for _, c := range decl.Doc.List {
			c.Slash = pa.next()
		}
	}

	// func keyword on next line (immediately after doc comment)
	funcLine := pa.next()
	decl.Type.Func = funcLine
	decl.Name.NamePos = funcLine

	// Parameters: all on the same line as func keyword
	if decl.Type.Params != nil {
		assignFieldListInline(decl.Type.Params, funcLine)
	}

	// Results: same line as func keyword
	if decl.Type.Results != nil {
		assignFieldListInline(decl.Type.Results, funcLine)
	}

	// Body opening brace: same line as func
	if decl.Body != nil {
		decl.Body.Lbrace = funcLine
		assignBlockStmtPositions(decl.Body, pa)
		decl.Body.Rbrace = pa.next()
	}
}

// assignGenDeclPositions assigns positions for a general declaration (var, import).
func assignGenDeclPositions(decl *ast.GenDecl, pa *posAssigner) {
	// Doc comment
	if decl.Doc != nil {
		for _, c := range decl.Doc.List {
			c.Slash = pa.next()
		}
	}

	// var/import keyword
	declLine := pa.next()
	decl.TokPos = declLine

	if decl.Lparen != 0 {
		decl.Lparen = pa.next()
	}

	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			assignValueSpecPositions(s, pa)
		case *ast.ImportSpec:
			assignImportSpecPositions(s, pa)
		}
	}

	if decl.Lparen != 0 {
		decl.Rparen = pa.next()
	}
}

// assignFieldListInline assigns all fields in a FieldList to the same line.
func assignFieldListInline(fl *ast.FieldList, linePos token.Pos) {
	fl.Opening = linePos
	fl.Closing = linePos
	for _, f := range fl.List {
		for _, name := range f.Names {
			name.NamePos = linePos
		}
		assignExprInline(f.Type, linePos)
	}
}

// assignExprInline assigns all positions in an expression to the same line.
func assignExprInline(expr ast.Expr, linePos token.Pos) {
	if expr == nil {
		return
	}
	ast.Inspect(
		expr, func(n ast.Node) bool {
			if n == nil {
				return false
			}
			switch x := n.(type) {
			case *ast.Ident:
				x.NamePos = linePos
			case *ast.BasicLit:
				x.ValuePos = linePos
			case *ast.StarExpr:
				x.Star = linePos
			case *ast.UnaryExpr:
				x.OpPos = linePos
			case *ast.SelectorExpr:
				// Children handle positions
			case *ast.ArrayType:
				x.Lbrack = linePos
			case *ast.MapType:
				x.Map = linePos
			case *ast.ChanType:
				x.Begin = linePos
			case *ast.ParenExpr:
				x.Lparen = linePos
				x.Rparen = linePos
			case *ast.Ellipsis:
				x.Ellipsis = linePos
			case *ast.IndexExpr:
				x.Lbrack = linePos
				x.Rbrack = linePos
			case *ast.FieldList:
				x.Opening = linePos
				x.Closing = linePos
			case *ast.InterfaceType:
				x.Interface = linePos
			case *ast.StructType:
				x.Struct = linePos
			case *ast.FuncType:
				x.Func = linePos
			case *ast.CallExpr:
				x.Lparen = linePos
				x.Rparen = linePos
			}
			return true
		},
	)
}

// assignBlockStmtPositions assigns positions for statements in a block,
// with each statement on its own line.
func assignBlockStmtPositions(block *ast.BlockStmt, pa *posAssigner) {
	for _, stmt := range block.List {
		assignStmtPositions(stmt, pa)
	}
}

// assignStmtPositions assigns positions for a single statement.
func assignStmtPositions(stmt ast.Stmt, pa *posAssigner) {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		stmtLine := pa.next()
		s.Return = stmtLine
		for _, result := range s.Results {
			assignExprMultiLine(result, pa, stmtLine)
		}
	case *ast.AssignStmt:
		stmtLine := pa.next()
		s.TokPos = stmtLine
		for _, lhs := range s.Lhs {
			assignExprInline(lhs, stmtLine)
		}
		for _, rhs := range s.Rhs {
			assignExprMultiLine(rhs, pa, stmtLine)
		}
	case *ast.ExprStmt:
		stmtLine := pa.next()
		assignExprMultiLine(s.X, pa, stmtLine)
	}
}

// assignExprMultiLine assigns positions for expressions, putting composite
// literal elements and struct fields on separate lines while keeping
// simple expressions inline.
func assignExprMultiLine(expr ast.Expr, pa *posAssigner, startLine token.Pos) {
	if expr == nil {
		return
	}
	switch x := expr.(type) {
	case *ast.UnaryExpr:
		x.OpPos = startLine
		assignExprMultiLine(x.X, pa, startLine)
	case *ast.CompositeLit:
		// Struct types in composite literals need multi-line field layout
		if st, ok := x.Type.(*ast.StructType); ok {
			lastLine := assignStructTypeMultiLine(st, pa, startLine)
			x.Lbrace = lastLine // Lbrace on same line as struct close: }{
		} else {
			x.Lbrace = startLine
			assignExprInline(x.Type, startLine)
		}
		// Each element on its own line for multi-line format
		for _, elt := range x.Elts {
			eltLine := pa.next()
			assignExprInline(elt, eltLine)
		}
		if len(x.Elts) > 0 {
			x.Rbrace = pa.next()
		} else {
			x.Rbrace = x.Lbrace
		}
	case *ast.CallExpr:
		// IIFE: FuncLit needs multi-line treatment, not inline
		if fl, ok := x.Fun.(*ast.FuncLit); ok {
			assignExprMultiLine(fl, pa, startLine)
			// Call parentheses on same line as body closing brace: }()
			x.Lparen = fl.Body.Rbrace
			x.Rparen = fl.Body.Rbrace
		} else {
			assignExprInline(x.Fun, startLine)
			x.Lparen = startLine
			x.Rparen = startLine
		}
		for _, arg := range x.Args {
			assignExprInline(arg, startLine)
		}
	case *ast.FuncLit:
		lastLine := startLine
		if x.Type != nil {
			x.Type.Func = startLine
			if x.Type.Params != nil {
				assignFieldListInline(x.Type.Params, startLine)
			}
			if x.Type.Results != nil {
				lastLine = assignResultsMultiLine(x.Type.Results, pa, startLine)
			}
		}
		if x.Body != nil {
			// Body lbrace on same line as result type closing: } {
			x.Body.Lbrace = lastLine
			assignBlockStmtPositions(x.Body, pa)
			x.Body.Rbrace = pa.next()
		}
	case *ast.Ident:
		x.NamePos = startLine
	case *ast.SelectorExpr:
		assignExprInline(x.X, startLine)
		x.Sel.NamePos = startLine
	case *ast.StarExpr:
		x.Star = startLine
		assignExprMultiLine(x.X, pa, startLine)
	default:
		assignExprInline(expr, startLine)
	}
}

// assignStructTypeMultiLine assigns positions for a struct type with fields
// on separate lines. Returns the position of the closing brace.
func assignStructTypeMultiLine(st *ast.StructType, pa *posAssigner, startLine token.Pos) token.Pos {
	st.Struct = startLine
	if st.Fields == nil {
		return startLine
	}
	st.Fields.Opening = startLine
	for _, sf := range st.Fields.List {
		sfLine := pa.next()
		for _, name := range sf.Names {
			name.NamePos = sfLine
		}
		assignExprInline(sf.Type, sfLine)
	}
	// Always put closing on next line for multi-line layout (even empty structs)
	closeLine := pa.next()
	st.Fields.Closing = closeLine
	return closeLine
}

// assignResultsMultiLine assigns positions for function result types.
// Struct types get multi-line treatment (fields on separate lines).
// Returns the last line used (for positioning the body lbrace after).
func assignResultsMultiLine(fl *ast.FieldList, pa *posAssigner, startLine token.Pos) token.Pos {
	fl.Opening = startLine
	lastLine := startLine
	for _, f := range fl.List {
		for _, name := range f.Names {
			name.NamePos = startLine
		}
		switch t := f.Type.(type) {
		case *ast.StructType:
			lastLine = assignStructTypeMultiLine(t, pa, startLine)
		default:
			assignExprInline(f.Type, startLine)
		}
	}
	fl.Closing = startLine
	return lastLine
}

// assignValueSpecPositions assigns positions for a var value spec.
func assignValueSpecPositions(spec *ast.ValueSpec, pa *posAssigner) {
	specLine := pa.next()
	for _, name := range spec.Names {
		name.NamePos = specLine
	}
	for _, value := range spec.Values {
		assignExprMultiLine(value, pa, specLine)
	}
}

// RenderImportBlock renders a sorted list of imports as a parenthesized import block string.
// Returns empty string for empty imports. Uses AST construction + format.Node.
func RenderImportBlock(sortedImports []ImportInfo) (string, error) {
	if len(sortedImports) == 0 {
		return "", nil
	}
	var specs []ast.Spec
	for _, imp := range sortedImports {
		specs = append(specs, astImportSpec(imp.Alias, imp.Path))
	}
	decl := astImportDecl(specs)
	return renderDecl(decl)
}

// assignImportSpecPositions assigns positions for an import spec.
func assignImportSpecPositions(spec *ast.ImportSpec, pa *posAssigner) {
	line := pa.next()
	if spec.Name != nil {
		spec.Name.NamePos = line
	}
	spec.Path.ValuePos = line
}
