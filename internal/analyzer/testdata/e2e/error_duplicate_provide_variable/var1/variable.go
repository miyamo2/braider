package var1

import (
	"error_duplicate_provide_variable/types"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

type MainName struct{}

func (MainName) Name() string { return "main" }

var _ = annotation.Variable[variable.Named[MainName]](types.DefaultLogger)
