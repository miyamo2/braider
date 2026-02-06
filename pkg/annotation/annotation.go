// Package annotation provides marker types and functions for braider's
// dependency injection code generation.
//
// This package defines annotations that instruct the braider analyzer
// to generate constructors, wiring code, and main function bootstrapping.
package annotation

import (
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// Injectable marks a struct as a dependency to be injected.
//
// When a struct is annotated with Injectable, braider generates a constructor
// function for the struct and registers it in the provider registry.
// The constructor function is expected to accept all dependencies as parameters.
//
// Example:
//
//	package service
//
//	import (
//		"github.com/miyamo2/braider/pkg/annotation"
//		"github.com/miyamo2/braider/pkg/annotation/inject"
//	)
//
//	type MyService interface {
//	    DoSomething() error
//	}
//
//	type myService struct {
//	    annotation.Injectable[inject.Default]
//	    myRepository MyRepository
//	}
//
//	func (s *myService) DoSomething() error { ... }
//
// Example generated constructor function:
//
//	package service
//
//	import (
//		"github.com/miyamo2/braider/pkg/annotation"
//		"github.com/miyamo2/braider/pkg/annotation/inject"
//	)
//
//	type MyService interface {
//	    DoSomething() error
//	}
//
//	type myService struct {
//	    annotation.Injectable[inject.Default]
//	    myRepository MyRepository
//	}
//
//	func (s *myService) DoSomething() error { ... }
//
//	func NewMyService(myRepository MyRepository) *myService {
//	    return &myService{
//	        myRepository: myRepository,
//	    }
//	}
type Injectable[T inject.Option] interface {
	isInjectable()
	option() T
}

type Provider[T provide.Option] interface {
	isProvider()
	option() T
}

type provider[T provide.Option] struct{}

func (p provider[T]) isProvider() {}

func (p provider[T]) option() T {
	var zero T
	return zero
}

// Provide marks a function as a dependency provider.
//
// When a function is annotated with Provide, braider registers it in the
// provider registry and generates a local variable in the bootstrap IIFE.
// The function is expected to return an instance of the provided dependency.
//
// Example:
//
//	package repository
//
//	import (
//		"github.com/miyamo2/braider/pkg/annotation"
//		"github.com/miyamo2/braider/pkg/annotation/provide"
//	)
//
//	var _ MyRepository = (*myRepository)(nil)
//
//	type MyRepository interface {
//	    GetData(id string) (string, error)
//	}
//
//	type myRepository struct{}
//
//	func (r *myRepository) GetData(id string) (string, error) { ... }
//
//	var _ = annotation.Provide[provide.Typed[MyRepository]](NewMyRepository)
//
//	func NewMyRepository() *myRepository {
//	    return &myRepository{}
//	}
func Provide[T provide.Option](providerFunc any) Provider[T] {
	_ = providerFunc
	return provider[T]{}
}

// App marks a struct as the top-level dependency injection target.
//
// When a struct is annotated with App, braider generates the main function
// to bootstrap the application by resolving and injecting all dependencies.
//
// Example:
//
//	package main
//
//	import "github.com/miyamo2/braider/pkg/annotation"
//
//	var _ = annotation.App(main)
//
//	func main() {}
//
// Example generated main function:
//
//	package main
//
//	import "github.com/miyamo2/braider/pkg/annotation"
//
//	var _ = annotation.App(main)
//
//	func main() {
//		_ = dependency
//	}
//
//	var dependency = func() struct {
//		myRepository MyRepository
//		myService    MyService
//		myHandler    MyHandler
//	} {
//		myRepository := NewMyRepository()
//		myService := NewMyService(myRepository)
//		myHandler := NewMyHandler(myService)
//		return struct {
//			myRepository MyRepository
//			myService    MyService
//			myHandler    MyHandler
//		} {
//			myRepository: myRepository,
//			myService:    myService,
//			myHandler:    myHandler,
//		}
//	}
func App(_ func()) struct{} {
	return struct{}{}
}
