package repository

import (
	"database/sql"

	"github.com/miyamo2/braider/pkg/annotation"
)

type UserRepository struct {
	annotation.Inject
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return UserRepository{db: db}
}
