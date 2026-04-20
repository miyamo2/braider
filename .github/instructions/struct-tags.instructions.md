---
applyTo: "pkg/annotation/**,internal/detect/**"
---

# `braider` Struct Tags

_Read this when: working on field-level DI customization, constructor generation for Injectable structs, container field resolution, or diagnostics around tag conflicts._

## Out of Scope

Topics under this instruction's `applyTo:` that are handled elsewhere (shares `pkg/annotation/**` and `internal/detect/**` with `annotations.instructions.md`; this file owns tag semantics only):

- Annotation types themselves (`Injectable` / `Provide` / `Variable` / `App`) → see `annotations.instructions.md`
- Option types (`inject.*` / `provide.*` / `variable.*` / `app.*`) → see `annotations.instructions.md`
- AST-level details of constructor generation derived from tag decisions → see `code-generation.instructions.md`

## Purpose

`braider:"..."` struct tags control field-level DI behavior on:

- `Injectable[T]` struct fields
- `app.Container[T]` struct fields (with stricter rules)

## Injectable Struct Field Tags

| Tag | Behavior |
|---|---|
| `braider:"name"` | Matches a named dependency with name `name` (equivalent to `Named[N]` at the field level) |
| `braider:"-"` | Excludes the field from DI entirely (skipped during constructor generation and dependency resolution) |
| _no tag_ | Matches by type (default behavior; concrete or interface) |
| `braider:""` | Empty tag emits a diagnostic error (ambiguous intent) |

### Examples

```go
type Service struct {
    annotation.Injectable[inject.Default]
    PrimaryDB *DB   `braider:"primaryDB"` // match named dep "primaryDB"
    Cache     Cache                       // match by type
    Logger    *log.Logger `braider:"-"`   // excluded from DI, not set by constructor
}
```

## Container Field Tags (`app.Container[T]`)

Container fields use the same `braider` struct tag convention, with **stricter rules**:

| Tag | Behavior |
|---|---|
| `braider:"name"` | Resolve the field to a named dependency |
| _no tag_ | Resolve by type (concrete or interface) |
| `braider:"-"` | **Not permitted** — emits diagnostic error |
| `braider:""` | **Not permitted** — emits diagnostic error |

## Conflict Detection

The following emit diagnostic errors:

- `braider:"name"` on a field whose type is already registered with a **different** name
- Using `braider` tag on a field of an Injectable with `inject.Named[N]` option (option and tag conflict)
- Empty tag `braider:""`
- `braider:"-"` on a container field

## Processing

Tag interpretation lives in `internal/detect/` (FieldAnalyzer / ConstructorAnalyzer). Conflicting tags are reported via the `CategoryOptionValidation` diagnostic category (Critical severity; aborts pipeline).
