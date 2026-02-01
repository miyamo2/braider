package repository

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"ambiguous/domain"
)

type UserRepositoryA struct {
	annotation.Inject
}

func NewUserRepositoryA() UserRepositoryA {
	return UserRepositoryA{}
}

func (r *UserRepositoryA) FindByID(id string) (domain.User, error) {
	return domain.User{}, nil
}
