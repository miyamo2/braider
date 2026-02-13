package main

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"variable_outdated/service"
)

var _ = annotation.App(main) // want "bootstrap code is outdated"

func main() {
	_ = dependency
}

// braider:hash:0000000000000000
var dependency = func() struct {
	logger *service.Logger
} {
	file := os.Stdout
	logger := service.NewLogger(file)
	return struct {
		logger *service.Logger
	}{
		logger: logger,
	}
}()
