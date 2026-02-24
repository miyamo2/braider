package service

import (
	"example.com/multitype/repository"
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// UserService is an Inject-annotated struct (field in dependency struct)
type UserService struct {
	annotation.Injectable[inject.Default]
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}
