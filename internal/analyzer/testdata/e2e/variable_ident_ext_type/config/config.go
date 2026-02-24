package config

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

var Output = os.Stdout

var _ = annotation.Variable[variable.Default](Output)
