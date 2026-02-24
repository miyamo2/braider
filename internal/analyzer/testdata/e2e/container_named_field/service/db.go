package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type PrimaryDBName struct{}

func (PrimaryDBName) Name() string { return "primaryDB" }

type SecondaryDBName struct{}

func (SecondaryDBName) Name() string { return "secondaryDB" }

type DB struct {
	annotation.Injectable[inject.Named[PrimaryDBName]]
}

func NewDB() *DB {
	return &DB{}
}

type SecondaryDB struct {
	annotation.Injectable[inject.Named[SecondaryDBName]]
}

func NewSecondaryDB() *SecondaryDB {
	return &SecondaryDB{}
}
