# Research & Design Decisions

---
**Purpose**: Capture discovery findings, architectural investigations, and rationale that inform the technical design for `app-with-container-def`.
---

## Summary
- **Feature**: `app-with-container-def`
- **Discovery Scope**: Extension (adds container-based bootstrap to existing AppAnalyzer pipeline)
- **Key Findings**:
  - The generic `annotation.App[T](main)` form produces `*ast.IndexExpr` wrapping `*ast.SelectorExpr` in the call expression's `Fun` field; the current `isAppCall` only handles `*ast.SelectorExpr` directly and must be extended.
  - The `AppContainer` marker interface already exists in `internal/annotation/app.go` and `app.Container[T]` in `pkg/annotation/app/options.go`; detection can leverage the existing `go/types` type-checking infrastructure to match against `annotation.AppContainer`.
  - Hash computation must incorporate container field definitions (field names, types, and struct tags) to ensure idempotent regeneration detects container struct changes.

## Research Log

### How Go AST Represents Generic Function Calls

- **Context**: Need to detect `annotation.App[app.Container[T]](main)` in the AST. The current `isAppCall` assumes `call.Fun` is `*ast.SelectorExpr`.
- **Sources Consulted**: Go specification for generic instantiation expressions; `go/ast` package documentation for `IndexExpr` and `IndexListExpr`.
- **Findings**:
  - A generic function call `f[T](args)` is represented as `*ast.CallExpr` where `Fun` is `*ast.IndexExpr{X: selectorExpr, Index: typeArgExpr}`.
  - A generic function call with multiple type args `f[T1, T2](args)` uses `*ast.IndexListExpr`. Since `App[T]` has a single type parameter, only `*ast.IndexExpr` is relevant.
  - The selector `annotation.App` is nested inside `IndexExpr.X`.
- **Implications**: `isAppCall` must unwrap `*ast.IndexExpr` to reach the underlying `*ast.SelectorExpr`, and then extract the type argument from `IndexExpr.Index` for option detection.

### Container Type Parameter Extraction via go/types

