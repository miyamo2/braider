package repository

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// UserRepository is the original injectable (present in old bootstrap).
type UserRepository struct {
	annotation.Injectable[inject.Default]
}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// OrderRepository is a newly added injectable (not in old bootstrap).
// Its addition causes hash mismatch and bootstrap regeneration.
type OrderRepository struct {
	annotation.Injectable[inject.Default]
}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}
