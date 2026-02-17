# Requirements Document

## Introduction

This specification defines the `app-with-container-def` feature for braider. Currently, `annotation.App` always generates a bootstrap IIFE that returns an anonymous struct whose fields are automatically derived from the dependency graph. The `app.Container[T]` option type already exists in the public API (`pkg/annotation/app/options.go`) but the analyzer does not yet implement the behavior.

This feature enables users to specify a user-defined container struct type as the type parameter `T` in `annotation.App[app.Container[T]](main)`. When this option is used, the bootstrap IIFE shall return an instance of the user-defined struct type instead of the auto-generated anonymous struct. The user-defined container struct defines which resolved dependencies are exposed to the caller, using field types (and optionally `braider` struct tags for named resolution) to map fields to registered dependencies.

## Requirements

### Requirement 1: Container Option Detection

**Objective:** As a braider user, I want the analyzer to detect when `annotation.App[app.Container[T]](main)` is used, so that braider knows to generate a bootstrap with my custom container type instead of the default anonymous struct.

#### Acceptance Criteria

1. When an `annotation.App[app.Container[T]](main)` call is detected, the braider analyzer shall extract the type parameter `T` from the `app.Container` option and make it available for bootstrap generation.
2. The braider analyzer shall support both named struct types (e.g., `app.Container[MyContainer]`) and anonymous struct types (e.g., `app.Container[struct { field Type }]`) as the type parameter `T`.
3. When `T` is an anonymous struct, the braider analyzer shall extract field definitions (names, types, struct tags) directly from the AST of the type parameter expression.
4. When an `annotation.App[app.Default](main)` call is detected, the braider analyzer shall generate the bootstrap using the existing default behavior (auto-generated anonymous struct).
5. The braider analyzer shall distinguish `app.Container[T]` from `app.Default` by checking whether the type argument implements the `AppContainer` marker interface from `internal/annotation`.

### Requirement 2: Container Struct Validation

**Objective:** As a braider user, I want the analyzer to validate my container struct definition at analysis time, so that I receive clear error diagnostics before code generation rather than broken generated code.

#### Acceptance Criteria

1. When the container type parameter `T` is a struct type, the braider analyzer shall accept it as a valid container definition.
2. If the container type parameter `T` is not a struct type, the braider analyzer shall emit a diagnostic error indicating that `app.Container` requires a struct type parameter.
3. When a container struct field's type does not match any registered dependency type (Injectable, Provide, or Variable), the braider analyzer shall emit a diagnostic error identifying the unresolved field and its type.
4. When a container struct field has a `braider:"name"` struct tag, the braider analyzer shall resolve the field to the dependency registered under that name.
5. When a container struct field has a `braider:"-"` struct tag, the braider analyzer shall emit a diagnostic error indicating that excluding fields is not permitted in container structs (fields that should not be wired should be removed from the container definition).
6. If a container struct field has an empty `braider:""` struct tag, the braider analyzer shall emit a diagnostic error indicating ambiguous intent.
7. When a container struct field's type matches multiple registered dependencies (ambiguous resolution) and no `braider` struct tag disambiguates, the braider analyzer shall emit a diagnostic error listing the ambiguous candidates.

### Requirement 3: Bootstrap Code Generation with User-Defined Container

**Objective:** As a braider user, I want the bootstrap IIFE to return my user-defined container struct, so that I can access resolved dependencies through well-known field names and types that I control.

#### Acceptance Criteria

1. When `app.Container[T]` is used and validation passes, the braider analyzer shall generate a bootstrap IIFE whose return type is `T`.
2. When `T` is a named struct type, the return type in the generated IIFE shall use the qualified type name (e.g., `func() pkg.MyContainer { ... }()`).
3. When `T` is an anonymous struct type, the return type in the generated IIFE shall use the anonymous struct type literal (e.g., `func() struct { Field Type } { ... }()`).
4. The braider analyzer shall generate initialization code for all dependencies required to populate the container struct fields, following topological order.
5. The braider analyzer shall generate initialization code for transitive dependencies that are not container fields but are needed to construct container field values.
6. The braider analyzer shall generate a return statement that populates each container struct field with its corresponding resolved dependency value.
7. The braider analyzer shall include the `// braider:hash:<hash>` comment on the generated bootstrap variable for idempotency.
8. The braider analyzer shall generate the `_ = dependency` reference in the main function body, consistent with default App behavior.

### Requirement 4: Import Management for Container Bootstrap

**Objective:** As a braider user, I want the generated bootstrap code to include all necessary imports, so that the generated code compiles without manual import adjustments.

#### Acceptance Criteria

1. The braider analyzer shall include import statements for all packages referenced by dependency constructors used in the bootstrap IIFE.
2. The braider analyzer shall include import statements for all packages referenced by Variable expression texts used in the bootstrap IIFE.
3. When the container struct (named or anonymous) references types from external packages in its field types, the braider analyzer shall include import statements for those packages in the generated code.
4. When `T` is a named struct type from an external package, the braider analyzer shall include an import statement for that package.
5. The braider analyzer shall apply the existing collision-safe alias generation for package name conflicts, consistent with default App behavior.

### Requirement 5: Idempotent Regeneration with Container

**Objective:** As a braider user, I want braider to skip regeneration when my container-based bootstrap is already up-to-date, so that unnecessary file rewrites are avoided.

#### Acceptance Criteria

1. When existing bootstrap code has a `// braider:hash:<hash>` comment that matches the current computed graph hash, the braider analyzer shall skip regeneration.
2. When the computed graph hash differs from the existing hash (e.g., dependencies changed, container struct modified), the braider analyzer shall regenerate the bootstrap code.
3. The braider analyzer shall compute the graph hash using the same inputs as default behavior (TypeName, ConstructorName, IsField, Dependencies, ExpressionText), plus the container struct field definitions when `app.Container[T]` is used.

### Requirement 6: Error Diagnostics

**Objective:** As a braider user, I want clear and actionable error messages when my container definition has problems, so that I can quickly fix issues.

#### Acceptance Criteria

1. If a container struct field references an interface type with no registered implementation, the braider analyzer shall emit a diagnostic error identifying the unresolved interface type and the field name.
2. If a container struct field references a concrete type with no registered provider or injectable, the braider analyzer shall emit a diagnostic error identifying the unresolved type and the field name.
3. If a circular dependency is detected among the dependencies required to populate the container, the braider analyzer shall emit a diagnostic error with the cycle path, consistent with default App behavior.
4. If the `app.Container` type parameter contains a non-struct type (e.g., `app.Container[int]`), the braider analyzer shall emit a diagnostic error specifying that a struct type is required.
5. The braider analyzer shall include source position information in all diagnostic errors for IDE navigation.

### Requirement 7: Mixed Options with Container

**Objective:** As a braider user, I want to combine `app.Container` with other future App options using anonymous interface embedding, so that the option system remains composable.

#### Acceptance Criteria

1. The braider analyzer shall support `app.Container[T]` as a standalone option in `annotation.App[app.Container[T]](main)`.
2. Where mixed App options are composed via anonymous interface embedding (e.g., `annotation.App[interface{ app.Container[T]; ... }](main)`), the braider analyzer shall extract the `Container` option and apply it correctly.
