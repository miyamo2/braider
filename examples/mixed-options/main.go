// Mixed-options example demonstrates composing multiple inject options.
//
// Multiple option interfaces can be embedded in an anonymous interface
// to combine behaviors. In this example, Typed[I] and Named[N] are
// combined so the dependency is:
//   - Registered as the interface type Repository
//   - Named "repository" for disambiguation
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

// RepositoryName is a Namer that returns the name "repository".
type RepositoryName struct{}

func (RepositoryName) Name() string { return "repository" }

// Repository is the interface type for registration.
type Repository interface {
	FindByID(id string) (string, error)
}

// MixedRepository combines Typed[Repository] and Named[RepositoryName].
// It is registered as Repository under the name "repository".
type MixedRepository struct {
	annotation.Injectable[interface {
		inject.Typed[Repository]
		inject.Named[RepositoryName]
	}]
}

func (r *MixedRepository) FindByID(id string) (string, error) {
	return id, nil
}

var _ = annotation.App[app.Default](main)

func main() {}
