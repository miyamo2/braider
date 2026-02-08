package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type PrimaryDBName struct{}

func (PrimaryDBName) Name() string { return "primaryDB" }

type SecondaryDBName struct{}

func (SecondaryDBName) Name() string { return "secondaryDB" }

type PrimaryDB struct {
	annotation.Injectable[inject.Named[PrimaryDBName]]
}

func NewPrimaryDB() *PrimaryDB {
	return &PrimaryDB{}
}

type SecondaryDB struct {
	annotation.Injectable[inject.Named[SecondaryDBName]]
}

func NewSecondaryDB() *SecondaryDB {
	return &SecondaryDB{}
}
