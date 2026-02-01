package service

import (
	"example.com/multitype/repository"
	"github.com/miyamo2/braider/pkg/annotation"
)

// OrderService is an Inject-annotated struct (field in dependency struct)
type OrderService struct {
	annotation.Inject
	repo repository.OrderRepository
}

func NewOrderService(repo repository.OrderRepository) OrderService {
	return OrderService{repo: repo}
}
