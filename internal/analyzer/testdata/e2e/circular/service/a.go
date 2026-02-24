package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type ServiceA struct {
	annotation.Injectable[inject.Default]
	b *ServiceB
}

func NewServiceA(b *ServiceB) *ServiceA {
	return &ServiceA{b: b}
}
