// Container-named example demonstrates app.Container with a named container type.
//
// When app.Container[T] is used with a named struct type from another package,
// the braider analyzer:
//   - Resolves container fields against registered DI dependencies
//   - Generates a bootstrap function that returns the named container type
//   - Enables type-safe access to dependencies through the container
//
// This pattern is useful for larger applications where the container
// definition lives in a separate package for reusability and testing.
//
// Run the analyzer:
//
//	braider -fix ./...
package main

import (
	"github.com/miyamo2/braider/examples/container-named/container"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

// Use app.Container with a named struct type from the container package.
var _ = annotation.App[app.Container[container.AppContainer]](main)

func main() {}
