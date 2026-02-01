package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main) // want "circular dependency detected: circular/service.ServiceA -> circular/service.ServiceB -> circular/service.ServiceA"

func main() {}
