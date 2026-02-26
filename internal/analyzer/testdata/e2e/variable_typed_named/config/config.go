package config

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
	"variable_typed_named/domain"
)

type StdoutName struct{}

func (StdoutName) Name() string { return "stdout" }

var _ = annotation.Variable[interface {
	variable.Typed[domain.Writer]
	variable.Named[StdoutName]
}](os.Stdout)
