package inject_test

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

func ExampleOption_custom() {
	type IService interface{}

	type Repository interface{}

	type Service struct {
		annotation.Injectable[interface {
			inject.Typed[IService]
			inject.WithoutConstructor
		}]
		repository Repository
	}
}
