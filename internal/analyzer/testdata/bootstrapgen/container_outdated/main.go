package main

import (
	"container_outdated/service"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[struct { // want "bootstrap code is outdated"
	Svc *service.UserService
}]](main)

func main() {
	_ = dependency
}

// braider:hash:0000000000000000
var dependency = func() struct {
	Svc *service.UserService
} {
	userService := service.NewUserService()
	return struct {
		Svc *service.UserService
	}{
		Svc: userService,
	}
}()
