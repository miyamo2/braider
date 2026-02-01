package repository

import (
	"ambiguous/domain"

	"github.com/miyamo2/braider/pkg/annotation"
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
