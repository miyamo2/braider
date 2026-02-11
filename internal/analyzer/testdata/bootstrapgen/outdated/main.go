package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"outdated/service"
)

var _ = annotation.App(main) // want "bootstrap code is outdated"

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
