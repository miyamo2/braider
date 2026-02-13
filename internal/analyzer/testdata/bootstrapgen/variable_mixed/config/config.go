package config

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
	"variable_mixed/domain"
)

var _ = annotation.Variable[variable.Typed[domain.Writer]](os.Stdout)
