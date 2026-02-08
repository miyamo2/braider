package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type IService interface {
	DoWork()
}

type BadService struct { // want "option validation error: .*"
	annotation.Injectable[inject.Typed[IService]]
	data string
}

func NewBadService(data string) *BadService {
	return &BadService{data: data}
}
