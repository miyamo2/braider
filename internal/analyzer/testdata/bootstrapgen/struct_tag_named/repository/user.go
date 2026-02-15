package repository

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

type PrimaryRepoName struct{}

func (PrimaryRepoName) Name() string { return "primaryRepo" }

type UserRepository struct{}

var _ = annotation.Provide[provide.Named[PrimaryRepoName]](NewUserRepository)

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}
