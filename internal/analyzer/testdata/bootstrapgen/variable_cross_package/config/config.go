package config

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

type Config struct {
	Debug bool
}

var DefaultConfig = &Config{Debug: true}

var _ = annotation.Variable[variable.Default](DefaultConfig)
