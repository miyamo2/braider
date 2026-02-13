package testvariabledefault

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

var _ = annotation.Variable[variable.Default](os.Stdout)
