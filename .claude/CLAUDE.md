# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**braider** is a `go vet` analyzer that resolves DI (Dependency Injection) bindings and generates wiring code automatically using `analysis.SuggestedFix`. Inspired by [google/wire](https://github.com/google/wire), it produces plain Go code with no runtime container.

- **go 1.25** / **`golang.org/x/tools/go/analysis`** / **`github.com/miyamo2/phasedchecker`**

## Build & Test Commands

```bash
go build ./...                                        # Build all packages
go test ./...                                         # Run all tests
go test -v -run TestAppAnalyzer ./internal/analyzer   # Run a specific test
go build -o braider ./cmd/braider                     # Build analyzer binary
braider ./...                                         # Run analyzer
braider -fix ./...                                    # Apply suggested fixes
```

## Architecture

### Two-Phase Pipeline Design

braider runs two analyzers via `phasedchecker.Main` (wired in `cmd/braider/main.go`) with an explicit phased pipeline:

1. **Phase "dependency"** (`braider_dependency` / DependencyAnalyzer) — runs per-package, 4 phases internally:
   - **Phase 1**: Detect `annotation.Injectable[T]` structs → generate constructors via SuggestedFix
   - **Phase 2**: Detect `annotation.Provide[T](fn)` calls → collect to `DependencyResult.Providers`
   - **Phase 2.5**: Detect `annotation.Variable[T](value)` calls → collect to `DependencyResult.Variables`
   - **Phase 3**: Re-detect Injectable structs → collect to `DependencyResult.Injectors` (with `IsPending` flag)
   - Returns `*DependencyResult` per package
   - **AfterPhase callback**: `Aggregator.AfterDependencyPhase` iterates all per-package results and aggregates into shared registries

2. **Phase "app"** (`braider_app` / AppAnalyzer) — runs on main package after all dependency phase packages complete:
   - Detect `annotation.App[T](main)` → extract App option → check for duplicate registrations → build dependency graph → topological sort → generate IIFE bootstrap code
   - When `app.Container[T]` option: validate container fields → resolve fields to graph nodes → generate container-typed bootstrap

### Cross-Phase Coordination

The two phases share state via **global registries** (thread-safe, `sync.RWMutex`), populated by the `Aggregator.AfterDependencyPhase` callback between phases:
- `ProviderRegistry` / `InjectorRegistry` / `VariableRegistry`: nested `map[TypeName]map[Name]*Info`
- `DuplicateRegistry`: collects duplicate named dependency registrations across packages; reported by AppAnalyzer as Critical diagnostics
- **DiagnosticPolicy**: maps three categories to `SeverityCritical` to abort the pipeline:
  - `CategoryOptionValidation` — annotation option constraint violations
  - `CategoryExpressionValidation` — unsupported expression types
  - `CategoryDependencyRegistration` — duplicate dependency registrations

### DependencyResult and Aggregator

`DependencyAnalyzer.Run()` returns `*DependencyResult` (per-package providers, injectors, variables) instead of writing directly to global registries. After all packages in the dependency phase complete, `Aggregator.AfterDependencyPhase` iterates the phasedchecker graph, extracts each `DependencyResult`, and populates the shared registries. Duplicate registrations are collected into `DuplicateRegistry` for deferred reporting.

### IsPending Flag

In `InjectorInfo`, `IsPending=true` means the constructor was generated in the current analysis pass (not yet on disk). `IsPending=false` means an existing constructor was found. This enables single-pass constructor + bootstrap generation.

### Hash-Based Idempotency

Bootstrap code includes a `// braider:hash:<hash>` comment. On subsequent runs, if the computed hash matches the existing one, regeneration is skipped. Hash inputs: `TypeName`, `ConstructorName`, `IsField`, `Dependencies`, `ExpressionText` (NOT `RegisteredType`).

### Dependency Graph

- Graph nodes use composite keys for named dependencies: `"TypeName#Name"`
- `InterfaceRegistry` maps interface types to implementing structs for resolution
- `TopologicalSorter` uses Kahn's algorithm with alphabetical ordering for deterministic output
- Cycle detection with path reconstruction for error messages

