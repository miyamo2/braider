# Research & Design Decisions

---
**Purpose**: Capture discovery findings, architectural investigations, and rationale that inform the technical design for the refine-annotation feature.

**Usage**:
- Log research activities and outcomes during the discovery phase.
- Document design decision trade-offs that are too detailed for `design.md`.
- Provide references and evidence for future audits or reuse.
---

## Summary
- **Feature**: `refine-annotation`
- **Discovery Scope**: Extension (extending existing annotation system with generic type parameters)
- **Key Findings**:
  - Go generics support via `types.Named.TypeArgs()` and `types.Info.Instances` provides foundation for type parameter extraction
  - Current annotation detection relies on `types.Named` type checking, easily extensible to generic types
  - Option pattern using interface constraints is idiomatic Go for generic customization
  - Registry structure requires extension to support typed and named dependencies

## Research Log

### Go Generics Type Parameter Extraction

- **Context**: Requirements 1-3 specify generic annotation interfaces with type parameters (`Injectable[T]`, `Provider[T]`). Need to understand how to extract type arguments from generic instantiations during AST analysis.

- **Sources Consulted**:
  - [Basic Go tooling for generics - Eli Bendersky](https://eli.thegreenplace.net/2022/basic-go-tooling-for-generics/)
  - [go/types Package Documentation](https://pkg.go.dev/go/types)
  - [Go Programming Language Specification](https://tip.golang.org/ref/spec)

- **Findings**:
  - `types.Named.TypeArgs()` returns `*types.TypeList` containing instantiated type arguments
  - `types.Info.Instances` maps AST identifiers to their generic instantiations
  - For embedded fields with generic types, `pass.TypesInfo.TypeOf(field.Type)` returns `*types.Named` with type arguments accessible
  - Type parameters must be iterated via `TypeList.Len()` and `TypeList.At(i)`
  - Type constraint validation requires checking if type argument implements constraint interface via `types.Implements()`

- **Implications**:
  - Existing `InjectDetector.isNamedInjectType()` needs extension to support `Injectable[T]` interface types
  - New `OptionExtractor` component required to extract type parameter `T` from `Injectable[T]` and `Provider[T]`
  - Option type validation must check `T implements inject.Option` or `provide.Option` constraint

### Interface Constraint Pattern for Options

- **Context**: Requirements 2-3 specify option types as interfaces (e.g., `inject.Default`, `inject.Typed[I]`, `inject.Named[N]`). Need idiomatic Go pattern for extensible option system.

- **Sources Consulted**:
  - [How to Use Generics with Type Constraints in Go](https://oneuptime.com/blog/post/2026-01-23-go-generics-constraints/view)
  - [Generic interfaces - Go Blog](https://go.dev/blog/generic-interfaces)
  - [Constraints in Go - Bitfield Consulting](https://bitfieldconsulting.com/posts/constraints)

- **Findings**:
  - Base constraint interface pattern (`Option interface { isOption() option }`) prevents external implementations
  - Marker methods (`isDefault()`, `typed()`, `named()`, `withoutConstructor()`) provide compile-time type discrimination
  - Generic constraint interfaces like `Typed[T any]` allow type-safe parameterization
  - Nested generics `Named[T namer.Namer]` enable chained constraints

- **Implications**:
  - Option interfaces already defined in `pkg/annotation/inject/options.go` and `pkg/annotation/provide/options.go` follow Go best practices
  - Type parameter extraction enables runtime type switch on option interfaces
  - No changes needed to existing option interface definitions

### Named Dependencies and Namer Interface

- **Context**: Requirement 4 specifies named dependency support via `Namer` interface with hardcoded string literals.

- **Sources Consulted**:
  - Existing `pkg/annotation/namer/namer.go` implementation
  - Go AST constant expression evaluation patterns

- **Findings**:
  - `Namer.Name() string` interface already defined
  - Hardcoded literal requirement necessitates AST-level validation of `Name()` method body
  - Requires traversing function declaration AST to verify return statement contains `*ast.BasicLit` with `STRING` token

- **Implications**:
  - New `NamerValidator` component required for static analysis of `Name()` method implementations
  - Validation must occur during dependency registration phase
  - Error diagnostic must report non-literal usage with clear guidance

### Registry Extension for Typed and Named Dependencies

- **Context**: Requirements 6-7 specify different registration behavior based on option types. Current `InjectorRegistry` and `ProviderRegistry` store only concrete type names.

- **Sources Consulted**:
  - Existing `internal/registry/injector_registry.go` and `provider_registry.go`
  - Requirements for interface-typed dependencies and named dependencies

- **Findings**:
  - Current `InjectorInfo.TypeName` stores fully qualified concrete type
  - `Implements` field exists but unused for registration key
  - Named dependencies require composite key: `(typeName, name)`
  - Interface-typed dependencies require registration under interface type instead of concrete type

- **Implications**:
  - `InjectorInfo` requires new fields: `RegisteredType` (interface or concrete), `Name` (optional), `Options` (parsed option metadata)
  - `ProviderInfo` requires same extensions
  - Registry key must become composite: `typeName` remains primary key, new `NamedDependencies` map for name-based lookups
  - Bootstrap generator must use `RegisteredType` for variable declarations instead of concrete type

### Constructor and Bootstrap Generation Changes

- **Context**: Requirements 6-7 specify different constructor return types and bootstrap variable types based on options.

- **Findings**:
  - `inject.Default` → constructor returns `*ConcreteType`
  - `inject.Typed[I]` → constructor returns interface type `I`
  - `inject.Named[N]` → bootstrap variable name derived from `N.Name()`
  - `inject.WithoutConstructor` → skip constructor generation, validate existing `New<Type>` function
  - `provide.Typed[I]` → bootstrap variable type is interface `I` instead of concrete type

- **Implications**:
  - `ConstructorGenerator` requires option-aware return type selection
  - `BootstrapGenerator` requires option-aware variable type and naming
  - Constructor existence validation required for `WithoutConstructor` option
  - Type compatibility validation required for `Typed[I]` options (verify concrete type implements interface)

## Architecture Pattern Evaluation

### Option Extraction Strategy

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|---------------------|-------|
| AST-based type switch | Extract type parameters from `*ast.IndexExpr`, match on type names | Fast, works with untyped AST | Fragile to refactoring, no type safety | Rejected: requires string matching |
| Type-based reflection | Use `types.Named.TypeArgs()` + interface assertion | Type-safe, robust to renaming | Requires full type info, complex type switches | **Selected**: aligns with existing type checking pattern |
| Dedicated option struct | Users pass option struct instead of type parameter | Simple to implement | Breaks type safety, verbose usage | Rejected: defeats purpose of generics |

### Namer Validation Strategy

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|---------------------|-------|
| Require constant declaration | Force users to declare `const MyName = "name"` | Simple AST check | Breaks encapsulation, awkward API | Rejected: poor UX |
| Runtime name extraction | Extract name at bootstrap time via reflection | No static validation | Errors appear at runtime, breaks static analysis | Rejected: defeats analyzer purpose |
| AST method body analysis | Parse `Name()` method body, validate return is `*ast.BasicLit` | Static validation, clear errors | Complex implementation | **Selected**: aligns with analyzer philosophy |

## Design Decisions

### Decision: Generic Interface vs. Struct-Based Annotations

- **Context**: Current annotations use struct embedding (`annotation.Inject` struct). Generic annotations require interface types (`annotation.Injectable[T]` interface).

- **Alternatives Considered**:
  1. Keep struct embedding, add separate `annotation.Options` field — maintains compatibility but verbose
  2. Switch to generic interface embedding — clean API, requires migration
  3. Support both patterns with deprecation period — gradual migration, higher maintenance

- **Selected Approach**: Generic interface embedding (`Injectable[T]`, `Provider[T]`) with backward compatibility shim

- **Rationale**:
  - Generic interfaces provide type-safe option configuration at compile time
  - Interface constraint pattern is idiomatic Go (follows stdlib patterns like `comparable`, `any`)
  - Existing code already uses `Injectable[inject.Default]` in annotation.go godoc examples
  - Backward compatibility achievable via type alias: `type Inject = Injectable[inject.Default]`

- **Trade-offs**:
  - **Benefits**: Type safety, clean API, extensibility, compile-time validation
  - **Compromises**: Breaking change if `Inject` struct used directly (mitigated by alias)

- **Follow-up**: Verify all test fixtures use new `Injectable[T]` syntax; add migration guide to documentation

### Decision: Option Type Extraction via TypeArgs vs. Marker Methods

- **Context**: Need to determine which option interfaces are applied to extract behavior configuration.

- **Alternatives Considered**:
  1. Marker methods — call methods like `isDefault()` via reflection or codegen
  2. Type parameter extraction — analyze `T` in `Injectable[T]` to determine option types
  3. Dedicated option registry — users register options separately

- **Selected Approach**: Type parameter extraction via `types.Named.TypeArgs()` combined with interface implementation checking

- **Rationale**:
  - Type parameters are available at compile time via `go/types`
  - Enables static analysis without runtime dependencies
  - Supports mixed-in options (single type implementing multiple option interfaces)
  - Aligns with existing type-based detection pattern in `InjectDetector`

- **Trade-offs**:
  - **Benefits**: Static analysis, no runtime overhead, supports composition
  - **Compromises**: Complex type checking logic for nested generics

- **Follow-up**: Implement comprehensive test coverage for mixed-in options and nested generics

### Decision: Registry Key Structure for Named Dependencies

- **Context**: Named dependencies allow multiple instances of same type with different names (Requirement 4). Need unique registry keys.

- **Alternatives Considered**:
  1. Composite key `(typeName, name)` — simple, supports unnamed and named
  2. Separate named registry — clear separation, harder to query
  3. Name suffix in TypeName — simple key, awkward string manipulation

- **Selected Approach**: Primary key remains `typeName`, add optional `Name` field and separate `NamedDependencies` index map

- **Rationale**:
  - Maintains backward compatibility with existing registry queries by `TypeName`
  - Enables efficient lookup by name when needed
  - Supports validation: detect duplicate `(type, name)` pairs
  - Clear data model: `Name == ""` means unnamed dependency

- **Trade-offs**:
  - **Benefits**: Backward compatible, efficient queries, clear semantics
  - **Compromises**: Slightly more complex registry implementation

- **Follow-up**: Add test cases for duplicate name detection and unnamed/named coexistence

### Decision: Constructor Return Type Determination

- **Context**: `inject.Typed[I]` requires constructor to return interface type `I` instead of `*ConcreteStruct` (Requirement 6.2).

- **Alternatives Considered**:
  1. Always return `*ConcreteStruct`, rely on interface assignment — breaks explicit typing
  2. Return interface type from constructor — type-safe, explicit
  3. Generate wrapper function returning interface — extra indirection

- **Selected Approach**: Constructor returns interface type `I` when `Typed[I]` option detected, `*ConcreteStruct` otherwise

- **Rationale**:
  - Explicit return types improve code readability
  - Enables interface-based dependency injection patterns
  - Aligns with google/wire's provider function pattern
  - Type checker validates compatibility automatically

- **Trade-offs**:
  - **Benefits**: Type safety, explicit contracts, familiar pattern
  - **Compromises**: Constructor signature varies by option (mitigated by clear naming convention)

- **Follow-up**: Validate that concrete type implements interface during analysis; report diagnostic if not

### Decision: Error Handling Strategy for Option Validation

- **Context**: Requirement 8.5-8.6 specifies different error handling for validation vs. correlation errors.

- **Alternatives Considered**:
  1. Fail fast on any error — simple, prevents bad state
  2. Collect all errors, report batch — better UX, complex state management
  3. Different severity levels — flexible, clear intent

- **Selected Approach**: Two-tier error handling: fatal errors (option validation) stop processing, non-fatal errors (correlation) report but continue

- **Rationale**:
  - Option validation errors indicate type system violations (e.g., `T` doesn't implement constraint) — cannot proceed safely
  - Correlation errors (duplicate names) are user errors but don't break analyzer invariants
  - Allows reporting multiple user errors in single pass when safe
  - Matches go/analysis best practices for diagnostic reporting

- **Trade-offs**:
  - **Benefits**: Better UX for correlation errors, maintains safety for type errors
  - **Compromises**: More complex error handling logic

- **Follow-up**: Document error handling tiers in implementation notes; test both scenarios

## Risks & Mitigations

- **Risk**: Complex generic type parameter extraction breaks on edge cases (deeply nested generics, type aliases) — **Mitigation**: Comprehensive test suite covering nested `Named[Typed[I]]` scenarios; fallback to diagnostic error for unsupported patterns
- **Risk**: Namer validation AST traversal fails to detect computed string concatenation — **Mitigation**: Conservative validation; reject non-obvious patterns with clear error; document literal-only requirement prominently
- **Risk**: Registry extension breaks existing code relying on internal registry structure — **Mitigation**: Internal package isolation; only `cmd/braider/main.go` constructs registries; add compatibility tests
- **Risk**: Constructor generation for `Typed[I]` produces invalid code if concrete type doesn't implement interface — **Mitigation**: Pre-validate interface implementation using `types.Implements()` before code generation; report diagnostic if incompatible
- **Risk**: Bootstrap generation order breaks for named dependencies with circular imports — **Mitigation**: Reuse existing topological sort; treat named dependencies as distinct nodes in dependency graph

## References

### Go Generics and Type Parameters
- [An Introduction To Generics - Go Blog](https://go.dev/blog/intro-generics) — Official generics introduction
- [Basic Go tooling for generics - Eli Bendersky](https://eli.thegreenplace.net/2022/basic-go-tooling-for-generics/) — Type parameter extraction patterns
- [go/types Package Documentation](https://pkg.go.dev/go/types) — Type checker API reference
- [Generic interfaces - Go Blog](https://go.dev/blog/generic-interfaces) — Interface design patterns for generics

### Type Constraints and Options
- [How to Use Generics with Type Constraints in Go](https://oneuptime.com/blog/post/2026-01-23-go-generics-constraints/view) — Constraint best practices
- [Constraints in Go - Bitfield Consulting](https://bitfieldconsulting.com/posts/constraints) — Practical constraint usage

### AST Analysis
- [go/ast Package Documentation](https://pkg.go.dev/go/ast) — AST structure reference
- [Exploring function parameter types with Go tooling](https://eli.thegreenplace.net/2022/exploring-function-parameter-types-with-go-tooling/) — Type extraction techniques
