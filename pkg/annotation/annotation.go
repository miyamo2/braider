// Package annotation provides marker types and functions for braider's
// dependency injection code generation.
//
// This package defines annotations that instruct the braider analyzer
// to generate constructors, wiring code, and main function bootstrapping.
package annotation

import (
	"github.com/miyamo2/braider/internal/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// Injectable marks a struct as a dependency to be injected.
//
// The option type parameter controls registration and constructor behavior.
// Common options include:
//   - inject.Default: default registration, constructor returns *StructType
//   - inject.Typed[I]: register the dependency as interface type I
//   - inject.Named[N]: register the dependency with name N.Name()
//   - inject.WithoutConstructor: skip constructor generation (provide New<Type>)
//
// Example:
//
//	type Service struct {
//		annotation.Injectable[inject.Default]
//		repo Repository
//	}
//
//	type RepositoryImpl struct {
//		annotation.Injectable[inject.Typed[Repository]]
//	}
//
// Mixed options can be composed with embedded option interfaces:
//
//	type SpecialService struct {
//		annotation.Injectable[interface {
//			inject.Typed[Repository]
//			inject.Named[ServiceName]
//		}]
//	}
type Injectable[T inject.Option] interface {
	annotation.Injectable
	option() T
}

type provider[T provide.Option] struct {
	annotation.Provider
}

func (p provider[T]) option() T {
	var zero T
	return zero
}

// Provide marks a function as a dependency provider.
//
// The option type parameter controls registration behavior.
//   - provide.Default: register the function's return type
//   - provide.Typed[I]: register the function as interface type I
//   - provide.Named[N]: register the function with name N.Name()
//
// Example:
//
//	var _ = annotation.Provide[provide.Default](NewRepository)
//	var _ = annotation.Provide[provide.Typed[Repository]](NewRepository)
//	var _ = annotation.Provide[provide.Named[PrimaryRepoName]](NewRepository)
func Provide[T provide.Option](providerFunc any) provider[T] {
	_ = providerFunc
	return provider[T]{}
}

type app struct {
	annotation.App
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
//	import (
//		"time"
//
//		"github.com/miyamo2/braider/pkg/annotation"
//		"github.com/miyamo2/braider/pkg/annotation/inject"
//		"github.com/miyamo2/braider/pkg/annotation/provide"
//	)
//
//	type Clock interface {
//		Now() time.Time
//	}
//
//	type realClock struct{}
//
//	func (realClock) Now() time.Time { return time.Now() }
//
//	var _ = annotation.Provide[provide.Typed[Clock]](NewClock)
//
//	func NewClock() *realClock { return &realClock{} }
//
//	type Service struct {
//		annotation.Injectable[inject.Default]
//		Clock Clock
//	}
//
//	var _ = annotation.App(main)
//
//	func main() {}
//
// Example generated main function:
//
//	package main
//
//	import (
//		"time"
//
//		"github.com/miyamo2/braider/pkg/annotation"
//		"github.com/miyamo2/braider/pkg/annotation/inject"
//		"github.com/miyamo2/braider/pkg/annotation/provide"
//	)
//
//	type Clock interface {
//		Now() time.Time
//	}
//
//	type realClock struct{}
//
//	func (realClock) Now() time.Time { return time.Now() }
//
//	var _ = annotation.Provide[provide.Typed[Clock]](NewClock)
//
//	func NewClock() *realClock { return &realClock{} }
//
//	type Service struct {
//		annotation.Injectable[inject.Default]
//		Clock Clock
//	}
//
//	var _ = annotation.App(main)
//
//	func main() {
//		_ = dependency
//	}
//
//	var dependency = func() struct {
//		Service Service
//	} {
//		clock := NewClock()
//		service := NewService(clock)
//		return struct {
//			Service Service
//		}{
//			Service: service,
//		}
//	}
func App(_ func()) annotation.App {
	return app{}
}
