// Container-basic example demonstrates app.Container with an anonymous struct.
//
// When app.Container[T] is used instead of app.Default, the braider analyzer:
//   - Generates a bootstrap function that returns an instance of T
//   - T must be a struct type whose fields map to registered dependencies
//   - Fields are matched by type; use braider:"name" tags for named dependencies
//
// This gives callers typed access to resolved dependencies without
// relying on the auto-generated anonymous struct.
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

// Repository is the interface for data access.
type Repository interface {
	FindByID(id string) (string, error)
}

// UserRepository is a concrete Repository implementation.
type UserRepository struct {
	annotation.Injectable[inject.Typed[Repository]]
}

func (r *UserRepository) FindByID(id string) (string, error) {
	return id, nil
}

// Service depends on Repository.
type Service struct {
	annotation.Injectable[inject.Default]
	repo Repository
}

// Use app.Container with an anonymous struct to define the bootstrap output.
// The container has a single field matching the Service dependency by type.
var _ = annotation.App[app.Container[struct {
	Svc *Service
}]](main)

func main() {}