### DI Annotations and Options

Annotation types in `pkg/annotation/`:
- **`annotation.Injectable[T inject.Option]`** — struct embedding; marks DI targets (becomes bootstrap struct field)
- **`annotation.Provide[T provide.Option](fn)`** — package-level `var _ =`; registers provider function (becomes bootstrap struct field)
- **`annotation.Variable[T variable.Option](value)`** — package-level `var _ =`; registers a pre-existing variable/expression as a dependency (becomes direct assignment in bootstrap IIFE, no constructor invocation)
- **`annotation.App[T appopt.Option](main)`** — triggers bootstrap generation with configurable output mode
- **`braider:"name"` struct tag** — on Injectable struct fields, controls dependency matching by name; `braider:"-"` excludes the field from DI

Option types customize registration:
- `inject.Default` / `provide.Default` / `variable.Default` — standard registration
- `inject.Typed[I]` / `provide.Typed[I]` / `variable.Typed[I]` — register as interface type `I`
- `inject.Named[N]` / `provide.Named[N]` / `variable.Named[N]` — register with name from `N.Name()` (`N` must implement `namer.Namer` and return a string literal)
- `inject.WithoutConstructor` — skip constructor generation (inject only)

Mixed options via anonymous interface embedding: `Injectable[interface{ inject.Typed[I]; inject.Named[N] }]`

### App Options (`pkg/annotation/app/`)

- **`app.Default`** — standard bootstrap: generates anonymous struct with all dependencies as fields
- **`app.Container[T]`** — user-defined container: `T` is a struct type (named or anonymous) whose fields map to dependencies; bootstrap returns an instance of `T`

Container fields use `braider:"name"` struct tags to match named dependencies. Fields without tags match by type. `braider:"-"` excludes a field from resolution.

### Variable Annotation Details

`annotation.Variable[T](value)` registers an existing variable or package-qualified identifier (e.g., `os.Stdout`) as a DI dependency without invoking a constructor. Key behaviors:
- Supported argument expressions: identifiers (`myVar`) and package-qualified selectors (`os.Stdout`)
- Unsupported expressions (literals, function calls, composite literals, unary/binary ops) emit diagnostic errors
- Variable nodes have no dependencies (`InDegree=0`), so they always appear first in topological order
- In bootstrap code: `varName := expressionText` (or `_ = expressionText` if not depended upon)
- Expression packages are normalized to declared names (not user aliases) for consistent import handling
- Aliased imports in expressions are rewritten to collision-safe aliases during bootstrap generation

### Struct Tag Details

`braider:"name"` struct tags on Injectable struct fields control dependency resolution:
- `braider:"primaryDB"` — matches a named dependency with name `primaryDB`
- `braider:"-"` — excludes the field from DI (skipped during constructor generation and dependency resolution)
- Fields without a `braider` tag are matched by type (default behavior)
- Conflicting tags (e.g., using `braider` tag on a field of an Injectable with `inject.Named` option) emit diagnostic errors

### Sealed Marker Interfaces

Annotation types are identified via `types.Implements` checks against sealed marker interfaces defined in `internal/annotation/`. The `detect.MarkerInterfaces` struct holds resolved `*types.Interface` values, cached by `ResolveMarkers()` (uses `debug/buildinfo` to resolve the module path dynamically, supporting forks). This replaces hard-coded package path string checks.

### Internal Package Layers

