package service

import (
	"provide_typed/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserService struct {
	annotation.Injectable[inject.Default]
	repo domain.IRepository
}

func NewUserService(repo domain.IRepository) *UserService {
	return &UserService{repo: repo}
}
