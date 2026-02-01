package repository

import "github.com/miyamo2/braider/pkg/annotation"

type OrderRepository struct {
	annotation.Inject
}

func NewOrderRepository() OrderRepository {
	return OrderRepository{}
}
