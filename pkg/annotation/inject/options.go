// Package inject provides option interfaces for configuring
// annotation.Injectable behavior in braider's dependency injection system.
//
// Options control how the braider analyzer generates constructors and
// registers dependencies. Available options:
//
//   - [Default]: generates a constructor returning *StructType (default behavior)
//   - [Typed]: registers the dependency as an interface type instead of the concrete struct
//   - [Named]: registers the dependency with a specific name for disambiguation
//   - [WithoutConstructor]: skips constructor generation, requiring a manual New<Type> function
//
// Options can be combined by embedding multiple option interfaces in an
// anonymous interface:
//
//	type SpecialRepository struct {
//	    annotation.Injectable[interface {
//	        inject.Typed[Repository]
//	        inject.Named[RepositoryName]
//	    }]
//	}
//
// Custom option types must implement the [Option] interface.
package inject

import (
	"github.com/miyamo2/braider/internal/annotation"
	"github.com/miyamo2/braider/pkg/annotation/namer"
)

// Option configures annotation.Injectable behavior.
//
// Your custom options(mixed-in options) must implement this interface.
type Option interface {
	annotation.InjectableOption
}

// Default configures annotation.Injectable to the default behavior.
//
// The analyzer generates a constructor that returns *StructType.
type Default interface {
	Option
	annotation.InjectableDefault
}

// Typed configures the annotation.Injectable to register a dependency with a specific type.
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
	annotation.InjectableTyped
	typeParam() T
}

// Named configures the annotation.Injectable to register a dependency with a specific name.
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
	annotation.InjectableNamed
	nameParam() T
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
	annotation.InjectableWithoutConstructor
}
