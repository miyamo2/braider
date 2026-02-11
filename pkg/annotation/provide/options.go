// Package provide provides option interfaces for configuring
// annotation.Provide behavior in braider's dependency injection system.
//
// Options control how the braider analyzer registers provider functions.
// Available options:
//
//   - [Default]: registers the provider function under its declared return type
//   - [Typed]: registers the provider function as returning a specific interface type
//   - [Named]: registers the provider function with a specific name for disambiguation
//
// Options can be combined by embedding multiple option interfaces in an
// anonymous interface:
//
//	var _ = annotation.Provide[interface {
//	    provide.Typed[RepositoryInterface]
//	    provide.Named[RepoName]
//	}](NewRepository)
//
// Custom option types must implement the [Option] interface.
package provide

import (
	"github.com/miyamo2/braider/internal/annotation"
	"github.com/miyamo2/braider/pkg/annotation/namer"
)

// Option configures annotation.Provide behavior.
//
// Your custom options(mixed-in options) must implement this interface.
type Option interface {
	annotation.ProviderOption
}

// Default configures annotation.Provide to default behavior.
//
// The analyzer registers the provider function under its return type.
type Default interface {
	Option
	annotation.ProviderDefault
}

// Typed configures the annotation.Provide to register a function as factory for a specific type.
// If not set, the return type of the provider function is used as the registration type.
//
// Example:
//
//	type Repository interface {
//		FindByID(id string) (string, error)
//	}
//
//	type UserRepository struct{}
//
//	func NewUserRepository() *UserRepository { return &UserRepository{} }
//
//	var _ = annotation.Provide[provide.Typed[Repository]](NewUserRepository)
type Typed[T any] interface {
	Option
	annotation.ProviderTyped[T]
}

// Named configures the annotation.Provide to register a function as factory for a specific name.
// If not set, the provider is registered without a name.
//
// Name values must come from a Namer implementation that returns a string literal.
//
// Example:
//
//	type PrimaryRepoName struct{}
//
//	func (PrimaryRepoName) Name() string { return "primaryRepo" }
//
//	var _ = annotation.Provide[provide.Named[PrimaryRepoName]](NewRepository)
type Named[T namer.Namer] interface {
	Option
	annotation.ProviderNamed[T]
}
