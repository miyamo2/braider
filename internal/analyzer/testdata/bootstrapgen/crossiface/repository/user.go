package repository

import (
	"crossiface/domain"

	"github.com/miyamo2/braider/pkg/annotation"
)

type UserRepository struct {
	annotation.Inject
}

func NewUserRepository() UserRepository {
	return UserRepository{}
}

func (r *UserRepository) FindByID(id string) (domain.User, error) {
	return domain.User{}, nil
}
