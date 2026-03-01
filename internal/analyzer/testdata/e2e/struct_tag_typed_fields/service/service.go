package service

import (
	"struct_tag_typed_fields/domain"
	"struct_tag_typed_fields/provider"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// InternalState is excluded from DI via braider:"-".
type InternalState interface {
	Reset()
}

// Type compatibility test across all supported field types with struct tags.
// Verifies constructor generation and bootstrap wiring for:
// - Pointer field with braider:"primaryCache" named tag
// - Pointer field with braider:"mainDB" named tag
// - Interface field (untagged, resolved via interface registry)
// - Interface field excluded via braider:"-"
type AppService struct { // want "missing constructor for AppService"
	annotation.Injectable[inject.Default]
	cache    *provider.CacheStore `braider:"primaryCache"`
	db       *provider.Database   `braider:"mainDB"`
	logger   domain.Logger
	internal InternalState `braider:"-"`
}
