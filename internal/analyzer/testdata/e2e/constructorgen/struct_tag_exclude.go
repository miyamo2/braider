package constructorgen

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type CacheService struct { // want "missing constructor for CacheService"
	annotation.Injectable[inject.Default]
	store     CacheStore
	debugTool DebugTool `braider:"-"`
	ttlConfig TTLConfig `braider:"-"`
}

type CacheStore interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
}

type DebugTool interface {
	Trace(msg string)
}

type TTLConfig struct {
	MaxAge int
}
