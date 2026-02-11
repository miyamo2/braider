package inject_test

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"github.com/miyamo2/braider/pkg/annotation/namer"
)

// ExampleDefault demonstrates the inject.Default option.
// When used with annotation.Injectable, the analyzer generates a constructor
// that returns a pointer to the concrete struct type (*StructType).
func ExampleDefault() {
	type UserService struct {
		annotation.Injectable[inject.Default]
	}
	_ = UserService{}
}

// ExampleTyped demonstrates the inject.Typed option.
// When used with annotation.Injectable, the analyzer registers the dependency
// as the specified interface type instead of the concrete struct type.
// The concrete struct must implement the interface.
func ExampleTyped() {
	type Repository interface {
		FindByID(id string) (string, error)
	}

	type UserRepository struct {
		annotation.Injectable[inject.Typed[Repository]]
	}
	_ = UserRepository{}
}

// ExampleNamed demonstrates the inject.Named option.
// When used with annotation.Injectable, the analyzer registers the dependency
// with the name returned by the Namer's Name() method.
// This enables multiple instances of the same type distinguished by name.
func ExampleNamed() {
	type PrimaryDB struct {
		annotation.Injectable[inject.Named[primaryDBNamer]]
	}
	_ = PrimaryDB{}
}

// ExampleWithoutConstructor demonstrates the inject.WithoutConstructor option.
// When used with annotation.Injectable, the analyzer skips constructor generation.
// You must provide a manual constructor named New<TypeName>.
func ExampleWithoutConstructor() {
	type CustomService struct {
		annotation.Injectable[inject.WithoutConstructor]
	}

	// Manual constructor required:
	// func NewCustomService() *CustomService { return &CustomService{} }
	_ = CustomService{}
}

// ExampleOption_custom demonstrates composing multiple inject options.
// Create an anonymous interface embedding multiple option interfaces
// to combine behaviors such as Typed[I] and WithoutConstructor.
func ExampleOption_custom() {
	type IService any

	type Repository any

	type Service struct {
		annotation.Injectable[interface {
			inject.Typed[IService]
			inject.WithoutConstructor
		}]
		repository Repository
	}
	_ = Service{}
}

// primaryDBNamer is a Namer implementation used in examples.
// It satisfies namer.Namer by returning a hardcoded string literal.
type primaryDBNamer struct{}

func (primaryDBNamer) Name() string { return "primaryDB" }

// Compile-time assertion that primaryDBNamer implements namer.Namer.
var _ namer.Namer = primaryDBNamer{}
