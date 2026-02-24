package service

import (
	"struct_tag_mixed/repository"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type Logger interface {
	Log(msg string)
}

type Debugger interface {
	Debug(msg string)
}

type AppService struct { // want "missing constructor for AppService"
	annotation.Injectable[inject.Default]
	repo     *repository.UserRepository `braider:"primaryRepo"`
	logger   Logger
	debugger Debugger `braider:"-"`
}

type LoggerImpl struct { // want "missing constructor for LoggerImpl"
	annotation.Injectable[inject.Default]
}

func (l *LoggerImpl) Log(msg string) {}
