package repository

import (
	"example.com/ambiguousiface/domain"
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type UserRepositoryB struct{}

var _ = annotation.Provide[provide.Default](NewUserRepositoryB)

func NewUserRepositoryB() UserRepositoryB {
	return UserRepositoryB{}
}

func (r UserRepositoryB) FindByID(id string) (domain.User, error) {
	return domain.User{}, nil
}
