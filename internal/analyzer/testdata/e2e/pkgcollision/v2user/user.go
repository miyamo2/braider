package user

import (
	v1user "pkgcollision/v1user"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type Handler struct {
	annotation.Injectable[inject.Default]
	v1Repo v1user.Repository
}

func NewHandler(v1Repo v1user.Repository) *Handler {
	return &Handler{v1Repo: v1Repo}
}