- **Context**: Once the generic call is detected, the type parameter `T` in `App[T]` must be inspected to determine if it implements `AppContainer`.
- **Sources Consulted**: `go/types` documentation for `types.Named.TypeArgs()`, interface implementation checking via `types.Implements`.
- **Findings**:
  - `pass.TypesInfo.TypeOf(callExpr)` returns the instantiated return type `app[T]`, from which `T` can be extracted via `TypeArgs().At(0)`.
  - To check if `T` implements `AppContainer`, resolve the `AppContainer` interface type from the `internal/annotation` package and use `types.Implements(T, appContainerInterface)`.
  - For `app.Container[T]`, the inner type parameter `T` (the user's struct) is accessible by further unwrapping via `TypeArgs()` on the `Container` named type.
  - Anonymous struct type parameters are represented as `*types.Struct` in the underlying type, while named struct types are `*types.Named` with `*types.Struct` underlying.
- **Implications**: The existing `OptionExtractor` pattern (extracting type args from named types) can be extended for App options. A new `AppOptionExtractor` or extension of the existing `OptionExtractor` interface handles this.

### Container Struct Field Resolution Strategy

- **Context**: Container struct fields must be matched to registered dependencies by type (and optionally by name via `braider` struct tag).
- **Sources Consulted**: Existing `FieldAnalyzer` and `StructDetector` patterns in `internal/detect/`.
- **Findings**:
  - For named structs, field information is available via `types.Named.Underlying().(*types.Struct)` and field-level struct tags via `types.Struct.Tag(i)`.
  - For anonymous structs (inline type parameter), the `*types.Struct` is directly available from the type argument.
  - AST-level struct tag extraction is needed for anonymous structs defined inline in the type parameter, since `types.Struct.Tag(i)` gives access to tags.
  - Resolution algorithm: For each field, check `braider` struct tag first (named resolution), then fall back to type-based resolution using the same `InterfaceRegistry` and graph node lookup that the default bootstrap uses.
- **Implications**: A new `ContainerResolver` component handles field-to-dependency mapping, reusing the `InterfaceRegistry` for interface resolution.

### Hash Computation Extension

- **Context**: Requirement 5.3 specifies that graph hash must include container struct field definitions.
- **Sources Consulted**: Existing `ComputeGraphHash` in `internal/generate/hash.go`.
- **Findings**:
  - Current hash inputs: TypeName, ConstructorName, IsField, Dependencies, ExpressionText.
  - Container field definitions (field name, field type string, struct tag) must be appended to the hash input to detect container struct changes.
  - The container definition is separate from the dependency graph nodes -- it defines the "view" of the graph. Changes to the container (e.g., reordering fields, adding/removing fields) must invalidate the hash even if the underlying dependency graph is unchanged.
- **Implications**: `ComputeGraphHash` receives an optional container definition parameter. When present, container fields are hashed after graph nodes.

## Architecture Pattern Evaluation

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|---------------------|-------|
| Extend existing pipeline | Add container detection and resolution as new steps in existing AppAnalyzer.Run() | Minimal disruption, reuses all existing infrastructure | AppAnalyzer.Run() grows in complexity | Selected approach |
| Separate ContainerAnalyzer | New analyzer specifically for container-based bootstrap | Clean separation of concerns | Requires coordination between three analyzers instead of two; duplicates graph building logic | Rejected: unnecessary complexity |
| Strategy pattern in BootstrapGenerator | BootstrapGenerator selects strategy (default vs container) based on detected option | Clean interface, testable | Adds abstraction layer for two variants | Incorporated into selected approach |

## Design Decisions

### Decision: Extend AppDetector to Handle Generic Form

- **Context**: `isAppCall` currently only matches non-generic `annotation.App(main)` via `*ast.SelectorExpr`. The generic form `annotation.App[T](main)` wraps the selector in `*ast.IndexExpr`.
- **Alternatives Considered**:
  1. Create a separate detection method for generic calls
  2. Extend `isAppCall` to unwrap `IndexExpr` transparently
- **Selected Approach**: Extend `isAppCall` to unwrap `*ast.IndexExpr` and extract the selector from `IndexExpr.X`. Add a new field to `AppAnnotation` to carry the extracted option type.
- **Rationale**: Keeps detection logic in one place; the unwrapping is a simple two-line addition.
- **Trade-offs**: Slightly more complex `isAppCall`, but avoids code duplication.
- **Follow-up**: Update unit tests to cover generic form detection.

### Decision: New ContainerDefinition Data Structure

- **Context**: Container struct field definitions must be carried from detection through validation, graph filtering, code generation, and hash computation.
- **Alternatives Considered**:
  1. Pass container info as part of `AppAnnotation`
  2. Create a separate `ContainerDefinition` struct passed alongside the graph
- **Selected Approach**: Add `ContainerDef *ContainerDefinition` field to `AppAnnotation` and create a `ContainerDefinition` struct in `internal/detect/` that carries field definitions (name, type, struct tag, resolved dependency key).
- **Rationale**: Container definition is intrinsically tied to the App annotation; keeping them together avoids parameter explosion in the pipeline.
- **Trade-offs**: `AppAnnotation` grows, but the additional field is nil for default App behavior, so no impact on existing code paths.

### Decision: BootstrapGenerator Strategy for Container vs Default

- **Context**: Bootstrap code generation differs between default (anonymous struct with all dependencies) and container (user-defined struct with selected dependencies).
- **Alternatives Considered**:
  1. Add conditional logic inside existing `GenerateBootstrap`
  2. Create separate `GenerateContainerBootstrap` method
- **Selected Approach**: Add `GenerateContainerBootstrap` method to `BootstrapGenerator` interface. The `AppAnalyzeRunner` selects which method to call based on the detected container definition.
- **Rationale**: Keeps each generation path clean and testable independently. The default path remains unchanged, minimizing regression risk.
- **Trade-offs**: Two generation methods instead of one, but each is simpler than a combined method with branching.

## Risks & Mitigations
- **Risk**: Anonymous struct type parameters with complex field types (e.g., generic types, function types) may be difficult to render back to source code for the return type literal. **Mitigation**: Use `types.TypeString` with a custom qualifier function for consistent rendering; restrict supported field types to concrete and interface types.
- **Risk**: Container struct field order in generated code may not match user expectation. **Mitigation**: Preserve field order as defined in the container struct definition (iteration order of `types.Struct` fields).
- **Risk**: Hash instability if `types.TypeString` output changes between Go versions. **Mitigation**: Use the same `types.TypeString` call consistently; hash includes field names and tags as additional anchors.

## References
- [Go AST specification - IndexExpr](https://pkg.go.dev/go/ast#IndexExpr) -- represents generic instantiation `f[T]`
- [go/types - Named.TypeArgs](https://pkg.go.dev/go/types#Named.TypeArgs) -- accessing type arguments of instantiated types
- [go/types - Implements](https://pkg.go.dev/go/types#Implements) -- checking interface implementation
- [go/types - Struct](https://pkg.go.dev/go/types#Struct) -- struct type representation with fields and tags
