package provider

import (
	"container_provide_cross_type/ext"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

var _ = annotation.Provide[provide.Default](NewClient)

func NewClient() *ext.Client {
	return &ext.Client{}
}
