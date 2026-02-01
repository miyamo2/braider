package basic

import "github.com/miyamo2/braider/pkg/annotation"

// UserService is an Inject struct with a constructor.
type UserService struct {
	annotation.Inject
	repo *UserRepository
}

// NewUserService is the constructor for UserService.
func NewUserService(repo *UserRepository) *UserService {
	return &UserService{repo: repo}
}
