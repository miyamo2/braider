package testprovidedefault

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type MyRepo struct{}

func NewMyRepo() *MyRepo { return nil }

var _ = annotation.Provide[provide.Default](NewMyRepo)
