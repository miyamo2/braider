package app_test

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

// ExampleDefault demonstrates the app.Default option.
// When used with annotation.App, the analyzer generates a bootstrap function
// that returns an anonymous struct with all dependencies as fields.
func ExampleDefault() {
	var _ = annotation.App[app.Default](main)
}

// ExampleContainer demonstrates the app.Container option.
// When used with annotation.App, the analyzer generates a bootstrap function
// that returns an instance of the user-defined container struct type.
// Container fields are matched to registered dependencies by type or braider struct tags.
func ExampleContainer() {
	type Service struct{}
	var _ = annotation.App[app.Container[struct {
		Svc *Service `braider:"someDependency"`
	}]](main)
}

var main func()
