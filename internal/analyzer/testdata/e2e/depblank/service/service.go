package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserService struct {
	annotation.Injectable[inject.Default]
}

func NewUserService() *UserService {
	return &UserService{}
}

func (s UserService) Run() {}

type ItemService struct {
	annotation.Injectable[inject.Default]
}

func NewItemService() *ItemService {
	return &ItemService{}
}
