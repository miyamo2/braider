package main

import (
	"error_container_ambiguous/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[struct {
	Repo domain.IUserRepository // want `container field "Repo".*multiple injectable structs implement interface`
}]](main)

func main() {}
