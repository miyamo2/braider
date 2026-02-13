package testvariableifaceerror

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

type CustomInterface interface {
	DoSomething()
}

// os.Stdout is *os.File which does NOT implement CustomInterface (missing DoSomething)
var _ = annotation.Variable[variable.Typed[CustomInterface]](os.Stdout)
