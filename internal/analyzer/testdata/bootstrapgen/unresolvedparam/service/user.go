package service

import (
	"example.com/unresolvedparam/repository"
	"github.com/miyamo2/braider/pkg/annotation"
)

type UserService struct {
	annotation.Inject
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return UserService{repo: repo}
}
