package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
)

type ServiceA struct {
	annotation.Inject
	b ServiceB
}

func NewServiceA(b ServiceB) ServiceA {
	return ServiceA{b: b}
}
