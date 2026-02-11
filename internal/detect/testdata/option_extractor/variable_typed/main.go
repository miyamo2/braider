package testvariabletyped

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

type MyWriter interface {
	Write(p []byte) (n int, err error)
}

var _ = annotation.Variable[variable.Typed[MyWriter]](os.Stdout)
