package constructorgen

import "github.com/miyamo2/braider/pkg/annotation"

// NoInjectService has no annotation.Inject embedding - should be skipped
type NoInjectService struct {
	repo NoInjectRepo
}

type NoInjectRepo interface{}

// NamedInjectService has named inject field (not embedded) - should be skipped
type NamedInjectService struct {
	inject annotation.Inject
	repo   NamedInjectRepo
}

type NamedInjectRepo interface{}

// InjectOnlyService has only annotation.Inject (no injectable fields) - should be skipped
type InjectOnlyService struct {
	annotation.Inject
}
