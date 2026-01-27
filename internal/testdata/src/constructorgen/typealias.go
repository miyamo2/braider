package constructorgen

import "github.com/miyamo2/braider/pkg/annotation"

type AliasedUserID = string
type AliasedRequestID = int64

type ServiceWithTypeAlias struct { // want "missing constructor for ServiceWithTypeAlias"
	annotation.Inject
	userID    AliasedUserID
	requestID AliasedRequestID
}
