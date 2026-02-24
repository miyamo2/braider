package reg

import (
	v1config "variable_pkg_collision/v1/config"
	v2config "variable_pkg_collision/v2/config"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

var _ = annotation.Variable[variable.Default](v1config.DefaultAlpha)
var _ = annotation.Variable[variable.Default](v2config.DefaultBeta)
