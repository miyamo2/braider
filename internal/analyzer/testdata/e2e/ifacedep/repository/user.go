package repository

import (
	"example.com/ifacedep/domain"
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type UserRepository struct{}

var _ = annotation.Provide[provide.Default](NewUserRepository)

func NewUserRepository() UserRepository {
	return UserRepository{}
}

func (r UserRepository) FindByID(id string) (domain.User, error) {
	return domain.User{}, nil
}
