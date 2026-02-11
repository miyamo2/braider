package testnamed

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type DatabaseName struct{}

func (DatabaseName) Name() string { return "database" }

type MyService struct {
	annotation.Injectable[inject.Named[DatabaseName]]
}
