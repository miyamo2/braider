package service

import (
	"struct_tag_named/repository"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type AppService struct { // want "missing constructor for AppService"
	annotation.Injectable[inject.Default]
	repo *repository.UserRepository `braider:"primaryRepo"`
}
