package main

import (
	"container_transitive/service"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[struct { // want "bootstrap code is missing"
	Svc *service.UserService
}]](main)

func main() {}
