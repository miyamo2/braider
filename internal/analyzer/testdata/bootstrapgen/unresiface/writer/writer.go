package writer

import (
	"io"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type MyWriter struct {
	annotation.Injectable[inject.Default]
	reader io.Reader // No injectable implements io.Reader
}

func NewMyWriter(reader io.Reader) *MyWriter {
	return &MyWriter{reader: reader}
}
