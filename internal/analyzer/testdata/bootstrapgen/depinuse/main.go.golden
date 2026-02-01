package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)

func main() {
	// dependency is already used, no _ = dependency needed
	dependency.userService.Run()
}

// braider:hash:e0c021f7f5273495
var dependency = func() struct {
	userService UserService
} {
	userService := NewUserService()
	return struct {
		userService UserService
	}{
		userService: userService,
	}
}()

type UserService struct {
	annotation.Inject
}

func NewUserService() UserService {
	return UserService{}
}

func (s UserService) Run() {}
