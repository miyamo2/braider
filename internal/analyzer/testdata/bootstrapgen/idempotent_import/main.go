package main

import "github.com/miyamo2/braider/pkg/annotation"
import app "github.com/miyamo2/braider/pkg/annotation/app"
import "idempotent_import/service"

var _ = annotation.App[app.Default](main) // want "bootstrap code is outdated"

func main() {
	_ = dependency
}

// braider:hash:old_hash_value
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
