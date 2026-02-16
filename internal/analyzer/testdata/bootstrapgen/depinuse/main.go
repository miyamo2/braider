package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	app "github.com/miyamo2/braider/pkg/annotation/app"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

var _ = annotation.App[app.Default](main)

func main() {
	// dependency is already used, no _ = dependency needed
	dependency.userService.Run()
}

// braider:hash:8764fec6541913b1
var dependency = func() struct {
	userService *UserService
} {
	userService := NewUserService()
	return struct {
		userService *UserService
	}{
		userService: userService,
	}
}()

type UserService struct {
	annotation.Injectable[inject.Default]
}

func NewUserService() *UserService {
	return &UserService{}
}

func (s UserService) Run() {}
