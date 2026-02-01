package constructorgen

import "github.com/miyamo2/braider/pkg/annotation"

type UserService struct { // want "missing constructor for UserService"
	annotation.Inject
	repo UserRepository
}

type UserRepository interface {
	FindByID(id string) (*User, error)
}

type User struct {
	ID   string
	Name string
}
