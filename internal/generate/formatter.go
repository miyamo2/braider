package generate

import (
	"errors"
	"go/format"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// CodeFormatter formats generated Go code.
type CodeFormatter interface {
	// FormatCode applies gofmt to the provided source code.
	FormatCode(code string) (string, error)
}

// codeFormatter is the default implementation of CodeFormatter.
type codeFormatter struct {
	annotation.Injectable[inject.Typed[CodeFormatter]]
}

// FormatCode applies gofmt to the provided source code.
func (f *codeFormatter) FormatCode(code string) (string, error) {
	if code == "" {
		return "", errors.New("empty code")
	}

	formatted, err := format.Source([]byte(code))
	if err != nil {
		return "", err
	}

	return string(formatted), nil
}
