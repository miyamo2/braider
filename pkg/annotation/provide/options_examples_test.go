package provide_test

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/namer"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// ExampleDefault demonstrates the provide.Default option.
// When used with annotation.Provide, the analyzer registers the provider
// function under its declared return type.
func ExampleDefault() {
	var _ = annotation.Provide[provide.Default](NewService)
}

// ExampleTyped demonstrates the provide.Typed option.
// When used with annotation.Provide, the analyzer registers the provider
// function as returning the specified interface type instead of its concrete return type.
func ExampleTyped() {
	var _ = annotation.Provide[provide.Typed[IService]](NewService)
}

// ExampleNamed demonstrates the provide.Named option.
// When used with annotation.Provide, the analyzer registers the provider
// with the name returned by the Namer's Name() method.
func ExampleNamed() {
	var _ = annotation.Provide[provide.Named[ServiceNamer]](NewService)
}

// ExampleOption_custom demonstrates composing multiple provide options.
// Create an anonymous interface embedding multiple option interfaces
// to combine behaviors such as Typed[I] and Named[N].
func ExampleOption_custom() {
	var _ = annotation.Provide[interface {
		provide.Typed[IService]
		provide.Named[ServiceNamer]
	}](NewService)
}

// ServiceNamer is a Namer implementation used in examples.
// It satisfies namer.Namer by returning a hardcoded string literal.
type ServiceNamer struct{}

func (ServiceNamer) Name() string {
	return "MyService"
}

// Compile-time assertion that ServiceNamer implements namer.Namer.
var _ namer.Namer = ServiceNamer{}

// IService is an example interface for provider type registration.
type IService any

// Service is an example concrete type implementing IService.
type Service struct{}

// NewService is an example provider function.
func NewService() *Service {
	return &Service{}
}
