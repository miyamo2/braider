package basic

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// UserService is an Inject struct with a constructor.
type UserService struct {
	annotation.Injectable[inject.Default]
	repo *UserRepository
}

// NewUserService is the constructor for UserService.
func NewUserService(repo *UserRepository) *UserService {
	return &UserService{repo: repo}
}
