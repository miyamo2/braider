package repository

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type OrderRepository struct { // want "missing constructor for OrderRepository"
	annotation.Injectable[inject.Default]
}
