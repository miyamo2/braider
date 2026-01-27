package constructorgen

import (
	mydb "database/sql"
	myhttp "net/http"

	"github.com/miyamo2/braider/pkg/annotation"
)

type AliasedService struct { // want "missing constructor for AliasedService"
	annotation.Inject
	db     *mydb.DB
	client *myhttp.Client
}
