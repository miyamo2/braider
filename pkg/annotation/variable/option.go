package variable

import "github.com/miyamo2/braider/internal/annotation"

// Option configures annotation.Variable behavior.
type Option interface {
	annotation.VariableOption
}

type Default interface {
	Option
	annotation.VariableDefault
}

type Typed[T any] interface {
	Option
	annotation.VariableTyped
	typeParam() T
}

type Named[T any] interface {
	Option
	annotation.VariableNamed
	nameParam() T
}
