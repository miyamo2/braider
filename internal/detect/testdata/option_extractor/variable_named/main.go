package testvariablenamed

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

type StdoutName struct{}

func (StdoutName) Name() string { return "stdout" }

var _ = annotation.Variable[variable.Named[StdoutName]](os.Stdout)
