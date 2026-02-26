package constructorgen

import (
	mydb "database/sql"
	myhttp "net/http"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type AliasedService struct { // want "missing constructor for AliasedService"
	annotation.Injectable[inject.Default]
	db     *mydb.DB
	client *myhttp.Client
}
