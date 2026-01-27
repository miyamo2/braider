package internal_test

import (
	"testing"

	"github.com/miyamo2/braider/internal"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()

	// Run with suggested fixes to verify code generation
	analysistest.RunWithSuggestedFixes(t, testdata, internal.Analyzer, "constructorgen")
}
