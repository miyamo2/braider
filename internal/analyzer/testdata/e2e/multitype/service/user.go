package service

import (
	"example.com/multitype/repository"
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// UserService is an Inject-annotated struct (field in dependency struct)
type UserService struct { // want "missing constructor for UserService"
	annotation.Injectable[inject.Default]
	repo repository.UserRepository
}
