package repository

type UserRepository struct{}

// No constructor defined - this is the error condition
// annotation.Provide call is intentionally omitted because there is no constructor function.
// The test registers the provider manually in the test setup with an empty constructor name.
