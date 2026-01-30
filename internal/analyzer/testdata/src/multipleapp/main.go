package main

import "github.com/miyamo2/braider/pkg/annotation"

// Multiple App annotations - should report error
var _ = annotation.App(main) // want "multiple annotation.App declarations in package"

var _ = annotation.App(main)

func main() {
}
