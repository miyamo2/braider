package repository

import (
	"provide_typed/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

var _ domain.IRepository = (*UserRepository)(nil)

type UserRepository struct{}

func (r *UserRepository) FindByID(id string) (string, error) {
	return id, nil
}

var _ = annotation.Provide[provide.Typed[domain.IRepository]](NewUserRepository)

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}
