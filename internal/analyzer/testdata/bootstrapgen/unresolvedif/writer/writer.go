package writer

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// MyInterface is a project-internal interface with no implementation
type MyInterface interface {
	DoSomething()
}

type MyWriter struct {
	annotation.Injectable[inject.Default]
	iface MyInterface // No injectable implements MyInterface
}

func NewMyWriter(iface MyInterface) MyWriter {
	return MyWriter{iface: iface}
}
