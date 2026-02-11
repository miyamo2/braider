package testifaceerror

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

// Note: Missing DoSomething() implementation - should cause validation error
