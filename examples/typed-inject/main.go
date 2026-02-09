package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type Repository interface {
	FindByID(id string) (string, error)
}

type UserRepository struct {
	annotation.Injectable[inject.Typed[Repository]]
}

func (r *UserRepository) FindByID(id string) (string, error) {
	return id, nil
}

var _ = annotation.App(main)

func main() {}
