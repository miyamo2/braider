package config

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

// Variable with a primitive literal (int) — the argument is *ast.BasicLit,
// not *ast.Ident or *ast.SelectorExpr. The detector silently skips unsupported
// expression types, so no Variable candidate is registered.
var _ = annotation.Variable[variable.Default](42)
