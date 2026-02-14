package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type Logger interface {
	Log(msg string)
}

type AppService struct { // want `braider struct tag conflict on field logger: field is excluded via braider:"-" but matches constructor parameter type`
	annotation.Injectable[inject.WithoutConstructor]
	logger Logger `braider:"-"`
}

func NewAppService(logger Logger) *AppService { // want "outdated constructor for AppService"
	return &AppService{logger: logger}
}

type LoggerImpl struct {
	annotation.Injectable[inject.Default]
}

func NewLoggerImpl() *LoggerImpl {
	return &LoggerImpl{}
}

func (l *LoggerImpl) Log(msg string) {}
