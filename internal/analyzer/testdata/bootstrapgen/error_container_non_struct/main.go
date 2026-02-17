package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[string]](main) // want `failed to build dependency graph: app.Container detected but missing struct type parameter`

func main() {}
