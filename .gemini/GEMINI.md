# Gemini CLI Instructions

This file provides guidance to Gemini CLI when working with code in this repository.

## Project Overview

**braider** is a `go vet` analyzer that resolves DI (Dependency Injection) bindings and generates wiring code automatically using `analysis.SuggestedFix`. Inspired by [google/wire](https://github.com/google/wire), it produces plain Go code with no runtime container.

- **go 1.25** / **`golang.org/x/tools/go/analysis`**

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

### Two-Analyzer Design

braider runs two analyzers via `multichecker.Main` (wired in `cmd/braider/main.go`):

1. **`braider_dependency`** (DependencyAnalyzer) — runs per-package, 5 phases:
   - **Phase 1**: Detect `annotation.Injectable[T]` structs → generate constructors via SuggestedFix
   - **Phase 2**: Detect `annotation.Provide[T](fn)` calls → register to `ProviderRegistry`
   - **Phase 2.5**: Detect `annotation.Variable[T](value)` calls → register to `VariableRegistry`
   - **Phase 3**: Re-detect Injectable structs → register to `InjectorRegistry` (with `IsPending` flag)
   - **Phase 4**: Mark package as scanned in `PackageTracker`

2. **`braider_app`** (AppAnalyzer) — runs on main package:
   - Detect `annotation.App[T](main)` → extract App option → wait for all packages to be scanned (10s timeout) → build dependency graph → topological sort → generate IIFE bootstrap code
   - When `app.Container[T]` option: validate container fields → resolve fields to graph nodes → generate container-typed bootstrap

### Cross-Analyzer Coordination

The two analyzers share state via **global registries** (thread-safe, `sync.RWMutex`):
- `ProviderRegistry` / `InjectorRegistry` / `VariableRegistry`: nested `map[TypeName]map[Name]*Info`
- `PackageTracker`: tracks scanned packages; AppAnalyzer calls `WaitForAllPackagesWithContext` before building the graph
- `context.CancelCauseFunc`: DependencyAnalyzer can cancel bootstrap generation on validation errors

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
- **`registry/`** — shared mutable state (ProviderRegistry, InjectorRegistry, VariableRegistry, PackageTracker)
- **`graph/`** — DependencyGraphBuilder, TopologicalSorter, InterfaceRegistry, ContainerValidator, ContainerResolver
- **`generate/`** — ConstructorGenerator, BootstrapGenerator, hash computation, import management
- **`report/`** — SuggestedFixBuilder, DiagnosticEmitter
- **`loader/`** — PackageLoader for module package discovery

## Testing Patterns

### analysistest Framework

Tests use `golang.org/x/tools/go/analysis/analysistest` with testdata directories under `internal/analyzer/testdata/`.

- **DependencyAnalyzer tests**: `analysistest.Run` (no golden files) — validates diagnostics only
- **AppAnalyzer tests**: `analysistest.RunWithSuggestedFixes` — validates generated code against `.golden` files

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
  - Container: container_anonymous, container_basic, container_cross_package, container_idempotent, container_iface_field, container_mixed_option, container_named, container_named_field, container_outdated, container_transitive, container_variable
  - Struct tag: struct_tag_all_excluded, struct_tag_exclude, struct_tag_idempotent, struct_tag_mixed, struct_tag_named, struct_tag_outdated, struct_tag_typed_fields
  - Idempotent: idempotent, idempotent_import, outdated, variable_idempotent, variable_outdated
  - Error: error_cases, error_duplicate_name, error_nonliteral, error_provide_typed, error_variable_*, error_struct_tag_*, error_container_*, circular, ambiguous*, missingctor, unresolvedparam, unresparam, unresolvedif, contextcancel, nonmainapp, noapp, multipleapp
- `testdata/constructorgen/` — per-file test cases: simple, multifield, pointer, existing, imported, aliasedimport, definedtypes, typealias, struct_tag_named, struct_tag_exclude, uppercamel
- `testdata/dependency/` — DependencyAnalyzer-only tests: basic, abstrct, cross_package, missing_constructor

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/v1.0.0/): `<type>(<scope>): <description>`

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

When Gemini CLI commits, add the following trailer:
`Co-Authored-By: gemini-cli <218195315+gemini-cli@users.noreply.github.com>`
