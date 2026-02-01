package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main) // want "bootstrap code is outdated"

func main() {
	// _ = dependency already exists, should not add another one
	_ = dependency
}

// braider:hash:a282cc0f1184f9aa
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
