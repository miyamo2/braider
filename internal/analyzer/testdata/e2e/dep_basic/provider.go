package basic

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// UserRepository is a Provide struct with a constructor.
type UserRepository struct{}

var _ = annotation.Provide[provide.Default](NewUserRepository)

// NewUserRepository is the constructor for UserRepository.
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}
