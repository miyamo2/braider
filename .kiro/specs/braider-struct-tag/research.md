# Research & Design Decisions

---
**Purpose**: Capture discovery findings, architectural investigations, and rationale that inform the technical design for the `braider-struct-tag` feature.

**Usage**:
- Log research activities and outcomes during the discovery phase.
- Document design decision trade-offs that are too detailed for `design.md`.
- Provide references and evidence for future audits or reuse.
---

## Summary
- **Feature**: `braider-struct-tag`
- **Discovery Scope**: Extension
- **Key Findings**:
  - The `FieldAnalyzer` component is the natural extension point for struct tag parsing, requiring changes to `FieldInfo` and `AnalyzeFields` to propagate tag metadata.
  - Go's `reflect.StructTag` provides standard parsing conventions but operates on string values, whereas braider must parse from `ast.Field.Tag` (`*ast.BasicLit`). The design uses `reflect.StructTag.Lookup()` after extracting the raw tag string from the AST.
  - The `Dependencies` field in `InjectorInfo` currently stores fully qualified type names. Struct tag-derived named dependencies require encoding as composite keys (`TypeName#Name`) at registration time, which aligns with the existing graph edge resolution pattern in `DependencyGraphBuilder.resolveDependency`.

## Research Log

### Go Struct Tag Parsing from AST
- **Context**: Need to parse `braider:"name"` and `braider:"-"` from `ast.Field.Tag` during static analysis.
- **Sources Consulted**: Go `reflect` package documentation, `go/ast` package documentation.
- **Findings**:
  - `ast.Field.Tag` is a `*ast.BasicLit` of kind `token.STRING`. The value includes surrounding backticks or quotes.
  - `reflect.StructTag` expects the raw tag string without surrounding quotes. `strconv.Unquote` or backtick stripping is required before calling `reflect.StructTag.Lookup("braider")`.
  - `reflect.StructTag.Lookup(key)` returns `(value, ok)` where `ok` distinguishes between absent tag and empty value. This supports requirement 3.2 (empty value validation) and 3.3 (absent tag default behavior).
- **Implications**: A new `StructTagParser` component in `internal/detect/` handles AST-to-StructTag conversion and validation. The parser returns a `StructTagInfo` value containing the parsed tag state (absent, excluded, or named).

### Dependency Resolution with Named Struct Tags
- **Context**: When a field has `braider:"name"`, the DependencyAnalyzer must resolve that field's dependency using the composite key `TypeName#name` instead of the plain `TypeName`.
- **Sources Consulted**: Existing `DependencyGraphBuilder.resolveDependency`, `makeNodeKey` function, Named[N] option implementation.
- **Findings**:
  - The graph already uses composite keys (`TypeName#Name`) for Named[N] dependencies. The `resolveDependency` method first checks composite keys in the graph.
  - Current `InjectorInfo.Dependencies` stores plain type names. For struct tag named fields, the dependency must be stored as `TypeName#name` to create the correct graph edge.
  - The `FieldAnalyzer.AnalyzeFields` currently returns `[]FieldInfo` without tag metadata. The `FieldInfo` struct must be extended with a `NamedDependency` field to carry the parsed struct tag name.
- **Implications**: `FieldInfo` gains a `NamedDependency string` field. When building `InjectorInfo.Dependencies` in Phase 3 of `DependencyAnalyzeRunner.Run`, the dependency string is constructed as `TypeName#NamedDependency` when `NamedDependency` is non-empty.

### Constructor Generation with Named Dependencies
- **Context**: Generated constructors must use appropriate parameter names when a field has a `braider:"name"` tag.
- **Sources Consulted**: Existing `GenerateConstructorWithNamedDeps` method.
- **Findings**:
  - `GenerateConstructorWithNamedDeps` already supports `dependencyNames map[string]string` to override parameter names. This method is already defined in the `ConstructorGenerator` interface.
  - The `dependencyNames` map keys are field names, values are custom parameter names. For struct tag named fields, the map entry is `{fieldName: tagName}`.
  - Phase 1 of `DependencyAnalyzeRunner.Run` currently uses `GenerateConstructor` (no named deps support). It must switch to `GenerateConstructorWithNamedDeps` when any field has a struct tag name.
