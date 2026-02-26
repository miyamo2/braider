package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	app "github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Default](main) // want `duplicate dependency name "primary" for type error_variable_duplicate_name/types\.Config`

func main() {}
