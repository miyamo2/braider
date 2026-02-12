package config

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

// Variable with a primitive literal (int) — the argument type resolves to
// *types.Basic "int" which is not a named package type. The detector still
// creates a candidate (TypeOf != nil) and the analyzer handles it gracefully.
var _ = annotation.Variable[variable.Default](42)
