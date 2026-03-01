package app_test

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

func ExampleDefault() {
	var _ = annotation.App[app.Default](main)
}

func ExampleContainer() {
	type Service struct{}
	var _ = annotation.App[app.Container[struct {
		Svc *Service
	}]](main)
}

var main func()
