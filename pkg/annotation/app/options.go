// Package app provides option interfaces for configuring
// annotation.App behavior in braider's dependency injection system.
//
// # Optional Declaration
//
// When the analyzed scope contains exactly one main package (a package declared
// as "package main" with a top-level "func main") and no explicit
// annotation.App declaration is present anywhere in scope, braider infers that
// main package as the application entry point and generates bootstrap wiring as
// if annotation.App[app.Default](main) had been declared in it. Inference
// always uses app.Default semantics; app.Container[T] cannot be inferred
// because the container type cannot be guessed.
//
// An explicit annotation.App declaration always takes precedence over
// inference. If any explicit annotation.App is present in the analyzed scope,
// no inference is performed for any package.
//
// When the analyzed scope contains more than one main package and no explicit
// annotation.App is declared, braider emits a diagnostic identifying the
// candidate main packages and asks the developer to add an explicit
// annotation.App[T](main) declaration to one of them instead of guessing.
package app

import "github.com/miyamo2/braider/internal/annotation"

// Option configures annotation.App behavior.
//
// Your custom options(mixed-in options) must implement this interface.
type Option interface {
	annotation.AppOption
}

// Default configures annotation.App to the default behavior.
//
// Default is also the option used implicitly when annotation.App is omitted in
// a project with exactly one main package; see the package documentation for
// the inference rule.
type Default interface {
	Option
	annotation.AppDefault
}

// Container configures annotation.App to use a user-defined container.
// The analyzer generates a bootstrap IIFE that returns the container instance provided by type parameter.
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
