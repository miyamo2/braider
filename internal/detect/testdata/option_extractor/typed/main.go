package testtyped

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type MyInterface interface {
	DoSomething()
}

type MyService struct {
	annotation.Injectable[inject.Typed[MyInterface]]
}

func (m *MyService) DoSomething() {}
