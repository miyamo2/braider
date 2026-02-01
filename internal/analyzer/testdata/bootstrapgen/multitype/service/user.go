package service

import (
	"example.com/multitype/repository"
	"github.com/miyamo2/braider/pkg/annotation"
)

// UserService is an Inject-annotated struct (field in dependency struct)
type UserService struct {
	annotation.Inject
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return UserService{repo: repo}
}
