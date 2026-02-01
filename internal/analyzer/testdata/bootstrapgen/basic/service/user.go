package service

import "github.com/miyamo2/braider/pkg/annotation"

type UserService struct {
	annotation.Inject
}

func NewUserService() UserService {
	return UserService{}
}
