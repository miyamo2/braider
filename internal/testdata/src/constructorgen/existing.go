package constructorgen

import "github.com/miyamo2/braider/pkg/annotation"

type ConfigService struct {
	annotation.Inject
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
