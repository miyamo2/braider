package main

import "github.com/miyamo2/braider/pkg/annotation"

// Same package, multiple Apps - should use first one only
var _ = annotation.App(main) // want "bootstrap code is missing"

var _ = annotation.App(main) // want "another annotation.App in the same package is being applied"

func main() {}
