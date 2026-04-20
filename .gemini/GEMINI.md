# Gemini CLI Instructions

**braider** is a `go/analysis`-based analyzer that resolves DI bindings and generates wiring code automatically using `analysis.SuggestedFix`. Inspired by [google/wire](https://github.com/google/wire); produces plain Go with no runtime container.

Stack: **go 1.25** / `golang.org/x/tools/go/analysis` / `github.com/miyamo2/phasedchecker`.

## Build & Test

```bash
go build ./...                                           # Build all packages
go test ./...                                            # Run all tests
go test -v -run TestIntegration ./internal/analyzer      # All e2e tests
go test -v -run TestIntegration/basic ./internal/analyzer # Single e2e case
go build -o braider ./cmd/braider                        # Build binary
braider ./...                                            # Run analyzer
braider -fix ./...                                       # Apply suggested fixes
```

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/v1.0.0/): `<type>(<scope>): <description>`. Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`.

Include the trailer on AI-assisted commits:

```
Co-Authored-By: gemini-cli <218195315+gemini-cli@users.noreply.github.com>
```

## Cross-AI Sync Policy

`.claude/rules/*.md` is the **source of truth** for detailed rules. When updating a topic, mirror the change to `.github/instructions/{topic}.instructions.md` (Copilot) and the corresponding section below (Gemini CLI).

## Task → Paths Guide

Reverse lookup from intent to the paths you need to touch.

| Intent | Primary paths to edit |
|---|---|
| Add a new annotation type (a 5th kind after Injectable / Provide / Variable / App) | `pkg/annotation/<new>/`, `internal/annotation/` (marker interface), `internal/detect/` (detector), `internal/registry/` (new registry if needed) |
| Add an option type to an existing annotation (e.g. `inject.Foo` / `provide.Bar`) | `pkg/annotation/{inject,provide,variable,app}/`, `internal/detect/` (around `OptionExtractor`) |
| Add or change `braider:"..."` struct tag behavior | `internal/detect/` (tag parsing), plus `internal/generate/` if the tag affects emitted code |
| Change pipeline behavior, phase split, or cross-phase registry sharing | `internal/analyzer/`, `internal/registry/`, `cmd/braider/main.go` |
| Change constructor / bootstrap code generation (AST builders, import handling, hash idempotency) | `internal/generate/` |
| Change diagnostic wording, `SuggestedFix` structure, or severity | `internal/report/` |
| Change Namer / marker resolver / AST validation logic | `internal/detect/` (e.g. `NamerValidator`, `ResolveMarkers`) |
| Add / modify E2E test cases or update goldens | `internal/analyzer/integration_test.go`, `internal/analyzer/testdata/**` |
| Add a new `internal/*` package or change package responsibilities | The affected paths themselves, plus the topic sections below that reference them |
| Change CLI entry point / command-line behavior | `cmd/braider/main.go` |

New categories: update this table before starting implementation.

---


# Architecture

_Read this when: working on the analyzer pipeline, registry population, cross-phase state, or adding/modifying the Aggregator or analyzer coordination logic._

## Out of Scope

Topics under this section's scope (`internal/analyzer/**`, `internal/registry/**`, `cmd/braider/main.go`) that are handled elsewhere:

- `internal/analyzer/integration_test.go` / `internal/analyzer/testdata/**` (golden + checkertest workflow) → see the `Testing` section
- Constructor / bootstrap AST construction itself (the pipeline only invokes it; emit details live elsewhere) → see the `Code Generation` section
- Diagnostic wording, `SuggestedFix` structure, severity mapping → see the `Code Generation` section
- Annotation / option type specifications themselves (the analyzer consumes them but does not define them) → see the `DI Annotations` section

## Two-Phase Pipeline Design

braider runs two analyzers via `phasedchecker.Main` (wired in `cmd/braider/main.go`) with an explicit phased pipeline:

### Phase "dependency" (`braider_dependency` / DependencyAnalyzer)
Runs per-package, 4 phases internally:

- **Phase 1**: Detect `annotation.Injectable[T]` structs → generate constructors via `analysis.SuggestedFix`
- **Phase 2**: Detect `annotation.Provide[T](fn)` calls → collect to `DependencyResult.Providers`
- **Phase 2.5**: Detect `annotation.Variable[T](value)` calls → collect to `DependencyResult.Variables`
- **Phase 3**: Re-detect Injectable structs → collect to `DependencyResult.Injectors` (with `IsPending` flag)

Returns `*DependencyResult` per package.

**AfterPhase callback**: `Aggregator.AfterDependencyPhase` iterates all per-package results and aggregates into shared registries.

### Phase "app" (`braider_app` / AppAnalyzer)
Runs on main package after all dependency phase packages complete:

- Detect `annotation.App[T](main)` → extract App option → check for duplicate registrations → build dependency graph → topological sort → generate IIFE bootstrap code
- When `app.Container[T]` option: validate container fields → resolve fields to graph nodes → generate container-typed bootstrap

## Cross-Phase Coordination

The two phases share state via **global registries** (thread-safe, `sync.RWMutex`), populated by `Aggregator.AfterDependencyPhase` between phases:

- `ProviderRegistry` / `InjectorRegistry` / `VariableRegistry`: nested `map[TypeName]map[Name]*Info`
- `DuplicateRegistry`: collects duplicate named dependency registrations across packages; reported by AppAnalyzer as Critical diagnostics

### DiagnosticPolicy

Maps three categories to `SeverityCritical` to abort the pipeline:

- `CategoryOptionValidation` — annotation option constraint violations
- `CategoryExpressionValidation` — unsupported expression types
- `CategoryDependencyRegistration` — duplicate dependency registrations

`DefaultSeverity: SeverityWarn` applies to diagnostics not matching any explicit category rule.

## DependencyResult and Aggregator

`DependencyAnalyzer.Run()` returns `*DependencyResult` (per-package providers, injectors, variables) instead of writing directly to global registries. After all packages in the dependency phase complete, `Aggregator.AfterDependencyPhase` iterates the phasedchecker `checker.Graph`, extracts each `DependencyResult`, and populates the shared registries. Duplicate registrations are collected into `DuplicateRegistry` for deferred reporting.

## IsPending Flag

In `InjectorInfo`:

- `IsPending=true`: constructor was generated in the current analysis pass (not yet on disk)
- `IsPending=false`: an existing constructor was found

This enables single-pass constructor + bootstrap generation.

## Hash-Based Idempotency

Bootstrap code includes a `// braider:hash:<hash>` comment. On subsequent runs, if the computed hash matches the existing one, regeneration is skipped.

**Hash inputs**: `TypeName`, `ConstructorName`, `IsField`, `Dependencies`, `ExpressionText`, `ConstructorPkgPath` (conditional: only when it differs from `PackagePath`). **NOT** `RegisteredType`.

## Dependency Graph

- Graph nodes use composite keys for named dependencies: `"TypeName#Name"`
- `InterfaceRegistry` maps interface types to implementing structs for resolution
- `TopologicalSorter` uses Kahn's algorithm with alphabetical ordering for deterministic output
- Cycle detection with path reconstruction for error messages

## Bootstrap Struct Field vs Local Variable

In the dependency graph, `Node.IsField` determines how a dependency appears in the bootstrap IIFE:

- **IsField=true** (Injectable and Provide nodes): Become fields in the returned dependency struct, accessible to the caller
- **IsField=false** (Variable nodes): Become local variables within the IIFE, not exposed to the caller

## Pipeline Configuration

The pipeline is configured via `phasedchecker.Config` with explicit:

- `Pipeline`: phase ordering, per-phase analyzers, AfterPhase callbacks
- `DiagnosticPolicy`: category-to-severity mappings plus `DefaultSeverity`

## Dogfooding (Self-Hosting)

braider uses its own annotations in `cmd/braider/main.go` to wire its internal components. The entry point declares `annotation.App[app.Container[T]](main)` with a container struct that exposes the two analyzers and the Aggregator. braider then generates the `dependency` IIFE that constructs and wires all detectors, generators, reporters, registries, graph builders, and analyzers. The generated container is consumed by `phasedchecker.Main()` to configure the phased pipeline. This ensures braider validates its own code generation against a real, non-trivial dependency graph.


# DI Annotations

_Read this when: adding/modifying annotation types, options, detectors under `internal/detect/`, or the public `pkg/annotation/` API._

## Out of Scope

Topics under this section's scope (`pkg/annotation/**`, `internal/detect/**`, shared with the struct-tags section, so the split is by responsibility, not by path):

- `braider:"..."` struct tag semantics and conflict rules → see the `` `braider` Struct Tags `` section
- How option values influence the emitted constructor / bootstrap code → see the `Code Generation` section
- Cross-phase aggregation of detection results into shared registries → see the `Architecture` section

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


# `braider` Struct Tags

_Read this when: working on field-level DI customization, constructor generation for Injectable structs, container field resolution, or diagnostics around tag conflicts._

## Out of Scope

Topics under this section's scope (`pkg/annotation/**`, `internal/detect/**`, shared with the annotations section; this section owns tag semantics only):

- Annotation types themselves (`Injectable` / `Provide` / `Variable` / `App`) → see the `DI Annotations` section
- Option types (`inject.*` / `provide.*` / `variable.*` / `app.*`) → see the `DI Annotations` section
- AST-level details of constructor generation derived from tag decisions → see the `Code Generation` section

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


# Code Generation

_Read this when: working on `internal/generate/`, constructor/bootstrap emission, AST builder helpers, import management, or hash-based idempotency._

## Out of Scope

Topics under this section's scope (`internal/generate/**`, `internal/report/**`) that are handled elsewhere:

- When and in which phase generation is invoked (pipeline coordination) → see the `Architecture` section
- Golden / checkertest validation workflow for emitted code → see the `Testing` section
- Detection logic that feeds inputs into generation (annotation detection, tag parsing) → see the `DI Annotations` / `` `braider` Struct Tags `` sections

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


# Testing

_Read this when: adding/modifying test cases, debugging golden-file diffs, working with `checkertest`, or setting up new testdata directories._

## Out of Scope

Topics under this section's scope (`internal/analyzer/integration_test.go`, `internal/analyzer/testdata/**`) that are handled elsewhere:

- Internal structure of the analyzer pipeline under test → see the `Architecture` section
- Rules governing how the generated code that appears in golden files is built → see the `Code Generation` section
- Annotation / struct-tag specifications exercised by the test cases → see the `DI Annotations` / `` `braider` Struct Tags `` sections

## phasedchecker/checkertest Framework

All e2e tests run through a single table-driven `TestIntegration` function in `internal/analyzer/integration_test.go`, using `checkertest.RunWithSuggestedFixes` to validate both diagnostics and generated code against `.golden` files.

Each test case builds a `phasedchecker.Config` with the **same Pipeline/DiagnosticPolicy as production**, using isolated registry instances.

Dependency-only smoke tests (no App annotation, no golden files) share the same test runner; `RunWithSuggestedFixes` verifies no unexpected diagnostics.

## Build & Test Commands

```bash
go build ./...                                           # Build all packages
go test ./...                                            # Run all tests
go test -v -run TestIntegration ./internal/analyzer      # Run all e2e tests
go test -v -run TestIntegration/basic ./internal/analyzer # Run a single e2e case
go build -o braider ./cmd/braider                        # Build analyzer binary
braider ./...                                            # Run analyzer
braider -fix ./...                                       # Apply suggested fixes
```

## Testdata Conventions

- Source files contain `// want "message"` directives to assert expected diagnostics
- `.golden` files must match the source file content **after** SuggestedFix application
- For idempotent tests (no `// want` on App annotation): existing bootstrap hash must match computed hash
- Testdata modules use `replace` directives in their `go.mod`:
  ```
  replace github.com/miyamo2/braider => ./../../../../..
  ```
  (count `..` levels carefully; points to the module root from `testdata/<case>/`)
- **Avoid `string`/primitive fields in testdata structs** — they become unresolvable DI dependencies

## Golden File Workflow

1. Create placeholder `.golden` file (e.g., copy source file as-is)
2. Run test → get diff output
3. Paste actual output as the golden

Golden file hashes must match the computed graph hash. For hash mismatches: run test first to get the actual hash from the diff, then update the golden.

## Testdata Directory Structure

All test cases unified under `internal/analyzer/testdata/e2e/` (82 cases), organized by category prefix:

### Core scenarios
`basic`, `crosspackage`, `simpleapp`, `modulewide`, `pkgcollision`, `emptygraph`, `depinuse`, `samefileapp`, `without_constructor`, `multitype`

### Interface resolution
`iface`, `ifacedep`, `crossiface`, `unresiface`

### Typed/Named inject
`typed_inject`, `named_inject`

### Provide variations
`provide_typed`, `provide_named`, `provide_cross_type`

### Variable annotation (`variable_*`)
`variable_basic`, `variable_named`, `variable_typed`, `variable_typed_named`, `variable_cross_package`, `variable_pkg_collision`, `variable_alias_import`, `variable_ident_ext_type`, `variable_mixed`, `variable_idempotent`, `variable_outdated`

### Container mode (`container_*`)
`container_basic`, `container_anonymous`, `container_named`, `container_named_field`, `container_cross_package`, `container_iface_field`, `container_mixed_option`, `container_transitive`, `container_variable`, `container_idempotent`, `container_outdated`, `container_provide_cross_type`

### Struct tag (`struct_tag_*`)
`struct_tag_named`, `struct_tag_exclude`, `struct_tag_mixed`, `struct_tag_all_excluded`, `struct_tag_typed_fields`, `struct_tag_idempotent`, `struct_tag_outdated`

### Idempotent/outdated
`idempotent`, `outdated`

### Error cases
- `error_*` prefix: `error_cases`, `error_duplicate_name`, `error_duplicate_provide_variable`, `error_nonliteral`, `error_provide_typed`, `error_variable_*`, `error_struct_tag_*`, `error_struct_tag_conflict`, `error_container_*`
- Other: `circular`, `ambiguous*`, `unresolvedparam`, `unresparam`, `unresolvedif`, `nonmainapp`, `noapp`, `multipleapp`

### Constructor generation
`constructorgen/` — per-file test cases with `.go`/`.golden` pairs inside a single test directory

### Dependency-only smoke tests
`dep_basic`, `dep_missing_constructor`, `dep_cross_package`, `dep_interface_impl` — no App annotation, no golden files; verify dependency phase runs without unexpected diagnostics

## Test Philosophy

- Test both **positive** cases (should report) and **negative** cases (should not report)
- Use `phasedchecker.Config` identical to production (Pipeline + DiagnosticPolicy), with isolated registry instances
- Each test case is a separate Go package that can be analyzed independently


# Internal Package Layout

_Read this when: navigating `internal/*`, adding a new component, or wiring a component into `cmd/braider/main.go`._

## Out of Scope

This section's scope is `internal/**`, the broadest in the project. It is intentionally a one-line-per-package index only; every detailed topic below is delegated to a topic-specific section:

- Analyzer coordination details → see the `Architecture` section
- Detector / annotation mapping → see the `DI Annotations` section
- Tag handling details → see the `` `braider` Struct Tags `` section
- `internal/generate/` / `internal/report/` internals → see the `Code Generation` section
- `integration_test.go` / `testdata/` layout → see the `Testing` section

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

