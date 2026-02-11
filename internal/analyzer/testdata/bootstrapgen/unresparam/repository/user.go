package repository

import (
	"database/sql"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserRepository struct {
	annotation.Injectable[inject.Default]
	db *sql.DB
}

// NewUserRepository is a constructor for UserRepository.
func NewUserRepository(db *sql.DB) UserRepository {
	return UserRepository{db: db}
}
