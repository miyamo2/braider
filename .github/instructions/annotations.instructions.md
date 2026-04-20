---
applyTo: "pkg/annotation/**,internal/detect/**"
---

# DI Annotations

_Read this when: adding/modifying annotation types, options, detectors under `internal/detect/`, or the public `pkg/annotation/` API._

## Out of Scope

Topics under this instruction's `applyTo:` that are handled elsewhere (both `pkg/annotation/**` and `internal/detect/**` are shared with `struct-tags.instructions.md`, so the split is by responsibility, not by path):

- `braider:"..."` struct tag semantics and conflict rules → see `struct-tags.instructions.md`
- How option values influence the emitted constructor / bootstrap code → see `code-generation.instructions.md`
- Cross-phase aggregation of detection results into shared registries → see `architecture.instructions.md`

## Annotation Types

Defined in `pkg/annotation/`:

| Annotation | Form | Purpose |
|---|---|---|
| `annotation.Injectable[T inject.Option]` | struct embedding | Marks DI target; becomes bootstrap struct field |
| `annotation.Provide[T provide.Option](fn)` | package-level `var _ =` | Registers provider function; becomes bootstrap struct field |
| `annotation.Variable[T variable.Option](value)` | package-level `var _ =` | Registers a pre-existing variable/expression; becomes direct assignment in bootstrap IIFE (no constructor invocation) |
| `annotation.App[T appopt.Option](main)` | call in main() | Triggers bootstrap generation with configurable output mode |

## Option Types

Located in `pkg/annotation/{inject,provide,variable,app}/`. Customize registration behavior:

### inject options
- `inject.Default` — standard registration
- `inject.Typed[I]` — register as interface type `I`
- `inject.Named[N]` — register with name from `N.Name()` (`N` must implement `namer.Namer` and return a string literal)
- `inject.WithoutConstructor` — skip constructor generation (inject only)

### provide options
- `provide.Default` / `provide.Typed[I]` / `provide.Named[N]`

### variable options
- `variable.Default` / `variable.Typed[I]` / `variable.Named[N]`

### app options (see below)
- `app.Default` / `app.Container[T]`

## Mixed Options

Combine multiple options via anonymous interface embedding:

```go
annotation.Injectable[interface{ inject.Typed[I]; inject.Named[N] }]
```

Supported across Injectable/Provide/Variable/App.

## Namer Interface

`namer.Namer` is used by `Named[N]` options:

- `N` must implement `namer.Namer`
- `N.Name()` must return a **string literal** (not a computed value)
- Validated by `NamerValidator` in `internal/detect/`

## App Options (`pkg/annotation/app/`)

Controls bootstrap output mode via the App type parameter:

### `app.Default`
Standard bootstrap: generates anonymous struct with all dependencies as fields.

### `app.Container[T]`
User-defined container: `T` is a struct type (named or anonymous) whose fields map to dependencies; bootstrap returns an instance of `T`.

- Container fields use `braider:"name"` struct tags to match named dependencies
- Fields without tags match by type
- `braider:"-"` is **not permitted** on container fields (emits validation error)

**Container pipeline** (detect-validate-resolve-generate):
1. `AppOptionExtractor` classifies the type argument as Default or Container and extracts the `ContainerDefinition` model
2. `ContainerValidator` validates all container fields are resolvable against the dependency graph before generation
3. `ContainerResolver` maps each container field to its dependency graph node key and bootstrap variable name
4. `BootstrapGenerator.GenerateContainerBootstrap` produces the typed IIFE code with a struct literal return

Mixed options (combining Container with other App options) are supported via anonymous interface embedding.

## Variable Annotation Details

`annotation.Variable[T](value)` registers an existing variable or package-qualified identifier (e.g., `os.Stdout`) as a DI dependency without invoking a constructor.

### Supported expressions
- identifiers (`myVar`)
- package-qualified selectors (`os.Stdout`)

### Unsupported expressions (emit diagnostic errors)
- literals
- function calls
- composite literals
- unary/binary operations

### Behavior
- Variable nodes have no dependencies (`InDegree=0`), so they always appear first in topological order
- In bootstrap code: `varName := expressionText` (or `_ = expressionText` if not depended upon)
- Expression packages are normalized to declared names (not user aliases) for consistent import handling
- Aliased imports in expressions are rewritten to collision-safe aliases during bootstrap generation

## Marker Interfaces

Annotation types are identified via `types.Implements` checks against **sealed marker interfaces** defined in `internal/annotation/`.

- `detect.MarkerInterfaces` struct holds resolved `*types.Interface` values
- Cached by `ResolveMarkers()` — uses `debug/buildinfo` to resolve the module path dynamically, supporting forks
- This replaces hard-coded package path string checks

Public `pkg/annotation/` types embed marker interfaces from `internal/annotation/` (e.g., `annotation.Injectable` embeds `internal/annotation.Injectable`), establishing the type-level contracts that detectors match against.
