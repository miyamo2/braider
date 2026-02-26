package main

import (
	"error_container_tag_empty/service"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[struct {
	Svc *service.UserService `braider:""` // want `container field "Svc".*empty tag is not permitted`
}]](main)

func main() {}
