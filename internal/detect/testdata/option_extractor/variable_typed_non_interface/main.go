package testvariabletypednoninterface

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

type SomeStruct struct {
	Value string
}

var _ = annotation.Variable[variable.Typed[SomeStruct]](os.Stdout)
