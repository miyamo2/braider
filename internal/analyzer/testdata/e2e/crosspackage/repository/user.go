package repository

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type UserRepository struct{}

var _ = annotation.Provide[provide.Default](NewUserRepository)

func NewUserRepository() UserRepository {
	return UserRepository{}
}
