package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"modulewide/repository"
)

type UserService struct {
	annotation.Inject
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return UserService{repo: repo}
}
