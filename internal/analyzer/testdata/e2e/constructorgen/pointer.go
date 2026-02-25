package constructorgen

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type DBService struct { // want "missing constructor for DBService"
	annotation.Injectable[inject.Default]
	db     *DB
	logger *LogService
}

type DB struct {
	DSN string
}

type LogService struct{}
