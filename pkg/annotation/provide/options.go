package provide

import (
	"github.com/miyamo2/braider/pkg/annotation/namer"
)

type option struct{}

// Option configures annotaion.Inject behavior.
//
// Your custom options(mixed-in options) must implement this interface.
type Option interface {
	isOption() option
}

// Default configures the annotation.Provide to default behavior.
type Default interface {
	Option
	isDefault()
}

// Typed configures the annotation.Provide to register a function as factory for a specific type.
// If not set, the return type of the provider function is used as the registration type.
type Typed[T any] interface {
	Option
	typed() T
}

// Named configures the annotation.Provide to register a function as factory for a specific name.
// If not set, the provider is registered without a name.
type Named[T namer.Namer] interface {
	Option
	named() T
}
