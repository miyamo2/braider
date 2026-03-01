package app

import "github.com/miyamo2/braider/internal/annotation"

// Option configures annotation.App behavior.
//
// Your custom options(mixed-in options) must implement this interface.
type Option interface {
	annotation.AppOption
}

// Default configures annotation.App to the default behavior.
type Default interface {
	Option
	annotation.AppDefault
}

// Container configures annotation.App to use a user-defined container.
// The analyzer generates a bootstrap function that returns the container instance provided by type parameter.
//
// Container fields are matched to registered dependencies by type.
// Use braider:"name" struct tags to match named dependencies.
//
// Example input (service is registered elsewhere with Injectable[inject.Default]):
//
//	var _ = annotation.App[app.Container[struct {
//		Svc *service.UserService
//	}]](main)
//
//	func main() {}
//
// Example generated code:
//
//	func main() {
//		_ = dependency
//	}
//
//	var dependency = func() struct {
//		Svc *service.UserService
//	} {
//		userService := service.NewUserService()
//		return struct {
//			Svc *service.UserService
//		}{
//			Svc: userService,
//		}
//	}()
type Container[T any] interface {
	Option
	annotation.AppContainer
	definedContainerParam() T
}
