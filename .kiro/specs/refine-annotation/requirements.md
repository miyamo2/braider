# Requirements Document

## Introduction

The **refine-annotation** feature enhances braider's dependency injection annotation system to support advanced DI scenarios through a flexible, type-safe options pattern. This feature extends the existing `Inject`, `Provide`, and `App` annotations with generic type parameters and option interfaces, enabling users to customize constructor generation, type registration, and dependency naming without sacrificing compile-time safety.

This enhancement maintains backward compatibility with existing annotation patterns while providing new capabilities for interface-based DI, named dependencies, and custom constructor implementations. The annotation refinement enables more sophisticated DI scenarios commonly found in large-scale Go applications.

## Requirements

### Requirement 1: Generic Annotation Interfaces

**Objective:** As a braider developer, I want to define generic annotation interfaces with type parameters, so that users can customize DI behavior through compile-time type constraints.

#### Acceptance Criteria

1. The braider annotation package shall expose an `Injectable[T inject.Option]` interface with type parameter for inject options
2. The braider annotation package shall expose a `Provider[T provide.Option]` interface with type parameter for provide options
3. When a struct embeds `Injectable[T]`, the braider analyzer shall extract the type parameter `T` for option resolution
4. When a function is annotated with `Provide[T](fn)`, the braider analyzer shall extract the type parameter `T` for option resolution
5. The braider analyzer shall validate that type parameters implement the respective option interfaces (`inject.Option` or `provide.Option`)

### Requirement 2: Inject Option Types

**Objective:** As a braider user, I want to configure Injectable annotations with option types, so that I can control constructor generation and dependency registration behavior.

#### Acceptance Criteria

1. The braider inject package shall provide a `Default` option interface for default Injectable behavior
2. The braider inject package shall provide a `Typed[T any]` option interface for interface-based type registration
3. The braider inject package shall provide a `Named[T namer.Namer]` option interface for named dependency registration
4. The braider inject package shall provide a `WithoutConstructor` option interface to skip constructor generation
5. When `Injectable[inject.Default]` is used, the braider analyzer shall generate a constructor with pointer-to-struct return type
6. When `Injectable[inject.Typed[I]]` is used, the braider analyzer shall register the dependency as interface type `I` instead of concrete struct type
7. When `Injectable[inject.Named[N]]` is used, the braider analyzer shall register the dependency with the name returned by `N.Name()`
8. When `Injectable[inject.WithoutConstructor]` is used, the braider analyzer shall skip constructor generation and require a manually-provided `New<TypeName>` function

### Requirement 3: Provide Option Types

**Objective:** As a braider user, I want to configure Provide annotations with option types, so that I can specify the registration type and name for provider functions.

#### Acceptance Criteria

1. The braider provide package shall provide a `Default` option interface for default Provide behavior
2. The braider provide package shall provide a `Typed[T any]` option interface for specifying provider return type registration
3. The braider provide package shall provide a `Named[T namer.Namer]` option interface for named provider registration
4. When `Provide[provide.Default](fn)` is used, the braider analyzer shall register the provider function with its declared return type
5. When `Provide[provide.Typed[I]](fn)` is used, the braider analyzer shall register the provider function as returning interface type `I`
6. When `Provide[provide.Named[N]](fn)` is used, the braider analyzer shall register the provider with the name returned by `N.Name()`
7. The braider analyzer shall validate that provider function return types are compatible with the `Typed[T]` parameter type

### Requirement 4: Named Dependencies Support

**Objective:** As a braider user, I want to register and resolve dependencies by name, so that I can distinguish between multiple instances of the same type.

#### Acceptance Criteria

1. The braider namer package shall provide a `Namer` interface with a `Name() string` method
2. When a dependency is annotated with `Named[N]`, the braider analyzer shall invoke `N.Name()` at analysis time to extract the name string
3. The braider analyzer shall require that `Namer.Name()` implementations return hardcoded string literals (not computed values)
4. When generating bootstrap code, the braider analyzer shall use variable names derived from dependency names for named dependencies
5. If multiple dependencies have the same name and type, the braider analyzer shall report a diagnostic error

