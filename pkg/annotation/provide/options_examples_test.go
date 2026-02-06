package provide_test

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

func ExampleOption_custom() {
	var _ = annotation.Provide[interface {
		provide.Typed[IService]
		provide.Named[ServiceNamer]
	}](NewService)
}

type ServiceNamer struct{}

func (ServiceNamer) Name() string {
	return "MyService"
}

type IService interface{}

type Service struct{}

func NewService() *Service {
	return &Service{}
}
