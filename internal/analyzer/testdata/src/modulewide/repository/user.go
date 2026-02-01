package repository

import "github.com/miyamo2/braider/pkg/annotation"

type UserRepository struct {
	annotation.Inject
}

func NewUserRepository() UserRepository {
	return UserRepository{}
}
