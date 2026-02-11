package constructorgen

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserService struct { // want "missing constructor for UserService"
	annotation.Injectable[inject.Default]
	repo UserRepository
}

type UserRepository interface {
	FindByID(id string) (*User, error)
}

type User struct {
	ID   string
	Name string
}
