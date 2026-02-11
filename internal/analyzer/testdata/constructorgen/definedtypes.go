package constructorgen

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserID string
type OrderID int64

type ServiceWithDefinedTypes struct { // want "missing constructor for ServiceWithDefinedTypes"
	annotation.Injectable[inject.Default]
	userID  UserID
	orderID OrderID
}
