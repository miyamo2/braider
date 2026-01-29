package basic

import "github.com/miyamo2/braider/pkg/annotation"

// UserRepository is a Provide struct with a constructor.
type UserRepository struct {
	annotation.Provide
}

// NewUserRepository is the constructor for UserRepository.
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}
