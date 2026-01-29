package interfaces

import "github.com/miyamo2/braider/pkg/annotation"

// UserService uses IRepository.
type UserService struct {
	annotation.Inject
	repo IRepository
}

// NewUserService is the constructor.
func NewUserService(repo IRepository) *UserService {
	return &UserService{repo: repo}
}
