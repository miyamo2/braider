package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"iface/domain"
)

type UserService struct {
	annotation.Inject
	repo domain.IUserRepository
}

func NewUserService(repo domain.IUserRepository) UserService {
	return UserService{repo: repo}
}
