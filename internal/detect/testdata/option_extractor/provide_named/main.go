package testprovidenamed

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type MyRepo struct{}

func NewMyRepo() *MyRepo { return nil }

type DatabaseName struct{}

func (DatabaseName) Name() string { return "database" }

var _ = annotation.Provide[provide.Named[DatabaseName]](NewMyRepo)
