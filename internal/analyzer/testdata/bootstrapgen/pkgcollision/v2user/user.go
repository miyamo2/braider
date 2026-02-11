package user

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"pkgcollision/v1user"
)

type Service struct {
	annotation.Injectable[inject.Default]
	v1Repo v1user.Repository
}
