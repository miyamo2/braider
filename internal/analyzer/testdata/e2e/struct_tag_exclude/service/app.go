package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type Logger interface {
	Log(msg string)
}

type Debugger interface {
	Debug(msg string)
}

type AppService struct {
	annotation.Injectable[inject.Default]
	logger   Logger
	debugger Debugger `braider:"-"`
}

func NewAppService(logger Logger) *AppService {
	return &AppService{logger: logger}
}

type LoggerImpl struct {
	annotation.Injectable[inject.Default]
}

func NewLoggerImpl() *LoggerImpl {
	return &LoggerImpl{}
}

func (l *LoggerImpl) Log(msg string) {}
