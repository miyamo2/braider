package repository

import (
	"ambiguous/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserRepositoryB struct {
	annotation.Injectable[inject.Default]
}

func NewUserRepositoryB() UserRepositoryB {
	return UserRepositoryB{}
}

func (r *UserRepositoryB) FindByID(id string) (domain.User, error) {
	return domain.User{}, nil
}