- **Implications**: The existing `GenerateConstructorWithNamedDeps` method serves as the generation path for struct tag-aware constructors. No new generator method is needed.

### Interaction with WithoutConstructor Option
- **Context**: Requirement 4.3 specifies that `braider` struct tags must not conflict with existing constructor signatures when `inject.WithoutConstructor` is used.
- **Sources Consulted**: `GenerateConstructorWithOptions`, `GenerateConstructorWithNamedDeps` methods, `InjectorInfo.OptionMetadata`.
- **Findings**:
  - When `WithoutConstructor` is set, constructor generation is skipped (returns nil). The existing constructor's parameter list defines the dependency set.
  - `ConstructorAnalyzer.ExtractDependencies` extracts dependencies from existing constructor parameters by type. It does not have field-level struct tag awareness.
  - For `WithoutConstructor` structs, `braider:"-"` on a field that the existing constructor accepts creates a conflict: the field is excluded from DI but the constructor still expects it.
  - For `WithoutConstructor` structs, `braider:"name"` on a field that the constructor does not accept as a parameter creates a conflict: the field requests a named dependency but no constructor parameter matches.
- **Implications**: Validation in `DependencyAnalyzeRunner.Run` Phase 3 must cross-check struct tag metadata against existing constructor parameters when `WithoutConstructor` is active. A new diagnostic error type is needed.

### Hash Computation Impact
- **Context**: Requirement 5.3 requires that changes to `braider` struct tags trigger bootstrap regeneration.
- **Sources Consulted**: `ComputeGraphHash` function, hash input fields.
- **Findings**:
  - Current hash inputs: `TypeName`, `ConstructorName`, `IsField`, `ExpressionText`, `Dependencies` (sorted).
  - Struct tag changes affect the `Dependencies` list (composite keys with `#Name` suffix) and `ConstructorName` (unchanged for tags). When a field gains or loses a `braider:"name"` tag, its dependency changes from `TypeName` to `TypeName#name` (or vice versa), which changes the hash.
  - When a field gains `braider:"-"`, it is removed from dependencies entirely, which also changes the hash.
- **Implications**: No modification to `ComputeGraphHash` is needed. The existing hash inputs naturally capture struct tag changes through their effect on the `Dependencies` list.

## Architecture Pattern Evaluation

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|---------------------|-------|
| Extend FieldAnalyzer | Add struct tag parsing to existing FieldAnalyzer | Minimal component count, natural extension point | Increased responsibility for FieldAnalyzer | Selected approach |
| New StructTagDetector | Separate component for tag parsing | Single responsibility, independent testing | Requires additional wiring in DependencyAnalyzer, indirect coupling with FieldAnalyzer | Considered but rejected: tag info must be correlated per-field |
| Inline in DependencyAnalyzer | Parse tags directly in Run() method | No new interfaces | Violates single responsibility, harder to test | Rejected |

## Design Decisions

### Decision: Extend FieldInfo Rather Than Create Separate Tag Result

- **Context**: Struct tag metadata must be associated with each field for use in constructor generation and dependency resolution.
- **Alternatives Considered**:
  1. Add `NamedDependency` and `Excluded` fields to `FieldInfo` -- single return value, direct correlation.
  2. Return separate `map[string]StructTagInfo` from a new detector -- requires post-join logic in callers.
- **Selected Approach**: Extend `FieldInfo` with `NamedDependency string` and `Excluded bool` fields. The `FieldAnalyzer` populates these during `AnalyzeFields`.
- **Rationale**: Tag metadata is inherently per-field. Embedding it in `FieldInfo` avoids correlation complexity and matches the existing pattern where `FieldInfo` carries all field-relevant data.
- **Trade-offs**: Slightly increases `FieldInfo` size; acceptable given the struct is short-lived (not stored in registries).
- **Follow-up**: Ensure `FieldAnalyzer` tests cover tag parsing edge cases (empty value, absent tag, multi-tag fields).

