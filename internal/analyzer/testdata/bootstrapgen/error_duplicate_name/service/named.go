package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type PrimaryName struct{}

func (PrimaryName) Name() string { return "primary" }

type NamedService struct {
	annotation.Injectable[inject.Named[PrimaryName]]
}

func NewNamedService() *NamedService {
	return &NamedService{}
}
