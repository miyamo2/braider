package main

import (
	"container_provide_cross_type/container"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[container.AppContainer]](main) // want "bootstrap code is missing"

func main() {}
