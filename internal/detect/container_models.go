package detect

import (
	"go/token"
	"go/types"
)

// AppOptionMetadata holds the extracted App option classification.
type AppOptionMetadata struct {
	IsDefault    bool                // true when app.Default or no type arg
	ContainerDef *ContainerDefinition // non-nil when app.Container[T] detected
}

// ContainerDefinition represents the user-defined container struct.
type ContainerDefinition struct {
	IsNamed     bool             // true for named struct types, false for anonymous
	TypeName    string           // Fully qualified type name (empty for anonymous)
	PackagePath string           // Import path of the named type (empty for anonymous)
	PackageName string           // Package name (empty for anonymous)
	LocalName   string           // Unqualified type name (empty for anonymous)
	StructType  *types.Struct    // The underlying struct type
	NamedType   *types.Named     // The named type (nil for anonymous)
	Fields      []ContainerField // Ordered field definitions
	Pos         token.Pos        // Position for diagnostics
}

// ContainerField represents a single field in the container struct.
type ContainerField struct {
	Name          string     // Field name
	Type          types.Type // Field type
	TypeString    string     // String representation of the field type
	Tag           string     // braider struct tag value (empty if no braider tag or braider:"")
	HasBraiderTag bool       // true if the field has a braider struct tag key (distinguishes no tag from braider:"")
	Pos           token.Pos  // Position for diagnostics
}

// ResolvedContainerField maps a container field to its dependency graph node.
type ResolvedContainerField struct {
	FieldName string // Container struct field name
	NodeKey   string // Dependency graph node key (e.g., "pkg.Type" or "pkg.Type#name")
	VarName   string // Variable name used in bootstrap initialization code
}
