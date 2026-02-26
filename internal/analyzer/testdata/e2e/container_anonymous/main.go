package main

import (
	"container_anonymous/repository"
	"container_anonymous/service"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[struct { // want "bootstrap code is missing"
	Repo *repository.UserRepository
	Svc  *service.UserService
}]](main)

func main() {}
