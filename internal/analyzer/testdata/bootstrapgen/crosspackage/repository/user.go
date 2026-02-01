package repository

import "github.com/miyamo2/braider/pkg/annotation"

type UserRepository struct {
	annotation.Provide
}

func NewUserRepository() UserRepository {
	return UserRepository{}
}
