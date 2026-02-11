package annotation_test

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"github.com/miyamo2/braider/pkg/annotation/provide"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

// ExampleInjectable_default demonstrates the default Injectable annotation.
// When Injectable[inject.Default] is used, braider generates a constructor
// that returns *StructType and registers the dependency by its concrete type.
func ExampleInjectable_default() {
	type Service struct {
		annotation.Injectable[inject.Default]
	}
	_ = Service{}
}

// ExampleInjectable_typed demonstrates registering a dependency as an interface type.
// When Injectable[inject.Typed[I]] is used, braider generates a constructor
// that returns the interface type I and registers the dependency as that interface.
func ExampleInjectable_typed() {
	type Repository interface {
		FindByID(id string) (string, error)
	}

	type RepositoryImpl struct {
		annotation.Injectable[inject.Typed[Repository]]
	}
	_ = RepositoryImpl{}
}

// ExampleInjectable_named demonstrates registering a dependency with a name.
// When Injectable[inject.Named[N]] is used, braider registers the dependency
// with the name returned by N.Name(). The Name() method must return a string literal.
func ExampleInjectable_named() {
	type PrimaryDB struct {
		annotation.Injectable[inject.Named[primaryDBName]]
	}
	_ = PrimaryDB{}
}

// ExampleInjectable_withoutConstructor demonstrates skipping constructor generation.
// When Injectable[inject.WithoutConstructor] is used, braider does not generate
// a constructor. The user must provide a manual New<TypeName> function.
func ExampleInjectable_withoutConstructor() {
	type CustomService struct {
		annotation.Injectable[inject.WithoutConstructor]
	}
	_ = CustomService{}
}

// ExampleInjectable_mixedOptions demonstrates composing multiple options.
// Multiple option interfaces can be embedded in an anonymous interface
// to combine behaviors such as Typed[I] and Named[N].
func ExampleInjectable_mixedOptions() {
	type Repository interface {
		FindByID(id string) (string, error)
	}

	type RepositoryImpl struct {
		annotation.Injectable[interface {
			inject.Typed[Repository]
			inject.Named[primaryDBName]
		}]
	}
	_ = RepositoryImpl{}
}

// ExampleProvide_default demonstrates the default Provide annotation.
// When Provide[provide.Default] is used, braider registers the provider
// function under its declared return type.
func ExampleProvide_default() {
	type Repository struct{}
	NewRepository := func() *Repository { return &Repository{} }

	var _ = annotation.Provide[provide.Default](NewRepository)
}

// ExampleProvide_typed demonstrates registering a provider as an interface type.
// When Provide[provide.Typed[I]] is used, braider registers the provider
// function as returning interface type I.
func ExampleProvide_typed() {
	type Repository interface {
		FindByID(id string) (string, error)
	}

	type RepositoryImpl struct{}
	NewRepositoryImpl := func() *RepositoryImpl { return &RepositoryImpl{} }

	var _ = annotation.Provide[provide.Typed[Repository]](NewRepositoryImpl)
}

// ExampleProvide_named demonstrates registering a provider with a name.
// When Provide[provide.Named[N]] is used, braider registers the provider
// with the name returned by N.Name().
func ExampleProvide_named() {
	type Repository struct{}
	NewRepository := func() *Repository { return &Repository{} }

	var _ = annotation.Provide[provide.Named[primaryDBName]](NewRepository)
}

// primaryDBName is a Namer implementation used in examples.
type primaryDBName struct{}

func (primaryDBName) Name() string { return "primaryDB" }

// ExampleVariable_default demonstrates registering a pre-existing variable.
// When Variable[variable.Default] is used, braider registers the variable
// under its declared type without invoking any constructor.
func ExampleVariable_default() {
	var _ = annotation.Variable[variable.Default](os.Stdout)
}

// ExampleVariable_typed demonstrates registering a variable as an interface type.
// When Variable[variable.Typed[I]] is used, braider registers the variable
// under the interface type I instead of the argument's declared type.
func ExampleVariable_typed() {
	var _ = annotation.Variable[variable.Typed[any]](os.Stdout)
}

// ExampleVariable_named demonstrates registering a variable with a name.
// When Variable[variable.Named[N]] is used, braider registers the variable
// with the name returned by N.Name().
func ExampleVariable_named() {
	var _ = annotation.Variable[variable.Named[primaryDBName]](os.Stdout)
}

// ExampleApp demonstrates marking the entry point for bootstrap code generation.
// annotation.App(main) triggers braider to generate an IIFE that initializes
// all registered dependencies in topological order.
func ExampleApp() {
	main := func() {}
	_ = annotation.App(main)
}
