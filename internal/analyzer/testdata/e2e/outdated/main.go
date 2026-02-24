package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	app "github.com/miyamo2/braider/pkg/annotation/app"
	"outdated/service"
)

var _ = annotation.App[app.Default](main) // want "bootstrap code is outdated"

func main() {
	_ = dependency
}

// braider:hash:old12345
var dependency = func() struct {
	userService *service.UserService
} {
	userService := service.NewUserService()
	return struct {
		userService *service.UserService
	}{
		userService: userService,
	}
}()
