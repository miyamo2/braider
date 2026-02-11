package service

import (
	"example.com/multitype/repository"
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// OrderService is an Inject-annotated struct (field in dependency struct)
type OrderService struct {
	annotation.Injectable[inject.Default]
	repo repository.OrderRepository
}

func NewOrderService(repo repository.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}
