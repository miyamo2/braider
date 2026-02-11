package repository

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// UserRepository is a Provide-annotated struct (local variable in bootstrap)
type UserRepository struct{}

var _ = annotation.Provide[provide.Default](NewUserRepository)

func NewUserRepository() UserRepository {
	return UserRepository{}
}
