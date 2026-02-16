package main

import (
	"container_idempotent/service"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[struct {
	Svc *service.UserService
}]](main)

func main() {
	_ = dependency
}

// braider:hash:0d049866ac814b82
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
