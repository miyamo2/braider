package service

import (
	"variable_mixed/domain"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserService struct { // want "missing constructor for UserService"
	annotation.Injectable[inject.Default]
	writer domain.Writer
}
