package registry

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// DuplicateRegistration records a duplicate (TypeName, Name) registration detected
// during aggregation. Since AfterPhase callbacks cannot emit diagnostics directly,
// these are collected and reported by AppAnalyzeRunner in the subsequent phase.
type DuplicateRegistration struct {
	TypeName  string
	Name      string
	Location1 string // package path of the first registration
	Location2 string // package path of the duplicate registration
}

var _ = annotation.Provide[provide.Default](NewDuplicateRegistry)

// DuplicateRegistry collects duplicate (TypeName, Name) registration errors
// detected during aggregation for deferred diagnostic reporting.
type DuplicateRegistry struct {
	duplicates []DuplicateRegistration
}

// NewDuplicateRegistry creates a new empty DuplicateRegistry.
func NewDuplicateRegistry() *DuplicateRegistry {
	return &DuplicateRegistry{}
}

// Add records a duplicate registration error.
func (r *DuplicateRegistry) Add(dup DuplicateRegistration) {
	r.duplicates = append(r.duplicates, dup)
}

// GetAll returns all collected duplicate registrations.
func (r *DuplicateRegistry) GetAll() []DuplicateRegistration {
	return r.duplicates
}
