package constructorgen

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type ConfigService struct {
	annotation.Injectable[inject.Default]
	config ConfigData
	logger ConfigLogger
}

// NewConfigService is an outdated constructor.
func NewConfigService(config ConfigData) *ConfigService { // want "outdated constructor for ConfigService"
	return &ConfigService{
		config: config,
	}
}

type ConfigData struct {
	DSN string
}

type ConfigLogger interface {
	Info(msg string)
}
