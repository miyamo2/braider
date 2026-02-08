package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type Service struct {
	annotation.Injectable[inject.Default]
}

func NewService() Service {
	return Service{}
}

// Same package, multiple Apps - should use first one only
var _ = annotation.App(main) // want "bootstrap code is missing"

var _ = annotation.App(main) // want "another annotation.App in the same package is being applied"

func main() {}
