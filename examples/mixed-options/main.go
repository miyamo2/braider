// Mixed-options example demonstrates composing multiple inject options.
//
// Multiple option interfaces can be embedded in an anonymous interface
// to combine behaviors. In this example, Typed[I] and Named[N] are
// combined so the dependency is:
//   - Registered as the interface type Repository
//   - Named "service" for disambiguation
//
// Run the analyzer:
//
//	braider -fix ./...
package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// ServiceName is a Namer that returns the name "service".
type ServiceName struct{}

func (ServiceName) Name() string { return "service" }

// Repository is the interface type for registration.
type Repository interface {
	FindByID(id string) (string, error)
}

// MixedService combines Typed[Repository] and Named[ServiceName].
// It is registered as Repository under the name "service".
type MixedService struct {
	annotation.Injectable[interface {
		inject.Typed[Repository]
		inject.Named[ServiceName]
	}]
}

func (s *MixedService) FindByID(id string) (string, error) {
	return id, nil
}

var _ = annotation.App[app.Default](main)

func main() {}
