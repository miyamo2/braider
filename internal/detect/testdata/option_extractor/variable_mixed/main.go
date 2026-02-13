package testvariablemixed

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

type MyWriter interface {
	Write(p []byte) (n int, err error)
}

type StdoutName struct{}

func (StdoutName) Name() string { return "stdout" }

var _ = annotation.Variable[interface {
	variable.Typed[MyWriter]
	variable.Named[StdoutName]
}](os.Stdout)
