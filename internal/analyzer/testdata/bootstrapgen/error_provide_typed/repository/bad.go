package repository

import (
	"error_provide_typed/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type BadRepository struct{}

var _ = annotation.Provide[provide.Typed[domain.IRepository]](NewBadRepository) // want "option validation error: .*"

func NewBadRepository() *BadRepository {
	return &BadRepository{}
}
