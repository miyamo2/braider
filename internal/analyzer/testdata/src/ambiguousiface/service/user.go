package service

import (
	"example.com/ambiguousiface/domain"
	"github.com/miyamo2/braider/pkg/annotation"
)

type UserService struct {
	annotation.Inject
	repo domain.IUserRepository
}

func NewUserService(repo domain.IUserRepository) UserService {
	return UserService{repo: repo}
}
