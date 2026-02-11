package service

import (
	"crossiface/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserService struct {
	annotation.Injectable[inject.Default]
	repo domain.IUserRepository
}

func NewUserService(repo domain.IUserRepository) *UserService {
	return &UserService{repo: repo}
}
