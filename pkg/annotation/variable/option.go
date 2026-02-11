package variable

import "github.com/miyamo2/braider/internal/annotation"

// Option configures annotation.Variable behavior.
type Option interface {
	annotation.VariableOption
}

// Default configures annotation.Variable to the default behavior.
//
// Example:
//
//	var _ = annotation.Variable[variable.Default](os.Stdout)
type Default interface {
	Option
	annotation.VariableDefault
}

// Typed configures the annotation.Variable to register an instance in the container with a specific type.
// If not set, the type of the variable is used as the registration type.
//
// Example:
//
//	var _ = annotation.Variable[variable.Typed[io.Write]](os.Stdout)
type Typed[T any] interface {
	Option
	annotation.VariableTyped
	typeParam() T
}

// Named configures the annotation.Variable to register an instance in the container with a specific name.
// If not set, the variable is registered without a name.
//
// Name values must come from a Namer implementation that returns a string literal.
//
// Example:
//
//	type stdoutName struct{}
//
//	func (stdoutName) Name() string { return "stdout" }
//
//	var _ = annotation.Variable[variable.Named[stdoutName]](os.Stdout)
type Named[T any] interface {
	Option
	annotation.VariableNamed
	nameParam() T
}
