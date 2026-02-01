package writer

import (
	"github.com/miyamo2/braider/pkg/annotation"
)

// MyInterface is a project-internal interface with no implementation
type MyInterface interface {
	DoSomething()
}

type MyWriter struct {
	annotation.Inject
	iface MyInterface // No injectable implements MyInterface
}

func NewMyWriter(iface MyInterface) MyWriter {
	return MyWriter{iface: iface}
}
