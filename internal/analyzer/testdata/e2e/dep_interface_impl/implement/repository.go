package implement

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// UserRepository implements IRepository.
type UserRepository struct{}

var _ = annotation.Provide[provide.Default](NewUserRepository)

// NewUserRepository is the constructor.
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// Get implements IRepository.
func (r *UserRepository) Get(id string) string {
	return "user"
}
