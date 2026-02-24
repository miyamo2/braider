package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type Service struct {
	annotation.Injectable[inject.Default]
}

func NewService() *Service {
	return &Service{}
}
