package repo

import "github.com/miyamo2/braider/pkg/annotation"

// OrderRepository is a Provide struct.
type OrderRepository struct {
	annotation.Provide
}

// NewOrderRepository is the constructor.
func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}
