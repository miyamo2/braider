package repo

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// OrderRepository is a Provide struct.
type OrderRepository struct{}

var _ = annotation.Provide[provide.Default](NewOrderRepository)

// NewOrderRepository is the constructor.
func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}
