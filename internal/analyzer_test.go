package internal_test

import (
	"testing"

	"github.com/miyamo2/braider/internal"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, internal.Analyzer, "example")
}
