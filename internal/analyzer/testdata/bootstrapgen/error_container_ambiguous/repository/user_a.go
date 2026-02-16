package repository

import (
	"error_container_ambiguous/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserRepositoryA struct {
	annotation.Injectable[inject.Default]
}

func NewUserRepositoryA() *UserRepositoryA {
	return &UserRepositoryA{}
}

func (r *UserRepositoryA) FindByID(id string) (domain.User, error) {
	return domain.User{}, nil
}
