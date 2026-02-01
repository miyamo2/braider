package repository

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"ambiguous/domain"
)

type UserRepositoryB struct {
	annotation.Inject
}

func NewUserRepositoryB() UserRepositoryB {
	return UserRepositoryB{}
}

func (r *UserRepositoryB) FindByID(id string) (domain.User, error) {
	return domain.User{}, nil
}
