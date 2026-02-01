package user

import "github.com/miyamo2/braider/pkg/annotation"

type Repository struct {
	annotation.Provide
}

type Service struct {
	annotation.Inject
	repo Repository
}
