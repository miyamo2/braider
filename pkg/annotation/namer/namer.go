package namer

// Namer provides a Name method for naming dependencies.
// Name must return a string, it must be hardcoded into return statement.
type Namer interface {
	Name() string
}
