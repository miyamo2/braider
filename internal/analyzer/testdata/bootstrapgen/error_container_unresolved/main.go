package main

import (
	"error_container_unresolved/service"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[struct {
	Unknown *service.UnknownService // want `container field "Unknown".*no matching dependency found`
}]](main)

func main() {}
