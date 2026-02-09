// Typed-inject example demonstrates Injectable[inject.Typed[I]] usage.
//
// When inject.Typed[I] is used, the braider analyzer:
//   - Generates a constructor that returns the interface type I
//   - Registers the dependency as interface I instead of *ConcreteStruct
//   - Bootstrap code declares variables with interface type I
//
// The concrete struct must implement the interface specified in Typed[I].
//
// Run the analyzer:
//
//	go vet -vettool=$(which braider) -fix ./...
package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// Repository is the interface that UserRepository will be registered as.
type Repository interface {
	FindByID(id string) (string, error)
}

// UserRepository is registered as Repository (not *UserRepository)
// via the inject.Typed[Repository] option.
type UserRepository struct {
	annotation.Injectable[inject.Typed[Repository]]
}

func (r *UserRepository) FindByID(id string) (string, error) {
	return id, nil
}

var _ = annotation.App(main)

func main() {}