### Decision: Reuse GenerateConstructorWithNamedDeps for Tag-Aware Generation

- **Context**: Constructor generation must adjust parameter names when fields have `braider:"name"` tags.
- **Alternatives Considered**:
  1. Create new `GenerateConstructorWithStructTags` method.
  2. Reuse existing `GenerateConstructorWithNamedDeps` by building the `dependencyNames` map from struct tag info.
- **Selected Approach**: Reuse `GenerateConstructorWithNamedDeps`. Build `dependencyNames` map from `FieldInfo.NamedDependency` values in `DependencyAnalyzeRunner`.
- **Rationale**: No new code generation logic is required. The existing method already handles custom parameter names per field. Adding a new method would duplicate functionality.
- **Trade-offs**: Slight coupling between struct tag metadata and named-deps generation path; acceptable since the semantics are identical.

### Decision: Use FieldAnalyzer for Struct Tag Parsing (Not Separate Component)

- **Context**: Struct tags are read from `ast.Field.Tag`, which is available during field analysis.
- **Alternatives Considered**:
  1. New `StructTagParser` interface in `detect/` -- separate concern, but requires passing `*ast.Field` separately.
  2. Parse directly in `FieldAnalyzer.AnalyzeFields` -- natural location, access to `ast.Field.Tag`.
- **Selected Approach**: Add a `parseStructTag` helper method to `fieldAnalyzer` called during `AnalyzeFields`. This helper uses `reflect.StructTag.Lookup("braider")` after extracting the raw tag string.
- **Rationale**: The tag value is a property of the field, read at the same time as type and name. Keeping parsing in `FieldAnalyzer` maintains cohesion.
- **Trade-offs**: Adds tag parsing responsibility to `FieldAnalyzer`; mitigated by keeping the parsing logic in a focused helper method.

### Decision: Validate WithoutConstructor Conflicts in Phase 3

- **Context**: When `inject.WithoutConstructor` is active, struct tags that conflict with the existing constructor must produce diagnostics.
- **Selected Approach**: In Phase 3 of `DependencyAnalyzeRunner.Run`, after extracting struct tag metadata from fields, cross-validate against `ConstructorAnalyzer.ExtractDependencies`. Emit diagnostic errors for conflicts.
- **Rationale**: Phase 3 already re-detects injectors and extracts dependencies. Adding validation here consolidates tag-related checks in one location.
- **Trade-offs**: Phase 3 complexity increases slightly; mitigated by clear validation helper functions.

## Risks & Mitigations
- **Risk**: Struct tag parsing from `ast.BasicLit` may fail for unusual tag formats (e.g., concatenated tags, non-backtick strings). **Mitigation**: Use `reflect.StructTag.Lookup` which handles standard Go struct tag conventions. Emit diagnostic for unparseable tags.
- **Risk**: Named dependencies via struct tags may collide with Named[N] option dependencies. **Mitigation**: Both use the same composite key format (`TypeName#Name`), so the graph resolution mechanism handles them uniformly. The tag-derived name and the Named[N]-derived name are functionally equivalent.
- **Risk**: Adding `Excluded` field to `FieldInfo` may cause regressions in existing field analysis callers. **Mitigation**: Default value of `false` preserves existing behavior. Only new code paths check `Excluded`.

## References
- [Go reflect.StructTag documentation](https://pkg.go.dev/reflect#StructTag) -- standard struct tag parsing API
- [Go ast.Field.Tag](https://pkg.go.dev/go/ast#Field) -- AST representation of struct field tags
- [braider existing Named[N] implementation](internal/detect/option_extractor.go) -- composite key pattern for named dependencies
