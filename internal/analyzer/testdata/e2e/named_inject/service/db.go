package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type PrimaryDBName struct{}

func (PrimaryDBName) Name() string { return "primaryDB" }

type SecondaryDBName struct{}

func (SecondaryDBName) Name() string { return "secondaryDB" }

type PrimaryDB struct { // want "missing constructor for PrimaryDB"
	annotation.Injectable[inject.Named[PrimaryDBName]]
}

type SecondaryDB struct { // want "missing constructor for SecondaryDB"
	annotation.Injectable[inject.Named[SecondaryDBName]]
}
