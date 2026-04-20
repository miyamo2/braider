# Project Structure

## Organization Philosophy

braider follows a **standard Go project layout** with clear separation between public CLI entry point and internal implementation. The structure prioritizes simplicity and follows Go community conventions for analyzer tools.

## Directory Patterns

### CLI Entry Point (`cmd/braider/`)
Single `main.go` that uses braider's own annotations (dogfooding) — `App[app.Container[T]](main)` with a container struct exposing the two analyzers and the Aggregator. braider generates the `dependency` IIFE that wires all internal components; `main()` calls `phasedchecker.Main()` with a `phasedchecker.Config`. The generated container is consumed by the pipeline configuration.

This self-hosting pattern means `cmd/braider/main.go` contains braider-generated bootstrap code, validating the tool against its own codebase.

### Internal Implementation (`internal/`)
Core analyzer logic, not importable by external packages. Organized into focused subpackages by responsibility. See `.claude/rules/internal-layout.md`.

### Runnable Examples (`examples/`)
Self-contained, runnable braider usage examples for documentation and user onboarding. Each subdirectory is an independent Go module (own `go.mod`) with a `main.go` demonstrating one annotation pattern. Isolated from the root module via `go.mod`'s `ignore` directive. Named using kebab-case.

### Test Fixtures (`internal/analyzer/testdata/`)
Go source files used as test inputs for `checkertest`. All e2e cases unified under `testdata/e2e/` (82 cases). See `.claude/rules/testing.md` for category prefix conventions and individual case listings.

## Naming Conventions

- **Files**: lowercase, underscore for multi-word (e.g., `analyzer_test.go`)
- **Packages**: short, lowercase, single word when possible
- **Functions**: camelCase, exported functions PascalCase
- **Variables**: camelCase, short names for local scope

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

Order: stdlib → external (third-party) → internal packages.

## Code Organization Principles

### Component-Based Architecture
The analyzer is built from composable components with clear responsibilities (detectors, generators, reporters, registries, graph, loader, analyzer). Components are wired in `cmd/braider/main.go` via braider's own DI annotations (dogfooding) and passed to analyzer constructors. Full component inventory: `.claude/rules/internal-layout.md`.

### Phased Pipeline Pattern
The project exposes two coordinated analyzers orchestrated via `phasedchecker.Main()`:

- `DependencyAnalyzer` (phase "dependency"): per-package; returns `*DependencyResult`
- `AppAnalyzer` (phase "app"): runs after dependency phase; generates bootstrap code
- `Aggregator`: `AfterDependencyPhase` callback that iterates per-package results and populates shared registries between phases

State flows from DependencyAnalyzer to AppAnalyzer through shared registries populated by the Aggregator.

### Minimal Public API
Only the CLI entry point (`cmd/braider/main.go`) is user-facing. All implementation details live in `internal/` to prevent accidental external dependencies.

### Test Data Isolation
Test fixtures in `testdata/e2e/` follow checkertest conventions. Each test case is a separate Go package analyzable independently.

## Public API (`pkg/`)

`pkg/annotation/` with subpackages `app/`, `inject/`, `provide/`, `variable/`, `namer/`. Four annotation mechanisms (`Injectable`, `Provide`, `Variable`, `App`) with generic option types. Public types embed marker interfaces from `internal/annotation/`, establishing type-level contracts that detectors match against.

See `.claude/rules/annotations.md` for annotation semantics and option types.

---
_Document patterns, not file trees. New files following patterns should not require updates_

_Updated: 2026-02-02 - Added ProvideFunc annotation, expanded generate package utilities, clarified loader package purpose_
_Updated: 2026-02-11 - Sync: Updated annotation API to current generics-based design (Injectable[T], Provide[T](fn)); added inject/provide/namer subpackages; updated component lists (ProvideCallDetector, BootstrapGenerator, OptionExtractor, NamerValidator); corrected testdata categories_
_Updated: 2026-02-12 - Sync: Added Variable[T](value) annotation and variable/ option subpackage; added VariableCallDetector, VariableRegistry to component lists; added variable test case categories_
_Updated: 2026-02-14 - Sync: Updated e2e case count (~52); removed negative from constructorgen (constructor gen now covers zero-dependency structs)_
_Updated: 2026-02-15 - Sync: Provide annotations are now struct fields in bootstrap dependency struct (not local variables); only Variable nodes remain as local variables_
_Updated: 2026-02-15 - Sync: Added internal/annotation marker interface layer; updated CLI entry point to document dogfooding pattern; added constructorgen struct_tag/uppercamel cases_
_Updated: 2026-02-16 - Sync: App annotation now generic App[T](main) with app option type parameter; added app/ option subpackage (Default, Container[T]); added AppOptionExtractor to detector components; added container validation/resolution to graph package; added container test case categories (~77 e2e cases); added AppOption/AppDefault/AppContainer marker interfaces_
_Updated: 2026-02-18 - Sync: Updated e2e case count to ~78; added provide_cross_type test case category; added MarkerInterfaces/ResolveMarkers to detect component description_
_Updated: 2026-02-20 - Sync: Generate package refactored to AST-based code generation; CodeFormatter removed; generators now use ast_builder.go helpers + format.Node instead of string concatenation_
_Updated: 2026-02-25 - Sync: Migrated from multichecker to phasedchecker; CLI entry point now uses app.Container[T] and phasedchecker.Main() with Config/Pipeline/DiagnosticPolicy; added Aggregator/DependencyResult to analyzer package; replaced PackageTracker with DuplicateRegistry; updated test framework references from analysistest to checkertest_
_Updated: 2026-02-26 - Sync: Unified testdata structure under single e2e/ directory (removed legacy bootstrapgen/, constructorgen/, dependency/, providefunc/ top-level directories); updated e2e case count to 81; consolidated redundant test category descriptions into categorized prefix listing; added dep_ smoke test description_
_Updated: 2026-03-02 - Sync: Added examples/ directory pattern (runnable examples as independent Go modules); updated e2e case count from ~81 to ~82_
_Updated: 2026-04-21 - Sync: Moved detailed component inventories and testdata case listings into .claude/rules/ (internal-layout, testing); structure.md retains organization philosophy and pattern descriptions, references rules for specifics_
