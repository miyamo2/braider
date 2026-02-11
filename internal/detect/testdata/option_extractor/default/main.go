package testdefault

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type MyService struct {
	annotation.Injectable[inject.Default]
}
