package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	app "github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Default](main) // want "failed to build dependency graph: unresolvable dependency type: \\*os\\.File; did you mean os\\.File#stdout\\?"

func main() {}
