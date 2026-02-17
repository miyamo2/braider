package config

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
	"container_mixed_option/domain"
)

var _ = annotation.Variable[variable.Typed[domain.Writer]](os.Stdout)
