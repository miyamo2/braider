package repository

import "github.com/miyamo2/braider/pkg/annotation"

// UserRepository is a Provide-annotated struct (local variable in bootstrap)
type UserRepository struct {
	annotation.Provide
}

func NewUserRepository() UserRepository {
	return UserRepository{}
}
