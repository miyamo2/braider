package service

import "github.com/miyamo2/braider/pkg/annotation"

type ServiceB struct {
	annotation.Inject
	a *ServiceA
}

func NewServiceB(a *ServiceA) *ServiceB {
	return &ServiceB{a: a}
}
