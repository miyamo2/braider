package main

import (
	"example.com/depblank/service"
	"github.com/miyamo2/braider/pkg/annotation"
	app "github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Default](main) // want "bootstrap code is outdated"

func main() {
	// _ = dependency already exists, should not add another one
	_ = dependency
}

// braider:hash:a282cc0f1184f9aa
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
