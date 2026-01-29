package interfaces

import "github.com/miyamo2/braider/pkg/annotation"

// IRepository is an interface.
type IRepository interface {
	Get(id string) string
}

// UserRepository implements IRepository.
type UserRepository struct {
	annotation.Provide
}

// NewUserRepository is the constructor.
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// Get implements IRepository.
func (r *UserRepository) Get(id string) string {
	return "user"
}
