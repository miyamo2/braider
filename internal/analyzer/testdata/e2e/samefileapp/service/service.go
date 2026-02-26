package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type Service struct { // want "missing constructor for Service"
	annotation.Injectable[inject.Default]
}
