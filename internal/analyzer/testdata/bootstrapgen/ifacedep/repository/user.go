package repository

import (
	"example.com/ifacedep/domain"
	"github.com/miyamo2/braider/pkg/annotation"
)

type UserRepository struct {
	annotation.Provide
}

func NewUserRepository() UserRepository {
	return UserRepository{}
}

func (r UserRepository) FindByID(id string) (domain.User, error) {
	return domain.User{}, nil
}
