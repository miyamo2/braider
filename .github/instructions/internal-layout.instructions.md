---
applyTo: "internal/**"
---

# Internal Package Layout

_Read this when: navigating `internal/*`, adding a new component, or wiring a component into `cmd/braider/main.go`._

## Out of Scope

This file's `applyTo:` is `internal/**`, the broadest in the project. It is intentionally a one-line-per-package index only; every detailed topic below is delegated to a topic-specific instruction:

- Analyzer coordination details → see `architecture.instructions.md`
- Detector / annotation mapping → see `annotations.instructions.md`
- Tag handling details → see `struct-tags.instructions.md`
- `internal/generate/` / `internal/report/` internals → see `code-generation.instructions.md`
- `integration_test.go` / `testdata/` layout → see `testing.instructions.md`

## Package Boundaries

braider follows a **standard Go project layout** with clear separation between the public CLI entry point (`cmd/braider/`) and internal implementation (`internal/`).

## `internal/annotation/`
**Sealed marker interfaces** used for `types.Implements` checks.

Types: `Injectable`, `Provider`, `Variable`, `App`, `AppOption`, `AppDefault`, `AppContainer`, and option variants (e.g., `Typed`, `Named`).

Public `pkg/annotation/` types embed these marker interfaces, establishing type-level contracts that detectors match against.

## `internal/detect/`
AST pattern recognition. Components:

- **Call/struct detectors**: `InjectDetector`, `ProvideCallDetector`, `VariableCallDetector`, `AppDetector`, `StructDetector`
- **Analyzers**: `FieldAnalyzer`, `ConstructorAnalyzer`
- **Option extraction**: `OptionExtractor`, `OptionMetadata`, `AppOptionExtractor`
- **Validation**: `NamerValidator`
- **Marker resolution**: `MarkerInterfaces`, `ResolveMarkers` (uses `debug/buildinfo` to resolve module path dynamically, supporting forks)
- **Container models**: `ContainerDefinition`, `ContainerField`

## `internal/registry/`
Shared mutable state (thread-safe, `sync.RWMutex`):

- `ProviderRegistry` / `InjectorRegistry` / `VariableRegistry` — nested `map[TypeName]map[Name]*Info`
- `DuplicateRegistry` — collects duplicate named registrations across packages for deferred reporting by AppAnalyzer

## `internal/graph/`
Dependency graph construction and resolution:

- `DependencyGraphBuilder`
- `TopologicalSorter` (Kahn's algorithm with alphabetical ordering)
- `InterfaceRegistry` — maps interface types to implementing structs
- `ContainerValidator` — validates all container fields are resolvable
- `ContainerResolver` — maps container fields to graph node keys and bootstrap variable names

Variable nodes participate in the graph as zero-dependency leaves (always appear first in topological order).

## `internal/generate/`
AST-based code generation + utilities:

- `ConstructorGenerator` / `BootstrapGenerator`
- AST helpers (`ast_builder.go`): `astIdent`, `astSelector`, `astStructType`, `astShortVar`, `astFuncDecl`, `astVarDecl`, etc.
- Rendering: `renderDecl`, `renderNode`, `RenderImportBlock`
- Hash markers for idempotency, naming conventions, keyword checking, import management

## `internal/report/`
Diagnostic + suggested fix building:

- `SuggestedFixBuilder`, `DiagnosticEmitter`
- Diagnostic category constants (all map to `SeverityCritical`):
  - `CategoryOptionValidation`
  - `CategoryExpressionValidation`
  - `CategoryDependencyRegistration`
- Delegates import rendering to `internal/generate/RenderImportBlock`

## `internal/loader/`
`PackageLoader` for module package discovery (cross-package dependency analysis).

## `internal/analyzer/`
Top-level orchestration:

- `DependencyAnalyzer`, `AppAnalyzer` — `analysis.Analyzer` definitions
- `Aggregator` — `AfterDependencyPhase` callback that iterates `checker.Graph` and populates shared registries between phases
- `DependencyResult` — per-package result type returned by `DependencyAnalyzer.Run()`
- `DependencyAnalyzeRunner`, `AppAnalyzeRunner` — per-analyzer execution drivers

## CLI Entry Point (`cmd/braider/`)

Single `main.go` using braider's own annotations (dogfooding):

1. Declares `annotation.App[app.Container[T]](main)` with a container struct exposing the two analyzers and the `Aggregator`
2. braider generates the `dependency` IIFE that wires all internal components
3. `main()` calls `phasedchecker.Main()` with a `phasedchecker.Config` (Pipeline + DiagnosticPolicy), using the generated container's fields

## Public API (`pkg/`)

`pkg/annotation/` with subpackages `app/`, `inject/`, `provide/`, `variable/`, `namer/`.

Four annotation mechanisms with generic option types (see `annotations.md` for details):

- `Injectable[T inject.Option]` — struct embedding
- `Provide[T provide.Option](fn)` — `var _ =` registration
- `Variable[T variable.Option](value)` — `var _ =` registration
- `App[T app.Option](main)` — main-package call

Option subpackages:

- `app/`: `Default`, `Container[T]`
- `inject/`: `Default`, `Typed[I]`, `Named[N]`, `WithoutConstructor`
- `provide/`: `Default`, `Typed[I]`, `Named[N]`
- `variable/`: `Default`, `Typed[I]`, `Named[N]`
- `namer/`: `Namer` interface (must return string literal from `Name()`)

## Examples (`examples/`)

Each subdirectory is an **independent Go module** (own `go.mod`) demonstrating one annotation pattern. Isolated from the root module via `go.mod`'s `ignore` directive. Named with kebab-case (e.g., `container-basic`, `typed-inject`, `struct-tag-named`).

## Naming Conventions

- **Files**: lowercase, underscore for multi-word (e.g., `analyzer_test.go`)
- **Packages**: short, lowercase, single word when possible
- **Functions**: camelCase, exported PascalCase
- **Variables**: camelCase, short for local scope

## Import Organization

```go
import (
    // Standard library
    "go/ast"

    // External dependencies
    "golang.org/x/tools/go/analysis"

    // Internal packages
    "github.com/miyamo2/braider/internal"
)
```

Order: stdlib → third-party → internal.

## Minimal Public API Principle

Only the CLI entry point (`cmd/braider/main.go`) is user-facing. All implementation details are in `internal/` to prevent accidental external dependencies.
