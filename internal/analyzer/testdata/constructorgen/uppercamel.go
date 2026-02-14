package constructorgen

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type eventHandler struct { // want "missing constructor for eventHandler"
	annotation.Injectable[inject.Default]
	dispatcher EventDispatcher
}

type EventDispatcher interface {
	Dispatch(event Event)
}

type Event struct {
	Type string
}
