// Named-inject example demonstrates Injectable[inject.Named[N]] usage.
//
// When inject.Named[N] is used, the braider analyzer:
//   - Registers the dependency with the name returned by N.Name()
//   - Uses the name for variable naming in bootstrap code
//   - Enables multiple instances of the same type distinguished by name
//
// The Namer type N must implement namer.Namer and its Name() method
// must return a hardcoded string literal.
//
// Run the analyzer:
//
//	braider -fix ./...
package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// PrimaryDBName is a Namer that returns the name "primaryDB".
// The Name() method must return a hardcoded string literal.
type PrimaryDBName struct{}

func (PrimaryDBName) Name() string { return "primaryDB" }

// SecondaryDBName is a Namer that returns the name "secondaryDB".
type SecondaryDBName struct{}

func (SecondaryDBName) Name() string { return "secondaryDB" }

// PrimaryDB is registered with the name "primaryDB".
type PrimaryDB struct {
	annotation.Injectable[inject.Named[PrimaryDBName]]
}

// SecondaryDB is registered with the name "secondaryDB".
// Both PrimaryDB and SecondaryDB can coexist because they have different names.
type SecondaryDB struct {
	annotation.Injectable[inject.Named[SecondaryDBName]]
}

var _ = annotation.App(main)

func main() {}
