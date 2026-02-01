package user

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"pkgcollision/v1user"
)

type Service struct {
	annotation.Inject
	v1Repo v1user.Repository
}
