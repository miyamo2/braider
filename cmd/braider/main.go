package main

import (
	"github.com/miyamo2/braider/internal"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(internal.Analyzer)
}
