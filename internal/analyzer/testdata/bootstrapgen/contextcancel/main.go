package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	app "github.com/miyamo2/braider/pkg/annotation/app"
)

// App annotation - should be skipped when context is cancelled (no diagnostic expected)
var _ = annotation.App[app.Default](main)

func main() {
}
