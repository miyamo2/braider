package variable_test

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/namer"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

// ExampleDefault demonstrates the variable.Default option.
// When used with annotation.Variable, the analyzer registers the variable
// under its declared type.
func ExampleDefault() {
	var _ = annotation.Variable[variable.Default](os.Stdout)
}

// ExampleTyped demonstrates the variable.Typed option.
// When used with annotation.Variable, the analyzer registers the variable
// as the specified interface type instead of its declared type.
func ExampleTyped() {
	var _ = annotation.Variable[variable.Typed[any]](os.Stdout)
}

// ExampleNamed demonstrates the variable.Named option.
// When used with annotation.Variable, the analyzer registers the variable
// with the name returned by the Namer's Name() method.
func ExampleNamed() {
	var _ = annotation.Variable[variable.Named[stdoutNamer]](os.Stdout)
}

// ExampleOption_custom demonstrates composing multiple variable options.
// Create an anonymous interface embedding multiple option interfaces
// to combine behaviors such as Typed[I] and Named[N].
func ExampleOption_custom() {
	var _ = annotation.Variable[interface {
		variable.Typed[any]
		variable.Named[stdoutNamer]
	}](os.Stdout)
}

// stdoutNamer is a Namer implementation used in examples.
// It satisfies namer.Namer by returning a hardcoded string literal.
type stdoutNamer struct{}

func (stdoutNamer) Name() string { return "stdout" }

// Compile-time assertion that stdoutNamer implements namer.Namer.
var _ namer.Namer = stdoutNamer{}
