package config

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

var dynamicName = "dynamic"

type BadNamer struct{}

func (BadNamer) Name() string { return dynamicName }

var _ = annotation.Variable[variable.Named[BadNamer]](os.Stdout) // want "option validation error: .*"
