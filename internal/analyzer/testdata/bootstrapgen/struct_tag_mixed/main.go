package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main) // want "bootstrap code is missing"

func main() {}
