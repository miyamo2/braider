package namer_test

import (
	"fmt"

	"github.com/miyamo2/braider/pkg/annotation/namer"
)

// ExampleNamer demonstrates implementing the Namer interface.
// The Name() method must return a hardcoded string literal.
// The braider analyzer validates this at analysis time and rejects
// computed values, concatenations, or variable references.
func ExampleNamer() {
	type PrimaryDBName struct{}
	// Name returns a hardcoded string literal.
	// This is the only pattern accepted by the braider analyzer.
	// func (PrimaryDBName) Name() string { return "primaryDB" }

	// Valid: hardcoded string literal
	n := primaryDBNamer{}
	fmt.Println(n.Name())
	// Output: primaryDB
}

// primaryDBNamer is a valid Namer implementation for examples.
type primaryDBNamer struct{}

func (primaryDBNamer) Name() string { return "primaryDB" }

// Compile-time assertion that primaryDBNamer implements namer.Namer.
var _ namer.Namer = primaryDBNamer{}
