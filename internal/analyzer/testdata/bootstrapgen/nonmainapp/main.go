package main

import "github.com/miyamo2/braider/pkg/annotation"

// App annotation referencing non-main function - should report error
var _ = annotation.App(setup) // want "annotation.App must reference main function"

func setup() {
}

func main() {
}
