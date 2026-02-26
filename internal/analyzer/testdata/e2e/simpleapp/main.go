package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	app "github.com/miyamo2/braider/pkg/annotation/app"
)

// Simple App annotation referencing main function
var _ = annotation.App[app.Default](main) // want "bootstrap code is missing"

func main() {
}
