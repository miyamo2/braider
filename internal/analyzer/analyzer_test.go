package analyzer_test

import (
	"testing"

	"github.com/miyamo2/braider/internal/analyzer"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	// Run with suggested fixes to verify code generation
	analysistest.RunWithSuggestedFixes(t, "testdata/constructorgen", analyzer.Analyzer, ".")
}
