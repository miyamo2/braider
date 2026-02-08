package repository

import (
	"iface/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserRepository struct {
	annotation.Injectable[inject.Default]
}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) FindByID(id string) (domain.User, error) {
	return domain.User{}, nil
}
