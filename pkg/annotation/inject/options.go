package inject

import "github.com/miyamo2/braider/pkg/annotation/namer"

type option struct{}

// Option configures annotaion.Inject behavior.
//
// Your custom options(mixed-in options) must implement this interface.
type Option interface {
	isOption() option
}

// Default configures the annotation.Injectable to default behavior.
type Default interface {
	Option
	isDefault()
}

// Typed configures the annotation.Injectable to register an instance in the container with a specific type.
// If not set, a pointer to the struct type is used as the registration type.
type Typed[T any] interface {
	Option
	typed() T
}

// Named configures the annotation.Injectable to register an instance in the container with a specific name.
// If not set, the injectable is registered without a name.
type Named[T namer.Namer] interface {
	Option
	named() T
}

// WithoutConstructor configures the annotation.Injectable to skip generating a constructor function.
// If this option is set, you must provide a custom constructor function for the injectable type manually.
// Custom constructor functions must be named New<UpperCamelTypeName>.
type WithoutConstructor interface {
	Option
	withoutConstructor()
}
