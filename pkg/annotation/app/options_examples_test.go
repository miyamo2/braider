package app_test

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

func ExampleDefault() {
	var _ = annotation.App[app.Default](main)
}

func ExampleContainer() {
	var _ = annotation.App[app.Container[struct {
		someDependency string `braider:"someDependency"`
	}]](main)
}

var main func()
