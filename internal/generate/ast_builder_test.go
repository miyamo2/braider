package generate

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

func TestParseExprString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "simple identifier",
			input:   "int",
			wantErr: false,
		},
		{
			name:    "pointer type",
			input:   "*Foo",
			wantErr: false,
		},
		{
			name:    "selector expression",
			input:   "pkg.Type",
			wantErr: false,
		},
		{
			name:    "star selector",
			input:   "*pkg.Type",
			wantErr: false,
		},
		{
			name:    "slice type",
			input:   "[]string",
			wantErr: false,
		},
		{
			name:    "map type",
			input:   "map[string]int",
			wantErr: false,
		},
		{
			name:    "channel type",
			input:   "chan int",
			wantErr: false,
		},
		{
			name:    "func type",
			input:   "func(int) string",
			wantErr: false,
		},
		{
			name:    "interface type",
			input:   "interface{}",
			wantErr: false,
		},
		{
			name:    "invalid expression",
			input:   "func {",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parseExprString(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseExprString(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseExprString(%q) unexpected error: %v", tt.input, err)
				return
			}
			if expr == nil {
				t.Errorf("parseExprString(%q) returned nil expr", tt.input)
			}
		})
	}
}

func TestClearPositions(t *testing.T) {
	tests := []struct {
		name    string
		node    ast.Node
		checkFn func(t *testing.T, node ast.Node)
	}{
		{
			name: "Ident",
			node: &ast.Ident{Name: "foo", NamePos: 42},
			checkFn: func(t *testing.T, node ast.Node) {
				x := node.(*ast.Ident)
				if x.NamePos != token.NoPos {
					t.Errorf("Ident.NamePos = %v, want NoPos", x.NamePos)
				}
			},
		},
		{
			name: "BasicLit",
			node: &ast.BasicLit{Kind: token.INT, Value: "1", ValuePos: 42},
			checkFn: func(t *testing.T, node ast.Node) {
				x := node.(*ast.BasicLit)
				if x.ValuePos != token.NoPos {
					t.Errorf("BasicLit.ValuePos = %v, want NoPos", x.ValuePos)
				}
			},
		},
		{
			name: "StarExpr",
			node: &ast.StarExpr{Star: 42, X: &ast.Ident{Name: "T", NamePos: 43}},
			checkFn: func(t *testing.T, node ast.Node) {
				x := node.(*ast.StarExpr)
				if x.Star != token.NoPos {
					t.Errorf("StarExpr.Star = %v, want NoPos", x.Star)
				}
				if x.X.(*ast.Ident).NamePos != token.NoPos {
					t.Errorf("StarExpr.X.NamePos = %v, want NoPos", x.X.(*ast.Ident).NamePos)
				}
			},
		},
		{
			name: "ArrayType",
			node: &ast.ArrayType{Lbrack: 42, Elt: &ast.Ident{Name: "int", NamePos: 43}},
			checkFn: func(t *testing.T, node ast.Node) {
				x := node.(*ast.ArrayType)
				if x.Lbrack != token.NoPos {
					t.Errorf("ArrayType.Lbrack = %v, want NoPos", x.Lbrack)
				}
			},
		},
		{
			name: "MapType",
			node: &ast.MapType{Map: 42, Key: &ast.Ident{Name: "string"}, Value: &ast.Ident{Name: "int"}},
			checkFn: func(t *testing.T, node ast.Node) {
				x := node.(*ast.MapType)
				if x.Map != token.NoPos {
					t.Errorf("MapType.Map = %v, want NoPos", x.Map)
				}
			},
		},
		{
			name: "SliceExpr",
			node: &ast.SliceExpr{Lbrack: 42, Rbrack: 43, X: &ast.Ident{Name: "s"}},
			checkFn: func(t *testing.T, node ast.Node) {
				x := node.(*ast.SliceExpr)
				if x.Lbrack != token.NoPos {
					t.Errorf("SliceExpr.Lbrack = %v, want NoPos", x.Lbrack)
				}
				if x.Rbrack != token.NoPos {
					t.Errorf("SliceExpr.Rbrack = %v, want NoPos", x.Rbrack)
				}
			},
		},
		{
			name: "TypeAssertExpr",
			node: &ast.TypeAssertExpr{Lparen: 42, Rparen: 43, X: &ast.Ident{Name: "x"}, Type: &ast.Ident{Name: "T"}},
			checkFn: func(t *testing.T, node ast.Node) {
				x := node.(*ast.TypeAssertExpr)
				if x.Lparen != token.NoPos {
					t.Errorf("TypeAssertExpr.Lparen = %v, want NoPos", x.Lparen)
				}
				if x.Rparen != token.NoPos {
					t.Errorf("TypeAssertExpr.Rparen = %v, want NoPos", x.Rparen)
				}
			},
		},
		{
			name: "CallExpr",
			node: &ast.CallExpr{Lparen: 42, Rparen: 43, Fun: &ast.Ident{Name: "foo"}},
			checkFn: func(t *testing.T, node ast.Node) {
				x := node.(*ast.CallExpr)
				if x.Lparen != token.NoPos {
					t.Errorf("CallExpr.Lparen = %v, want NoPos", x.Lparen)
				}
				if x.Rparen != token.NoPos {
					t.Errorf("CallExpr.Rparen = %v, want NoPos", x.Rparen)
				}
			},
		},
		{
			name: "CompositeLit",
			node: &ast.CompositeLit{Lbrace: 42, Rbrace: 43, Type: &ast.Ident{Name: "T"}},
			checkFn: func(t *testing.T, node ast.Node) {
				x := node.(*ast.CompositeLit)
				if x.Lbrace != token.NoPos {
					t.Errorf("CompositeLit.Lbrace = %v, want NoPos", x.Lbrace)
				}
				if x.Rbrace != token.NoPos {
					t.Errorf("CompositeLit.Rbrace = %v, want NoPos", x.Rbrace)
				}
			},
		},
		{
			name: "KeyValueExpr",
			node: &ast.KeyValueExpr{Colon: 42, Key: &ast.Ident{Name: "k"}, Value: &ast.Ident{Name: "v"}},
			checkFn: func(t *testing.T, node ast.Node) {
				x := node.(*ast.KeyValueExpr)
				if x.Colon != token.NoPos {
					t.Errorf("KeyValueExpr.Colon = %v, want NoPos", x.Colon)
				}
			},
		},
		{
			name: "ParenExpr",
			node: &ast.ParenExpr{Lparen: 42, Rparen: 43, X: &ast.Ident{Name: "x"}},
			checkFn: func(t *testing.T, node ast.Node) {
				x := node.(*ast.ParenExpr)
				if x.Lparen != token.NoPos {
					t.Errorf("ParenExpr.Lparen = %v, want NoPos", x.Lparen)
				}
				if x.Rparen != token.NoPos {
					t.Errorf("ParenExpr.Rparen = %v, want NoPos", x.Rparen)
				}
			},
		},
		{
			name: "IndexExpr",
			node: &ast.IndexExpr{Lbrack: 42, Rbrack: 43, X: &ast.Ident{Name: "x"}, Index: &ast.Ident{Name: "i"}},
			checkFn: func(t *testing.T, node ast.Node) {
				x := node.(*ast.IndexExpr)
				if x.Lbrack != token.NoPos {
					t.Errorf("IndexExpr.Lbrack = %v, want NoPos", x.Lbrack)
				}
				if x.Rbrack != token.NoPos {
					t.Errorf("IndexExpr.Rbrack = %v, want NoPos", x.Rbrack)
				}
			},
		},
		{
			name: "ChanType",
			node: &ast.ChanType{Begin: 42, Arrow: 43, Dir: ast.SEND, Value: &ast.Ident{Name: "int"}},
			checkFn: func(t *testing.T, node ast.Node) {
				x := node.(*ast.ChanType)
				if x.Begin != token.NoPos {
					t.Errorf("ChanType.Begin = %v, want NoPos", x.Begin)
				}
				if x.Arrow != token.NoPos {
					t.Errorf("ChanType.Arrow = %v, want NoPos", x.Arrow)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearPositions(tt.node)
			tt.checkFn(t, tt.node)
		})
	}
}

func TestRenderNode(t *testing.T) {
	tests := []struct {
		name string
		node ast.Node
		want string
	}{
		{
			name: "simple identifier",
			node: &ast.Ident{Name: "foo"},
			want: "foo",
		},
		{
			name: "selector expression",
			node: &ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}, Sel: &ast.Ident{Name: "Type"}},
			want: "pkg.Type",
		},
		{
			name: "star expression",
			node: &ast.StarExpr{X: &ast.Ident{Name: "Foo"}},
			want: "*Foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderNode(tt.node)
			if err != nil {
				t.Fatalf("renderNode() error: %v", err)
			}
			if got != tt.want {
				t.Errorf("renderNode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderDecl(t *testing.T) {
	tests := []struct {
		name     string
		decl     ast.Decl
		contains string
	}{
		{
			name: "function declaration",
			decl: astFuncDecl(
				nil,
				"Foo",
				&ast.FieldList{},
				nil,
				&ast.BlockStmt{},
			),
			contains: "func Foo()",
		},
		{
			name: "var declaration",
			decl: astVarDecl(nil, "x", &ast.Ident{Name: "42"}),
			contains: "var x = 42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderDecl(tt.decl)
			if err != nil {
				t.Fatalf("renderDecl() error: %v", err)
			}
			if !strings.Contains(got, tt.contains) {
				t.Errorf("renderDecl() = %q, want to contain %q", got, tt.contains)
			}
		})
	}
}
