---
applyTo: "internal/generate/**,internal/report/**"
---

# Code Generation

_Read this when: working on `internal/generate/`, constructor/bootstrap emission, AST builder helpers, import management, or hash-based idempotency._

## Out of Scope

Topics under this instruction's `applyTo:` that are handled elsewhere:

- When and in which phase generation is invoked (pipeline coordination) → see `architecture.instructions.md`
- Golden / checkertest validation workflow for emitted code → see `testing.instructions.md`
- Detection logic that feeds inputs into generation (annotation detection, tag parsing) → see `annotations.instructions.md` / `struct-tags.instructions.md`

## AST-Based Generation

Both constructor and bootstrap code are built as `go/ast` trees and rendered via `format.Node` — **not** string concatenation (`fmt.Sprintf`, `strings.Builder`).

### Benefits
- Produces correctly formatted Go code without a separate formatting pass
- Eliminates an entire component (`CodeFormatter`) from the dependency graph
- Makes structural code manipulation safer (no string interpolation bugs)

### Helpers (`internal/generate/ast_builder.go`)
Concise AST-node constructors:

- `astIdent`, `astSelector`
- `astStructType`
- `astShortVar`, `astFuncDecl`, `astVarDecl`
- plus related utilities

### Rendering
- **Declarations** via `renderDecl` — wraps in dummy file, assigns synthetic positions, strips package prefix
- **Expressions** via `renderNode`
- **Import blocks** via `RenderImportBlock` (shared by both `internal/generate/` and `internal/report/`)

## Constructor Generation

- Constructors always return **pointer types** (`*StructName`) — both generated and existing ones
- Zero-dependency structs also get constructors (no `HasInjectableFields` guard)
- Handled by `ConstructorGenerator` via `analysis.SuggestedFix`

## Bootstrap Generation

Produces an IIFE (immediately invoked function expression) returning either:

- An anonymous struct with all dependencies as fields (`app.Default`)
- A user-defined container struct (`app.Container[T]`)

### Struct field types
Concrete types appear as `*package.Type` (pointer).

### Bootstrap assembly order
Determined by `TopologicalSorter` (Kahn's algorithm with alphabetical tie-break for determinism).

### Field vs local variable
Controlled by `Node.IsField`:

- Injectable and Provide → fields in returned struct
- Variable → local variables inside IIFE

Variable nodes not depended on by anyone use `_ = expression` to avoid unused-variable errors.

## Hash-Based Idempotency

Bootstrap code embeds a `// braider:hash:<hash>` comment. On subsequent runs, if the computed hash matches the existing one, regeneration is skipped. This prevents unnecessary rewrites and preserves manual edits in unrelated code sections.

### Hash inputs (`ComputeGraphHash`)
- `TypeName`
- `ConstructorName`
- `IsField`
- `Dependencies`
- `ExpressionText`
- `ConstructorPkgPath` — included **only when** it differs from `PackagePath`

**Not** included: `RegisteredType`.

## Cross-Package Constructor Qualification

When a `Provide[T](fn)` registers a function that returns a type from a **different package** than where the function is defined, the bootstrap generator must use two separate package qualifiers:

- **Return type's package** (`PackagePath` / `PackageName`): struct field type qualification (e.g., `analysis.Analyzer`)
- **Constructor function's package** (`ConstructorPkgPath` / `ConstructorPkgName`): function call qualification (e.g., `analyzer.NewAppAnalyzer(...)`)

### Node fields
The `Node` struct carries both sets of fields:
- `PackagePath` / `PackageName`
- `ConstructorPkgPath` / `ConstructorPkgName` / `ConstructorPkgAlias`

Import collection and collision detection consider both packages.

`ConstructorPkgPath` is included in hash computation only when it differs from `PackagePath`.

## Variable Expression Handling

`Variable[T](value)` accepts only:

- simple identifiers (`myVar`)
- package-qualified identifiers (`os.Stdout`)

The detector **normalizes aliased import qualifiers to declared package names**. Example: `import myos "os"` with `myos.Stdout` becomes `os.Stdout` in `ExpressionText`.

During bootstrap generation, expression aliases are rewritten if package name collisions occur.

## Import Management

- Collision-safe aliases are generated when imports overlap
- `RenderImportBlock` is the single source of truth for rendering `import (...)` blocks, shared between `internal/generate/` and `internal/report/`
- Consider both Return-type package and Constructor package when collecting imports for Provide nodes

## Suggested Fix Flow

Code generation is exposed as `analysis.SuggestedFix` — not via a separate codegen binary. This enables:

- Integration with the `braider -fix` workflow
- IDE integration (fixes appear as quick actions)
- Atomic application of related changes
