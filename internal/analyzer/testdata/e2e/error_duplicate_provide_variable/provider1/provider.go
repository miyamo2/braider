package provider1

import (
	"error_duplicate_provide_variable/types"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type PrimaryName struct{}

func (PrimaryName) Name() string { return "primary" }

var _ = annotation.Provide[provide.Named[PrimaryName]](NewConfig)

func NewConfig() *types.Config {
	return &types.Config{}
}
