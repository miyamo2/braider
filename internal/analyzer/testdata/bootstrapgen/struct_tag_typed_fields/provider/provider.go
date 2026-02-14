package provider

import (
	"struct_tag_typed_fields/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// --- Named Providers (matched via braider:"name" struct tags) ---

// PrimaryCacheName is a namer for the primaryCache named dependency.
type PrimaryCacheName struct{}

func (PrimaryCacheName) Name() string { return "primaryCache" }

// CacheStore is a concrete struct registered with named provider.
type CacheStore struct{}

var _ = annotation.Provide[provide.Named[PrimaryCacheName]](NewCacheStore)

func NewCacheStore() *CacheStore {
	return &CacheStore{}
}

// MainDBName is a namer for the mainDB named dependency.
type MainDBName struct{}

func (MainDBName) Name() string { return "mainDB" }

// Database is a concrete struct registered with named provider.
type Database struct{}

var _ = annotation.Provide[provide.Named[MainDBName]](NewDatabase)

func NewDatabase() *Database {
	return &Database{}
}

// --- Typed Provider (matched via interface resolution, no name tag) ---

// ConsoleLogger implements domain.Logger and is registered as Typed[Logger].
type ConsoleLogger struct{}

func (ConsoleLogger) Log(msg string) {}

var _ = annotation.Provide[provide.Typed[domain.Logger]](NewConsoleLogger)

func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{}
}
