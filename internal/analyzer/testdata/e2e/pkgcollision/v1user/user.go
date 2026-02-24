package user

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type Repository struct{}

var _ = annotation.Provide[provide.Default](NewRepository)

func NewRepository() Repository {
	return Repository{}
}

type Service struct { // want "missing constructor for Service"
	annotation.Injectable[inject.Default]
	repo Repository
}
