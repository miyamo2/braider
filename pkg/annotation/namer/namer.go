package namer

// Namer provides a Name method for naming dependencies.
// Name must return a string literal in the return statement.
type Namer interface {
	Name() string
}
