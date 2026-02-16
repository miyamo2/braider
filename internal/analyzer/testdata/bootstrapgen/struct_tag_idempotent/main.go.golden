// Task 7.1: Idempotent behavior test for braider struct tags.
// Verifies that when bootstrap code with correct hash already exists,
// re-running the analyzer produces NO diagnostic (hash stability).
// The braider:"-" tag on AppService.debug excludes that field from DI,
// affecting both constructor params and hash (fewer Dependencies in graph node).
package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	app "github.com/miyamo2/braider/pkg/annotation/app"
	"struct_tag_idempotent/repository"
	"struct_tag_idempotent/service"
)

var _ = annotation.App[app.Default](main)

func main() {
	_ = dependency
}

// braider:hash:61e2c8cf48b5572b
var dependency = func() struct {
	userRepository *repository.UserRepository
	appService     *service.AppService
} {
	userRepository := repository.NewUserRepository()
	appService := service.NewAppService(userRepository)
	return struct {
		userRepository *repository.UserRepository
		appService     *service.AppService
	}{
		userRepository: userRepository,
		appService:     appService,
	}
}()
