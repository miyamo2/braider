package constructorgen

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// NoInjectService has no annotation.Inject embedding - should be skipped
type NoInjectService struct {
	repo NoInjectRepo
}

type NoInjectRepo interface{}

// NamedInjectService has named inject field (not embedded) - should be skipped
type NamedInjectService struct {
	inject annotation.Injectable[inject.Default]
	repo   NamedInjectRepo
}

type NamedInjectRepo interface{}

// InjectOnlyService has only annotation.Inject (no injectable fields) - should be skipped
type InjectOnlyService struct {
	annotation.Injectable[inject.Default]
}
