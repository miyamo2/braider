package main

import (
	"container_iface_field/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[struct { // want "bootstrap code is missing"
	Repo domain.IUserRepository
}]](main)

func main() {}
