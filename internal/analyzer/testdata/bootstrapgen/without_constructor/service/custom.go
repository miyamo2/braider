package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type CustomService struct {
	annotation.Injectable[inject.WithoutConstructor]
}

func NewCustomService() *CustomService {
	return &CustomService{}
}
