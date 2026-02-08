package main

import "github.com/miyamo2/braider/pkg/annotation"

// App annotation - should be skipped when context is cancelled (no diagnostic expected)
var _ = annotation.App(main)

func main() {
}
