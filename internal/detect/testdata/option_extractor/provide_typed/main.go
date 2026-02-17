package testprovidetyped

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type MyInterface interface {
	DoWork()
}

type MyRepo struct{}

func (*MyRepo) DoWork() {}

func NewMyRepo() *MyRepo { return nil }

var _ = annotation.Provide[provide.Typed[MyInterface]](NewMyRepo)
