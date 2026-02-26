package repository

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// UserRepository is a simple Injectable with braider:"-" excluded field.
type UserRepository struct {
	annotation.Injectable[inject.Default]
}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}
