package constructorgen

import "github.com/miyamo2/braider/pkg/annotation"

type UserID string
type OrderID int64

type ServiceWithDefinedTypes struct { // want "missing constructor for ServiceWithDefinedTypes"
	annotation.Inject
	userID  UserID
	orderID OrderID
}
