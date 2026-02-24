package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type UserRepository struct{}

var _ = annotation.Provide[provide.Default](NewUserRepository)

func NewUserRepository() UserRepository {
	return UserRepository{}
}

type UserService struct {
	annotation.Injectable[inject.Default]
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}
