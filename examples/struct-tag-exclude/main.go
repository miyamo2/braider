// Struct-tag-exclude example demonstrates braider:"-" struct tag usage.
//
// When a field has a braider:"-" struct tag, the braider analyzer:
//   - Excludes that field from dependency injection entirely
//   - Does not generate a constructor parameter for that field
//   - Does not create a dependency graph edge for that field
//
// This is useful for fields that are initialized manually after construction,
// such as internal state, caches, or optional dependencies.
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

// Logger is an interface for logging.
type Logger interface {
	Log(msg string)
}

type stdLogger struct {
	annotation.Injectable[inject.Default]
}

func (l *stdLogger) Log(msg string) {}

// Metrics is an optional dependency that is excluded from DI.
type Metrics interface {
	Record(name string, value float64)
}

// Service has logger injected via DI, but metrics is excluded via braider:"-".
// The generated constructor will only accept Logger as a parameter.
type Service struct {
	annotation.Injectable[inject.Default]
	logger  Logger
	metrics Metrics `braider:"-"`
}

var _ = annotation.App[app.Default](main)

func main() {}
