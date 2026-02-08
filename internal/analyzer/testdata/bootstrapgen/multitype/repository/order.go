package repository

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// OrderRepository is a Provide-annotated struct (local variable in bootstrap)
type OrderRepository struct{}

var _ = annotation.Provide[provide.Default](NewOrderRepository)

func NewOrderRepository() OrderRepository {
	return OrderRepository{}
}
