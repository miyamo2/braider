package repository

import (
	"database/sql"

	"github.com/miyamo2/braider/pkg/annotation"
)

type UserRepository struct {
	annotation.Inject
	db *sql.DB
}

// NewUserRepository is a constructor for UserRepository.
func NewUserRepository(db *sql.DB) UserRepository {
	return UserRepository{db: db}
}
