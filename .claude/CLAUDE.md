# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**braider** is a `go vet` analyzer that resolves DI (Dependency Injection) bindings and generates wiring code automatically using `analysis.SuggestedFix`. Inspired by [google/wire](https://github.com/google/wire), it produces plain Go code with no runtime container.

- **Go 1.24** / **`golang.org/x/tools/go/analysis`**

## Build & Test Commands

```bash
go build ./...                                        # Build all packages
go test ./...                                         # Run all tests
go test -v -run TestAppAnalyzer ./internal/analyzer   # Run a specific test
go build -o braider ./cmd/braider                     # Build analyzer binary
go vet -vettool=$(which braider) ./...                # Run analyzer
go vet -vettool=$(which braider) -fix ./...           # Apply suggested fixes
```

## Architecture

### Two-Analyzer Design

braider runs two analyzers via `multichecker.Main` (wired in `cmd/braider/main.go`):

1. **`braider_dependency`** (DependencyAnalyzer) — runs per-package, 4 phases:
   - **Phase 1**: Detect `annotation.Injectable[T]` structs → generate constructors via SuggestedFix
   - **Phase 2**: Detect `annotation.Provide[T](fn)` calls → register to `ProviderRegistry`
   - **Phase 3**: Re-detect Injectable structs → register to `InjectorRegistry` (with `IsPending` flag)
   - **Phase 4**: Mark package as scanned in `PackageTracker`

2. **`braider_app`** (AppAnalyzer) — runs on main package:
   - Detect `annotation.App(main)` → wait for all packages to be scanned (10s timeout) → build dependency graph → topological sort → generate IIFE bootstrap code

### Cross-Analyzer Coordination

The two analyzers share state via **global registries** (thread-safe, `sync.RWMutex`):
- `ProviderRegistry` / `InjectorRegistry`: nested `map[TypeName]map[Name]*Info`
- `PackageTracker`: tracks scanned packages; AppAnalyzer calls `WaitForAllPackagesWithContext` before building the graph
- `context.CancelCauseFunc`: DependencyAnalyzer can cancel bootstrap generation on validation errors

### IsPending Flag

In `InjectorInfo`, `IsPending=true` means the constructor was generated in the current analysis pass (not yet on disk). `IsPending=false` means an existing constructor was found. This enables single-pass constructor + bootstrap generation.

### Hash-Based Idempotency

Bootstrap code includes a `// braider:hash:<hash>` comment. On subsequent runs, if the computed hash matches the existing one, regeneration is skipped. Hash inputs: `TypeName`, `ConstructorName`, `IsField`, `Dependencies` (NOT `RegisteredType`).

### Dependency Graph

- Graph nodes use composite keys for named dependencies: `"TypeName#Name"`
- `InterfaceRegistry` maps interface types to implementing structs for resolution
- `TopologicalSorter` uses Kahn's algorithm with alphabetical ordering for deterministic output
- Cycle detection with path reconstruction for error messages

### DI Annotations and Options

Three annotation types in `pkg/annotation/`:
- **`annotation.Injectable[T inject.Option]`** — struct embedding; marks DI targets (becomes bootstrap struct field)
- **`annotation.Provide[T provide.Option](fn)`** — package-level `var _ =`; registers provider function (becomes local var in bootstrap IIFE)
- **`annotation.App(main)`** — triggers bootstrap generation

Option types customize registration:
- `inject.Default` / `provide.Default` — standard registration
- `inject.Typed[I]` / `provide.Typed[I]` — register as interface type `I`
- `inject.Named[N]` / `provide.Named[N]` — register with name from `N.Name()` (`N` must implement `namer.Namer` and return a string literal)
- `inject.WithoutConstructor` — skip constructor generation (inject only)

Mixed options via anonymous interface embedding: `Injectable[interface{ inject.Typed[I]; inject.Named[N] }]`

### Internal Package Layers

- **`detect/`** — AST pattern recognition (InjectDetector, ProvideCallDetector, AppDetector, StructDetector, FieldAnalyzer, ConstructorAnalyzer, OptionExtractor, NamerValidator)
- **`registry/`** — shared mutable state (ProviderRegistry, InjectorRegistry, PackageTracker)
- **`graph/`** — DependencyGraphBuilder, TopologicalSorter, InterfaceRegistry
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

- `testdata/bootstrapgen/` — ~50 test cases covering basic, typed_inject, named_inject, provide_typed, provide_named, circular, error cases, idempotent cases
- `testdata/constructorgen/`, `testdata/providefunc/`, `testdata/dependency/`

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/v1.0.0/): `<type>(<scope>): <description>`

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

### AI Assistant Documentation

When running `/init` to update this CLAUDE.md, also update these files to stay in sync:
- `.github/copilot-instructions.md` (trailer: `Co-authored-by: Copilot <175728472+Copilot@users.noreply.github.com>`)
- `.gemini/GEMINI.md` (trailer: `Co-Authored-By: gemini-cli <218195315+gemini-cli@users.noreply.github.com>`)

All three files should contain the same architectural information with only AI-specific commit trailer differences.
