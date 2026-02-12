package config

import (
	"os"

	"error_variable_typed/domain"
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

// os.Stdout (*os.File) does not implement domain.IRepository
var _ = annotation.Variable[variable.Typed[domain.IRepository]](os.Stdout) // want "option validation error: .*"
