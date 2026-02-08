package repository

import (
	"example.com/ambiguousiface/domain"
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type UserRepositoryA struct{}

var _ = annotation.Provide[provide.Default](NewUserRepositoryA)

func NewUserRepositoryA() UserRepositoryA {
	return UserRepositoryA{}
}

func (r UserRepositoryA) FindByID(id string) (domain.User, error) {
	return domain.User{}, nil
}
