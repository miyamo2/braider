package providecall

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type Repository interface {
	FindByID(id string) (string, error)
}

type UserRepository struct{}

func (UserRepository) FindByID(id string) (string, error) { return "", nil }

func NewUserRepository() *UserRepository { return &UserRepository{} }

var _ = annotation.Provide[provide.Default](NewUserRepository)

type ServiceImpl struct{}

func NewServiceImpl() *ServiceImpl { return &ServiceImpl{} }

var _ = annotation.Provide[provide.Default](NewServiceImpl)
