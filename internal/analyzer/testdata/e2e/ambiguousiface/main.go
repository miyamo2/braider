package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	app "github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Default](main) // want "multiple injectable structs implement interface"

func main() {}
