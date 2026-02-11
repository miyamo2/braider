// Provide-typed example demonstrates Provide[provide.Typed[I]] usage.
//
// When provide.Typed[I] is used, the braider analyzer:
//   - Registers the provider function as returning interface type I
//   - Bootstrap code declares the variable with interface type I
//   - Validates that the provider return type implements interface I
//
// This enables interface-based dependency injection where consumers
// depend on interfaces rather than concrete types.
//
// Run the analyzer:
//
//	go vet -vettool=$(which braider) -fix ./...
package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// Repository is the interface that the provider registers as.
type Repository interface {
	FindByID(id string) (string, error)
}

// UserRepository is the concrete implementation.
type UserRepository struct{}

func (r *UserRepository) FindByID(id string) (string, error) {
	return id, nil
}

// Register NewUserRepository as a provider for the Repository interface.
var _ = annotation.Provide[provide.Typed[Repository]](NewUserRepository)

// NewUserRepository returns a concrete *UserRepository.
// The provide.Typed[Repository] option tells braider to register it as Repository.
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// UserService depends on Repository (the interface, not *UserRepository).
type UserService struct {
	annotation.Injectable[inject.Default]
	repo Repository
}

var _ = annotation.App(main)

func main() {}
