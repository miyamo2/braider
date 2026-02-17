package detect

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestDescribeExprType(t *testing.T) {
	tests := []struct {
		name     string
		expr     ast.Expr
		expected string
	}{
		{
			name:     "BasicLit",
			expr:     &ast.BasicLit{Kind: token.INT, Value: "42"},
			expected: "literal value",
		},
		{
			name:     "CompositeLit",
			expr:     &ast.CompositeLit{},
			expected: "composite literal",
		},
		{
			name:     "CallExpr",
			expr:     &ast.CallExpr{Fun: &ast.Ident{Name: "fn"}},
			expected: "function call",
		},
		{
			name:     "UnaryExpr",
			expr:     &ast.UnaryExpr{Op: token.NOT, X: &ast.Ident{Name: "x"}},
			expected: "unary expression",
		},
		{
			name:     "BinaryExpr",
			expr:     &ast.BinaryExpr{X: &ast.Ident{Name: "a"}, Op: token.ADD, Y: &ast.Ident{Name: "b"}},
			expected: "binary expression",
		},
		{
			name:     "FuncLit",
			expr:     &ast.FuncLit{Type: &ast.FuncType{}},
			expected: "function literal",
		},
		{
			name:     "StarExpr",
			expr:     &ast.StarExpr{X: &ast.Ident{Name: "x"}},
			expected: "pointer dereference",
		},
		{
			name:     "default case (ParenExpr)",
			expr:     &ast.ParenExpr{X: &ast.Ident{Name: "x"}},
			expected: "expression type *ast.ParenExpr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := describeExprType(tt.expr)
			if result != tt.expected {
				t.Errorf("describeExprType(%T) = %q, want %q", tt.expr, result, tt.expected)
			}
		})
	}
}
