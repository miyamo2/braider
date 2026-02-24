package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type Logger interface {
	Log(msg string)
}

type Tracer interface {
	Trace(msg string)
}

type AppService struct {
	annotation.Injectable[inject.Default]
	logger Logger `braider:"-"`
	tracer Tracer `braider:"-"`
}

func NewAppService() *AppService {
	return &AppService{}
}
