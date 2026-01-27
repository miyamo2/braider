package annotation

// Inject marks a struct as a dependency injection target.
//
// When a struct is annotated with Inject, braider generates a constructor and
// the required wiring code to resolve and provide its dependencies.
type Inject struct{}
