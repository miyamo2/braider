package constructorgen

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type AliasedUserID = string
type AliasedRequestID = int64

type ServiceWithTypeAlias struct { // want "missing constructor for ServiceWithTypeAlias"
	annotation.Injectable[inject.Default]
	userID    AliasedUserID
	requestID AliasedRequestID
}
