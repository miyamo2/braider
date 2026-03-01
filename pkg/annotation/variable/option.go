// Package variable provides option interfaces for configuring
// annotation.Variable behavior in braider's dependency injection system.
//
// Options control how the braider analyzer registers pre-existing variables
// or package-qualified identifiers as dependencies. Available options:
//
//   - [Default]: registers the variable under its declared type
//   - [Typed]: registers the variable as a specific interface type instead of the declared type
//   - [Named]: registers the variable with a specific name for disambiguation
//
// Options can be combined by embedding multiple option interfaces in an
// anonymous interface:
//
//	var _ = annotation.Variable[interface {
//	    variable.Typed[io.Writer]
//	    variable.Named[StdoutName]
//	}](os.Stdout)
//
// Custom option types must implement the [Option] interface.
package variable

import (
	"github.com/miyamo2/braider/internal/annotation"
	"github.com/miyamo2/braider/pkg/annotation/namer"
)

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

// Typed configures the annotation.Variable to register a dependency with a specific type.
// If not set, the type of the variable is used as the registration type.
//
// Example:
//
//	var _ = annotation.Variable[variable.Typed[io.Writer]](os.Stdout)
type Typed[T any] interface {
	Option
	annotation.VariableTyped
	typeParam() T
}

// Named configures the annotation.Variable to register a dependency with a specific name.
// If not set, the variable is registered without a name.
//
// Name values must come from a Namer implementation that returns a string literal.
//
// Example:
//
//	type stdoutNamer struct{}
//
//	func (stdoutNamer) Name() string { return "stdout" }
//
//	var _ = annotation.Variable[variable.Named[stdoutNamer]](os.Stdout)
type Named[T namer.Namer] interface {
	Option
	annotation.VariableNamed
	nameParam() T
}
