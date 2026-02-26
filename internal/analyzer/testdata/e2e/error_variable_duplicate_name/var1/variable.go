package var1

import (
	"error_variable_duplicate_name/types"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

type PrimaryName struct{}

func (PrimaryName) Name() string { return "primary" }

var _ = annotation.Variable[variable.Named[PrimaryName]](types.DefaultConfig)