### Requirement 5: Option Interface Extensibility

**Objective:** As a braider user, I want to create custom mixed-in option types, so that I can combine multiple option behaviors in a single annotation.

#### Acceptance Criteria

1. The braider inject package shall define a base `Option` interface with `isOption() option` method
2. The braider provide package shall define a base `Option` interface with `isOption() option` method
3. When a custom type implements multiple option interfaces (e.g., `Typed[T]` and `Named[N]`), the braider analyzer shall apply all option behaviors
4. The braider analyzer shall validate that custom option types satisfy the base `Option` interface constraint
5. When conflicting options are detected (e.g., both `Default` and `WithoutConstructor`), the braider analyzer shall report a diagnostic error

### Requirement 6: Constructor Generation with Options

**Objective:** As a braider analyzer developer, I want to generate constructors based on option types, so that generated code reflects user-specified customization.

#### Acceptance Criteria

1. When `Injectable[inject.Default]` is detected, the braider analyzer shall generate a constructor returning `*StructType`
2. When `Injectable[inject.Typed[I]]` is detected, the braider analyzer shall generate a constructor returning interface type `I`
3. When `Injectable[inject.Named[N]]` is detected, the braider analyzer shall generate constructor parameter names reflecting dependency names
4. When `Injectable[inject.WithoutConstructor]` is detected, the braider analyzer shall not emit a constructor SuggestedFix
5. The braider analyzer shall validate that all injectable field types can be resolved from the dependency graph

### Requirement 7: Bootstrap Code Generation with Typed Dependencies

**Objective:** As a braider analyzer developer, I want to generate bootstrap IIFE code with interface-typed variables, so that generated main functions use interface types instead of concrete structs.

#### Acceptance Criteria

1. When `Injectable[inject.Typed[I]]` is registered, the braider bootstrap generator shall declare variables with interface type `I` instead of `*StructType`
2. When `Provide[provide.Typed[I]]` is registered, the braider bootstrap generator shall assign provider function results to interface-typed variables
3. The braider bootstrap generator shall include proper type assertions or casts if needed for interface assignments
4. When both typed and untyped dependencies coexist, the braider bootstrap generator shall handle both patterns correctly in topological order
5. The braider bootstrap generator shall validate at analysis time that all interface types are satisfied by concrete implementations

### Requirement 8: Diagnostic Error Messages

**Objective:** As a braider user, I want clear error messages when annotation options are misconfigured, so that I can quickly identify and fix issues.

#### Acceptance Criteria

1. When `Injectable[T]` type parameter does not implement `inject.Option`, the braider analyzer shall report an error with the constraint violation details
2. When `Provide[T]` type parameter does not implement `provide.Option`, the braider analyzer shall report an error with the constraint violation details
3. When `Namer.Name()` returns a non-literal value, the braider analyzer shall report an error indicating that names must be hardcoded strings
4. When duplicate named dependencies are detected, the braider analyzer shall report an error with the conflicting dependency names and locations
5. When individual option validation errors occur (AC 1-3), the AppAnalyzer shall cancel processing and skip bootstrap code generation
6. When correlation check errors occur (AC 4), the AppAnalyzer shall report the error but continue processing

### Requirement 9: Documentation and Examples

**Objective:** As a braider user, I want comprehensive documentation and examples for the new annotation options, so that I can adopt them in my projects effectively.

#### Acceptance Criteria

1. The braider annotation package shall include godoc comments for `Injectable[T]` and `Provider[T]` interfaces with usage examples
2. The braider inject package shall include godoc comments for each option interface (`Default`, `Typed[T]`, `Named[T]`, `WithoutConstructor`) with code examples
3. The braider provide package shall include godoc comments for each option interface (`Default`, `Typed[T]`, `Named[T]`) with code examples
4. The braider repository shall include example projects demonstrating interface-typed dependencies, named dependencies, and custom constructors
5. The braider README shall document the option pattern and link to detailed examples

