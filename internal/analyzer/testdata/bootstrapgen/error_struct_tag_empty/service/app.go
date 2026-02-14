package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type Logger interface {
	Log(msg string)
}

type AppService struct { // want "invalid braider struct tag on field logger: tag value must not be empty"
	annotation.Injectable[inject.Default]
	logger Logger `braider:""`
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
