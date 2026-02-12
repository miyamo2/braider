package main

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"github.com/miyamo2/braider/pkg/annotation/variable"
)

var _ = annotation.Variable[variable.Default](os.Stdout)

var _ = annotation.App(main)

func main() {
	_ = dependency
}

// braider:hash:b52de62f27d3832e
var dependency = func() struct {
	logger *Logger
} {
	file := os.Stdout
	logger := NewLogger(file)
	return struct {
		logger *Logger
	}{
		logger: logger,
	}
}()

type Logger struct {
	annotation.Injectable[inject.Default]
	Out *os.File
}

func NewLogger(out *os.File) *Logger {
	return &Logger{Out: out}
}
