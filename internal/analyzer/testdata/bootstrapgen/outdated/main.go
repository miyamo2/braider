package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main) // want "bootstrap code is outdated"

func main() {
	_ = dependency
}

// braider:hash:old12345
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

type OrderService struct {
	annotation.Inject
}

func NewOrderService() OrderService {
	return OrderService{}
}
