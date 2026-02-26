package service

import (
	"crosspackage/repository"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserService struct { // want "missing constructor for UserService"
	annotation.Injectable[inject.Default]
	repo repository.UserRepository
}
