package service

import (
	"typed_inject/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserRepository struct { // want "missing constructor for UserRepository"
	annotation.Injectable[inject.Typed[domain.IRepository]]
}

func (r UserRepository) FindByID(id string) (string, error) {
	return id, nil
}
