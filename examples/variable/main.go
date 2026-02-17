// Variable example demonstrates annotation.Variable[variable.Default] usage.
//
// When annotation.Variable is used, the braider analyzer:
//   - Registers a pre-existing variable or package-qualified identifier as a DI dependency
//   - No constructor is generated or invoked for the variable
//   - In bootstrap code, the variable is assigned directly (e.g., stdout := os.Stdout)
//
// Supported argument expressions are identifiers (myVar) and
// package-qualified selectors (os.Stdout).
//
// Run the analyzer:
//
//	braider -fix ./...
package main

import (
	"io"
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

// Register os.Stdout as a DI dependency of type *os.File.
var _ = annotation.Variable[variable.Default](os.Stdout)

// Logger writes messages to a *os.File dependency (resolved from os.Stdout).
type Logger struct {
	annotation.Injectable[inject.Default]
	out *os.File
}

func (l *Logger) Log(msg string) {
	_, _ = io.WriteString(l.out, msg+"\n")
}

var _ = annotation.App[app.Default](main)

func main() {}
