package generate

import (
	"errors"
	"go/format"
)

// CodeFormatter formats generated Go code.
type CodeFormatter interface {
	// FormatCode applies gofmt to the provided source code.
	FormatCode(code string) (string, error)
}

// codeFormatter is the default implementation of CodeFormatter.
type codeFormatter struct{}

// NewCodeFormatter creates a new CodeFormatter instance.
func NewCodeFormatter() CodeFormatter {
	return &codeFormatter{}
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
