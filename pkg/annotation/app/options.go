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
// Example:
//
//	package main
//
//	import (
//		"net/http"
//
//		"github.com/miyamo2/braider/pkg/annotation"
//	)
//
//	var _ = annotation.App[DefinedContainer[struct {
//		handler http.Handler `braider:"handler"`
//	}]](main)
//
//	func() main() {}
//
// Example generated main function:
//
//	package main
//
//	import (
//		"net/http"
//
//		"github.com/miyamo2/braider/pkg/annotation"
//	)
//
//	var _ = annotation.App[app.DefinedContainer[struct {
//		handler http.Handler `braider:"handler"`
//	}]](main)
//
//	func main() {
//		_ = dependency
//	}
//
//	var dependency = func() struct {
//		handler http.Handler `braider:"handler"`
//	} {
//		handler := http.NewServeMux()
//		return struct {
//			handler http.Handler `braider:"handler"`
//		}{
//			handler: handler,
//		}
//	}()
type Container[T any] interface {
	Option
	annotation.AppDefinedContainer
	definedContainerParam() T
}
