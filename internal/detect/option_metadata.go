package detect

import "go/types"

// OptionMetadata contains extracted option configuration from type parameters.
// This metadata is extracted during annotation detection and stored in registry
// for use during code generation.
type OptionMetadata struct {
	// IsDefault indicates inject.Default, provide.Default, or variable.Default option
	IsDefault bool

	// TypedInterface contains the interface type for Typed[I] option (nil if not typed)
	TypedInterface types.Type

	// Name contains the extracted string from Named[N] option (empty if not named)
	Name string

	// WithoutConstructor indicates inject.WithoutConstructor option
	WithoutConstructor bool
}
