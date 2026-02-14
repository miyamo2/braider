# Requirements Document

## Introduction

This specification defines the `braider` struct tag feature, which provides a field-level mechanism for controlling dependency injection behavior. The `braider` struct tag enables two capabilities: (1) injecting a named dependency into a specific struct field using `braider:"name"`, and (2) excluding a field from dependency injection entirely using `braider:"-"`. This complements the existing type-level annotation options (`inject.Named[N]`, `inject.Typed[I]`) by offering fine-grained, per-field control within `Injectable[T]` structs.

## Requirements

### Requirement 1: Named Dependency Injection via Struct Tag

**Objective:** As a developer, I want to annotate a struct field with `braider:"name"` so that braider resolves that field's dependency using the named dependency matching the specified name, enabling multiple instances of the same type to be distinguished at the field level.

#### Acceptance Criteria

1. When a field in an `Injectable[T]` struct has a `braider:"<name>"` tag, the DependencyAnalyzer shall resolve that field's dependency by looking up a provider, injector, or variable registered with the matching name.
2. When a field has a `braider:"<name>"` tag and no dependency with the specified name is registered, the analyzer shall emit a diagnostic error identifying the unresolved named dependency and the field.
3. When a field has a `braider:"<name>"` tag, the generated constructor shall use the named dependency as the parameter for that field.
4. When a field has a `braider:"<name>"` tag, the dependency graph shall create an edge from the containing struct to the node identified by the composite key `TypeName#name`.
5. The DependencyAnalyzer shall support `braider:"<name>"` tags on fields of any supported type (concrete types, pointer types, and interface types).
6. When multiple fields in the same struct have `braider:"<name>"` tags with different names, the DependencyAnalyzer shall resolve each field independently using its respective named dependency.

### Requirement 2: Field Exclusion via Struct Tag

**Objective:** As a developer, I want to annotate a struct field with `braider:"-"` so that braider ignores that field during dependency injection, allowing me to have fields in an `Injectable[T]` struct that are not managed by the DI container.

#### Acceptance Criteria

1. When a field in an `Injectable[T]` struct has a `braider:"-"` tag, the FieldAnalyzer shall exclude that field from the list of injectable fields.
2. When a field is excluded via `braider:"-"`, the generated constructor shall not include a parameter for that field.
3. When a field is excluded via `braider:"-"`, the dependency graph shall not include an edge for that field's type.
4. When all non-annotation fields in an `Injectable[T]` struct have `braider:"-"` tags, the generated constructor shall have no parameters and return an empty struct literal.

### Requirement 3: Struct Tag Parsing and Validation

**Objective:** As a developer, I want braider to correctly parse and validate `braider` struct tags so that malformed or unsupported tag values produce clear diagnostic errors.

#### Acceptance Criteria

1. The DependencyAnalyzer shall parse the `braider` struct tag from the `Tag` field of `ast.Field` using Go's standard `reflect.StructTag` parsing conventions.
2. If a field has a `braider` tag with an empty value (i.e., `braider:""`), the analyzer shall emit a diagnostic error indicating that the tag value is invalid.
3. If a field has no `braider` tag, the DependencyAnalyzer shall treat the field as a standard injectable dependency (current default behavior).
4. The DependencyAnalyzer shall only recognize the tag key `braider`; other struct tags on the same field shall be ignored and have no effect on DI behavior.

### Requirement 4: Interaction with Existing Annotation Options

**Objective:** As a developer, I want the `braider` struct tag to work correctly alongside existing annotation options (`inject.Named[N]`, `inject.Typed[I]`, `inject.WithoutConstructor`) so that the tag provides field-level refinement without conflicting with type-level options.

#### Acceptance Criteria

1. When an `Injectable[T]` struct uses `inject.Named[N]` at the type level and a field uses `braider:"<name>"` at the field level, the DependencyAnalyzer shall apply the type-level name for registration and the field-level name for dependency resolution independently.
2. When an `Injectable[T]` struct uses `inject.Typed[I]` at the type level, the `braider:"<name>"` and `braider:"-"` tags on fields shall function identically to structs without `inject.Typed[I]`.
3. When an `Injectable[T]` struct uses `inject.WithoutConstructor`, the `braider:"-"` tag shall not be used on fields that the existing constructor accepts as parameters, and the `braider:"<name>"` tag shall not be used on fields that the existing constructor does not accept as parameters. The analyzer shall emit a diagnostic error when a `braider` struct tag conflicts with the existing constructor's signature.

### Requirement 5: Bootstrap Code Generation Compatibility

**Objective:** As a developer, I want the bootstrap code generated by the AppAnalyzer to correctly wire named dependencies specified via `braider` struct tags so that the generated IIFE initializes all dependencies in the correct order.

#### Acceptance Criteria

1. When the AppAnalyzer generates bootstrap code, dependencies resolved via `braider:"<name>"` tags shall be passed as arguments to the constructor using the named dependency's variable or expression.
2. When the dependency graph includes edges derived from `braider:"<name>"` tags, the TopologicalSorter shall include those named dependencies in the initialization order.
3. When bootstrap code is regenerated, the hash computation shall include struct tag-derived dependency information so that changes to `braider` tags trigger regeneration.

### Requirement 6: Idempotent Behavior

**Objective:** As a developer, I want braider to handle struct tag changes idempotently so that re-running the analyzer after modifying `braider` tags produces correct updated output without duplication.

#### Acceptance Criteria

1. When a `braider` struct tag is added to a field that was previously an unnamed dependency, the analyzer shall regenerate the constructor and bootstrap code to reflect the named dependency.
2. When a `braider:"-"` tag is added to a field that was previously injectable, the analyzer shall regenerate the constructor without that field's parameter and update the bootstrap code accordingly.
3. When a `braider` struct tag is removed from a field, the analyzer shall revert to treating the field as a standard unnamed dependency and regenerate accordingly.
4. When no `braider` struct tags have changed since the last analysis, the analyzer shall skip regeneration based on the existing hash marker.
