package config

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

// Variable with a primitive literal (int) — the argument is *ast.BasicLit,
// not *ast.Ident or *ast.SelectorExpr. The detector emits a diagnostic error
// for unsupported expression types and cancels bootstrap generation.
var _ = annotation.Variable[variable.Default](42) // want "unsupported Variable argument: .*"
