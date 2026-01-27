package constructorgen

import "github.com/miyamo2/braider/pkg/annotation"

type DBService struct { // want "missing constructor for DBService"
	annotation.Inject
	db     *DB
	logger *LogService
}

type DB struct {
	DSN string
}

type LogService struct{}
