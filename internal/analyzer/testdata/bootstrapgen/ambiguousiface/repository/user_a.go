package repository

import (
	"example.com/ambiguousiface/domain"
	"github.com/miyamo2/braider/pkg/annotation"
)

type UserRepositoryA struct {
	annotation.Provide
}

func NewUserRepositoryA() UserRepositoryA {
	return UserRepositoryA{}
}

func (r UserRepositoryA) FindByID(id string) (domain.User, error) {
	return domain.User{}, nil
}
