// Struct-tag-named example demonstrates braider:"<name>" struct tag usage.
//
// When a field has a braider:"<name>" struct tag, the braider analyzer:
//   - Resolves that field's dependency using the named dependency matching <name>
//   - Creates a dependency graph edge to the composite key TypeName#name
//   - Passes the named dependency as a constructor argument for that field
//
// This enables field-level control over which named instance is injected,
// complementing the type-level inject.Named[N] option.
//
// Run the analyzer:
//
//	braider -fix ./...
package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// Database represents a database connection.
type Database struct{}

// PrimaryDBName is a Namer that returns the name "primaryDB".
type PrimaryDBName struct{}

func (PrimaryDBName) Name() string { return "primaryDB" }

// ReplicaDBName is a Namer that returns the name "replicaDB".
type ReplicaDBName struct{}

func (ReplicaDBName) Name() string { return "replicaDB" }

// NewPrimaryDB creates the primary database connection.
func NewPrimaryDB() *Database { return &Database{} }

// NewReplicaDB creates the replica database connection.
func NewReplicaDB() *Database { return &Database{} }

// Register two Database instances with different names.
var _ = annotation.Provide[provide.Named[PrimaryDBName]](NewPrimaryDB)
var _ = annotation.Provide[provide.Named[ReplicaDBName]](NewReplicaDB)

// Service uses braider struct tags to specify which named Database goes to which field.
// The "primaryDB" tag on writer resolves to the provider named "primaryDB".
// The "replicaDB" tag on reader resolves to the provider named "replicaDB".
type Service struct {
	annotation.Injectable[inject.Default]
	writer *Database `braider:"primaryDB"`
	reader *Database `braider:"replicaDB"`
}

var _ = annotation.App(main)

func main() {}
