// Without-constructor example demonstrates Injectable[inject.WithoutConstructor] usage.
//
// When inject.WithoutConstructor is used, the braider analyzer:
//   - Skips constructor generation for the annotated struct
//   - Requires a manually-provided New<TypeName> function
//
// This is useful when the constructor has custom initialization logic
// that cannot be expressed by braider's auto-generated constructors.
//
// Run the analyzer:
//
//	braider -fix ./...
package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// CustomService uses WithoutConstructor to provide its own constructor.
type CustomService struct {
	annotation.Injectable[inject.WithoutConstructor]
}

// NewCustomService is the manual constructor required by WithoutConstructor.
// It must follow the naming convention New<TypeName>.
func NewCustomService() *CustomService {
	return &CustomService{}
}

var _ = annotation.App[app.Default](main)

func main() {}
