package missing_constructor

// UserRepository is a struct without a constructor and without Provide annotation.
// With the new annotation.Provide[T](fn) API, a missing constructor means
// there is no Provide call, so no provider is registered.
type UserRepository struct{}
