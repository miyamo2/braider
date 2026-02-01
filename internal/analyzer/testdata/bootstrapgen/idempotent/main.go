package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)

func main() {
	_ = dependency
}

// braider:hash:4edf020376a292d1
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
