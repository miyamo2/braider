// Package namer provides the Namer interface for naming dependencies
// in braider's dependency injection system.
//
// Implementations of Namer are used with inject.Named[N], provide.Named[N], and variable.Named[N]
// to register dependencies under specific names, enabling multiple instances
// of the same type to be distinguished by name.
//
// The braider analyzer validates at analysis time that Name() returns a
// hardcoded string literal. Computed values, concatenations, variable
// references, and function calls are rejected with a diagnostic error.
//
// Example implementation:
//
//	type PrimaryDBName struct{}
//
//	func (PrimaryDBName) Name() string { return "primaryDB" }
//
// Usage with inject.Named:
//
//	type PrimaryDB struct {
//	    annotation.Injectable[inject.Named[PrimaryDBName]]
//	}
//
// Usage with provide.Named:
//
//	var _ = annotation.Provide[provide.Named[PrimaryDBName]](NewPrimaryDB)
package namer

import "github.com/miyamo2/braider/internal/annotation"

var _ Namer = (annotation.Namer)(nil)

// Namer provides a Name method for naming dependencies.
//
// The Name method must return a hardcoded string literal in its return
// statement. The braider analyzer performs AST analysis on the Name()
// method body to verify this requirement. Non-literal return values
// (such as concatenations, variables, or function calls) will cause
// a diagnostic error during analysis.
//
// Example:
//
//	// Valid: returns a hardcoded string literal
//	type PrimaryDBName struct{}
//	func (PrimaryDBName) Name() string { return "primaryDB" }
//
//	// Invalid: computed value (will cause analyzer error)
//	type BadName struct{ prefix string }
//	func (b BadName) Name() string { return b.prefix + "DB" }
type Namer interface {
	Name() string
}
