package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main) // want "failed to build dependency graph: unresolvable dependency type: io.Reader"

func main() {}
