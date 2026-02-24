package service

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type Logger struct {
	annotation.Injectable[inject.Default]
	Out *os.File
}

func NewLogger(out *os.File) *Logger {
	return &Logger{Out: out}
}
