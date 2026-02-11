package repository

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type OrderRepository struct {
	annotation.Injectable[inject.Default]
}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}
