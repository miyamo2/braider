package repository

import "github.com/miyamo2/braider/pkg/annotation"

// OrderRepository is a Provide-annotated struct (local variable in bootstrap)
type OrderRepository struct {
	annotation.Provide
}

func NewOrderRepository() OrderRepository {
	return OrderRepository{}
}
