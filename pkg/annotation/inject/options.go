package inject

import "github.com/miyamo2/braider/pkg/annotation/namer"

type option struct{}

// Option configures annotation.Injectable behavior.
//
// Your custom options(mixed-in options) must implement this interface.
type Option interface {
	isOption() option
}

// Default configures annotation.Injectable to the default behavior.
//
// The analyzer generates a constructor that returns *StructType.
type Default interface {
	Option
	isDefault()
}

// Typed configures the annotation.Injectable to register an instance in the container with a specific type.
// If not set, a pointer to the struct type is used as the registration type.
//
// Example:
//
//	type Repository interface {
//		FindByID(id string) (string, error)
//	}
//
//	type UserRepository struct {
//		annotation.Injectable[inject.Typed[Repository]]
//	}
type Typed[T any] interface {
	Option
	typed() T
}

// Named configures the annotation.Injectable to register an instance in the container with a specific name.
// If not set, the injectable is registered without a name.
//
// Name values must come from a Namer implementation that returns a string literal.
//
// Example:
//
//	type PrimaryDBName struct{}
//
//	func (PrimaryDBName) Name() string { return "primaryDB" }
//
//	type PrimaryDB struct {
//		annotation.Injectable[inject.Named[PrimaryDBName]]
//	}
type Named[T namer.Namer] interface {
	Option
	named() T
}

// WithoutConstructor configures the annotation.Injectable to skip generating a constructor function.
// If this option is set, you must provide a custom constructor function for the injectable type manually.
// Custom constructor functions must be named New<UpperCamelTypeName>.
//
// Example:
//
//	type CustomService struct {
//		annotation.Injectable[inject.WithoutConstructor]
//	}
//
//	func NewCustomService() *CustomService { return &CustomService{} }
type WithoutConstructor interface {
	Option
	withoutConstructor()
}
