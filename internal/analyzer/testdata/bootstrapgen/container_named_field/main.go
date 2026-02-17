package main

import (
	"container_named_field/service"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[struct { // want "bootstrap code is missing"
	Primary   *service.DB          `braider:"primaryDB"`
	Secondary *service.SecondaryDB `braider:"secondaryDB"`
}]](main)

func main() {}
