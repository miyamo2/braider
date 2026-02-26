package repository

import (
	"container_iface_field/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserRepository struct { // want "missing constructor for UserRepository"
	annotation.Injectable[inject.Default]
}

func (r *UserRepository) FindByID(id string) (domain.User, error) {
	return domain.User{}, nil
}
