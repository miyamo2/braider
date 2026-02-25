package service

import (
	"example.com/dep_cross_package/repo"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// OrderService is an Inject struct.
type OrderService struct {
	annotation.Injectable[inject.Default]
	repo *repo.OrderRepository
}

// NewOrderService is the constructor.
func NewOrderService(repo *repo.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}
