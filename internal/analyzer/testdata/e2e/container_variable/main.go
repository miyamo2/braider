package main

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container[struct { // want "bootstrap code is missing"
	Writer *os.File
}]](main)

func main() {}