- **`annotation/`** — sealed marker interfaces (`Injectable`, `Provider`, `Variable`, `App`, and their option variants) used for `types.Implements` checks
- **`detect/`** — AST pattern recognition (InjectDetector, ProvideCallDetector, VariableCallDetector, AppDetector, AppOptionExtractor, StructDetector, FieldAnalyzer, ConstructorAnalyzer, OptionExtractor, OptionMetadata, NamerValidator, MarkerResolver, ContainerDefinition/ContainerField models)
- **`registry/`** — shared mutable state (ProviderRegistry, InjectorRegistry, VariableRegistry, DuplicateRegistry)
- **`graph/`** — DependencyGraphBuilder, TopologicalSorter, InterfaceRegistry, ContainerValidator, ContainerResolver
- **`generate/`** — ConstructorGenerator, BootstrapGenerator, hash computation, import management, AST-based code generation (ast_builder helpers + format.Node)
- **`report/`** — SuggestedFixBuilder, DiagnosticEmitter, diagnostic category constants (CategoryOptionValidation, CategoryExpressionValidation, CategoryDependencyRegistration map to SeverityCritical)
- **`loader/`** — PackageLoader for module package discovery
- **`analyzer/`** — Aggregator (AfterPhase callback), DependencyResult (per-package result type)

## Testing Patterns

### phasedchecker/checkertest Framework

Tests use `github.com/miyamo2/phasedchecker/checkertest` with testdata directories under `internal/analyzer/testdata/`.

- **DependencyAnalyzer-only tests**: `checkertest.Run` (no golden files) — validates diagnostics only
- **Integration tests (DependencyAnalyzer + AppAnalyzer)**: `checkertest.RunWithSuggestedFixes` — validates generated code against `.golden` files
- Tests build a `phasedchecker.Config` with the same Pipeline/DiagnosticPolicy as production, using isolated registry instances

### Testdata Conventions

- Source files contain `// want "message"` directives to assert expected diagnostics
- `.golden` files must match the source file content after SuggestedFix application
- For idempotent tests (no `// want` on App annotation): existing bootstrap hash must match computed hash
- **Golden file workflow**: create placeholder `.golden` → run test → get diff → paste actual output as golden
- Testdata modules use `replace` directives: from `testdata/<case>/` to `pkg` = `../../../../../pkg` (count `..` levels carefully)
- Avoid `string`/primitive fields in testdata structs — they become unresolvable DI dependencies

### Key Test Directories

- `testdata/bootstrapgen/` — 78 test case directories organized by category:
  - Core: basic, simpleapp, multitype, crosspackage, modulewide, samefileapp, emptygraph, depinuse, depblank, pkgcollision, without_constructor
  - Interface: iface, ifacedep, crossiface, unresiface
  - Typed/Named inject: typed_inject, named_inject
  - Provide: provide_typed, provide_named, provide_cross_type
  - Variable: variable_basic, variable_named, variable_typed, variable_typed_named, variable_cross_package, variable_pkg_collision, variable_alias_import, variable_ident_ext_type, variable_mixed
  - Container: container_anonymous, container_basic, container_cross_package, container_idempotent, container_iface_field, container_mixed_option, container_named, container_named_field, container_outdated, container_provide_cross_type, container_transitive, container_variable
  - Struct tag: struct_tag_all_excluded, struct_tag_exclude, struct_tag_idempotent, struct_tag_mixed, struct_tag_named, struct_tag_outdated, struct_tag_typed_fields
  - Idempotent: idempotent, idempotent_import, outdated, variable_idempotent, variable_outdated
  - Error: error_cases, error_duplicate_name, error_nonliteral, error_provide_typed, error_variable_*, error_struct_tag_*, error_container_*, circular, ambiguous*, missingctor, unresolvedparam, unresparam, unresolvedif, nonmainapp, noapp, multipleapp
- `testdata/constructorgen/` — per-file test cases: simple, multifield, pointer, existing, imported, aliasedimport, definedtypes, typealias, struct_tag_named, struct_tag_exclude, uppercamel
- `testdata/dependency/` — DependencyAnalyzer-only tests: basic, abstrct, cross_package, missing_constructor

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/v1.0.0/): `<type>(<scope>): <description>`

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

### AI Assistant Documentation

When running `/init` to update this CLAUDE.md, also update these files to stay in sync:
- `.github/copilot-instructions.md` (trailer: `Co-authored-by: Copilot <175728472+Copilot@users.noreply.github.com>`)
- `.gemini/GEMINI.md` (trailer: `Co-Authored-By: gemini-cli <218195315+gemini-cli@users.noreply.github.com>`)

All three files should contain the same architectural information with only AI-specific commit trailer differences.
