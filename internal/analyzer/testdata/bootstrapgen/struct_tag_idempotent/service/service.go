package service

import (
	"struct_tag_idempotent/repository"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// Debugger is an optional interface excluded via braider:"-".
type Debugger interface {
	Debug(msg string)
}

// AppService uses braider:"-" to exclude debug field from DI.
// The exclude tag affects constructor generation (debug not in params)
// and hash computation (fewer Dependencies in the graph node).
type AppService struct {
	annotation.Injectable[inject.Default]
	repo  *repository.UserRepository
	debug Debugger `braider:"-"`
}

func NewAppService(repo *repository.UserRepository) *AppService {
	return &AppService{
		repo: repo,
	}
}
