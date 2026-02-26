package service

import (
	"struct_tag_outdated/repository"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// Logger is an optional dependency excluded via braider:"-".
type Logger interface {
	Log(msg string)
}

// AppService uses braider:"-" to exclude logger from DI.
// The exclude tag ensures logger is not part of hash computation,
// verifying that struct tag exclusions are correctly reflected in the hash.
type AppService struct {
	annotation.Injectable[inject.Default]
	userRepo  *repository.UserRepository
	orderRepo *repository.OrderRepository
	logger    Logger `braider:"-"`
}

func NewAppService(userRepo *repository.UserRepository, orderRepo *repository.OrderRepository) *AppService {
	return &AppService{
		userRepo:  userRepo,
		orderRepo: orderRepo,
	}
}
