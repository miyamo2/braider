package testtypednoninterface

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type SomeStruct struct {
	Value string
}

type MyService struct {
	annotation.Injectable[inject.Typed[SomeStruct]]
}
