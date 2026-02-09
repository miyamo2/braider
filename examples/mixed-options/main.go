package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type ServiceName struct{}

func (ServiceName) Name() string { return "service" }

type Repository interface {
	FindByID(id string) (string, error)
}

type MixedService struct {
	annotation.Injectable[interface {
		inject.Typed[Repository]
		inject.Named[ServiceName]
	}]
}

func (s *MixedService) FindByID(id string) (string, error) {
	return id, nil
}

var _ = annotation.App(main)

func main() {}
