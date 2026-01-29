package service

import (
	"example.com/dependency/cross_package/repo"

	"github.com/miyamo2/braider/pkg/annotation"
)

// OrderService is an Inject struct.
type OrderService struct {
	annotation.Inject
	repo *repo.OrderRepository
}

// NewOrderService is the constructor.
func NewOrderService(repo *repo.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}
