package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

var dynamicName = "dynamic"

type BadNamer struct{}

func (BadNamer) Name() string { return dynamicName }

type BadNamedService struct { // want "option validation error: .*"
	annotation.Injectable[inject.Named[BadNamer]]
}

func NewBadNamedService() *BadNamedService {
	return &BadNamedService{}
}
