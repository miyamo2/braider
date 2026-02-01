package writer

import (
	"io"

	"github.com/miyamo2/braider/pkg/annotation"
)

type MyWriter struct {
	annotation.Inject
	reader io.Reader
}

func NewMyWriter(reader io.Reader) MyWriter {
	return MyWriter{reader: reader}
}
