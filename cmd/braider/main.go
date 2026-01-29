package main

import (
	"github.com/miyamo2/braider/internal/analyzer"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		analyzer.DependencyAnalyzer,
		analyzer.AppAnalyzer,
	)
}
