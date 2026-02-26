package repository

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type UserRepository struct { // want "missing constructor for UserRepository"
	annotation.Injectable[inject.Default]
}
