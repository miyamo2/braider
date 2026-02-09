package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type Repository interface {
	FindByID(id string) (string, error)
}

type UserRepository struct{}

func (r *UserRepository) FindByID(id string) (string, error) {
	return id, nil
}

var _ = annotation.Provide[provide.Typed[Repository]](NewUserRepository)

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

type UserService struct {
	annotation.Injectable[inject.Default]
	repo Repository
}

var _ = annotation.App(main)

func main() {}
