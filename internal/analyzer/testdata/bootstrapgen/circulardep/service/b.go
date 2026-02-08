package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type ServiceB struct {
	annotation.Injectable[inject.Default]
	a ServiceA
}

func NewServiceB(a ServiceA) ServiceB {
	return ServiceB{a: a}
}
