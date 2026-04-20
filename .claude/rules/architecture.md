---
paths:
  - "internal/analyzer/**"
  - "internal/registry/**"
  - "cmd/braider/main.go"
---

# Architecture

_Read this when: working on the analyzer pipeline, registry population, cross-phase state, or adding/modifying the Aggregator or analyzer coordination logic._

## Out of Scope

Topics under this rule's `paths:` that are handled elsewhere:

- `internal/analyzer/integration_test.go` / `internal/analyzer/testdata/**` (golden + checkertest workflow) → see `testing.md`
- Constructor / bootstrap AST construction itself (the pipeline only invokes it; emit details live elsewhere) → see `code-generation.md`
- Diagnostic wording, `SuggestedFix` structure, severity mapping → see `code-generation.md`
- Annotation / option type specifications themselves (the analyzer consumes them but does not define them) → see `annotations.md`

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
