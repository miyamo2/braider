package main

import "github.com/miyamo2/braider/pkg/annotation"

// Simple App annotation referencing main function
var _ = annotation.App(main) // want "bootstrap code is missing"

func main() {
}
