package main

import "github.com/miyamo2/braider/pkg/annotation"

type UserService struct {
	annotation.Inject
}

func NewUserService() UserService {
	return UserService{}
}

func (s UserService) Run() {}

type ItemService struct {
	annotation.Inject
}

func NewItemService() ItemService {
	return ItemService{}
}
