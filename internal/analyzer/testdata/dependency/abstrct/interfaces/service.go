package interfaces

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// UserService uses IRepository.
type UserService struct {
	annotation.Injectable[inject.Default]
	repo IRepository
}

// NewUserService is the constructor.
func NewUserService(repo IRepository) *UserService {
	return &UserService{repo: repo}
}
