package main

import (
	"error_container_tag_exclude/service"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[struct {
	Svc *service.UserService `braider:"-"` // want `container field "Svc".*braider:"-" tag is not permitted`
}]](main)

func main() {}
