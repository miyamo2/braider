package service

import (
	"variable_mixed/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserService struct {
	annotation.Injectable[inject.Default]
	writer domain.Writer
}

func NewUserService(writer domain.Writer) *UserService {
	return &UserService{writer: writer}
}
