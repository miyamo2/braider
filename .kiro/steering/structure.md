# Project Structure

## Organization Philosophy

braider follows a **standard Go project layout** with clear separation between public CLI entry point and internal implementation. The structure prioritizes simplicity and follows Go community conventions for analyzer tools.

## Directory Patterns

### CLI Entry Point
**Location**: `cmd/braider/`
**Purpose**: CLI wrapper that instantiates components and invokes multiple analyzers
**Pattern**: Single `main.go` that uses braider's own annotations (dogfooding):
1. Declares `annotation.App[app.Container[T]](main)` with a container struct exposing the two analyzers and the Aggregator
2. braider generates the `dependency` IIFE that wires all internal components (detectors, generators, reporters, registries, graph builders, analyzers)
3. `main()` calls `phasedchecker.Main()` with a `phasedchecker.Config` that configures the Pipeline (phase ordering, AfterPhase callbacks) and DiagnosticPolicy (category-to-severity rules), using the generated container's fields

This self-hosting pattern means braider's own `cmd/braider/main.go` contains braider-generated bootstrap code, validating the tool against its own codebase.

### Internal Implementation
**Location**: `internal/`
**Purpose**: Core analyzer logic, not importable by external packages
**Pattern**: Organized into focused subpackages by responsibility (see Internal Package Organization below)

### Test Fixtures
**Location**: `internal/analyzer/testdata/`
**Purpose**: Go source files used as test inputs for checkertest
**Pattern**: All test cases unified under a single `testdata/e2e/` directory (~81 cases), organized by category prefix:
- Core scenarios: `basic`, `crosspackage`, `simpleapp`, `modulewide`, `pkgcollision`, `emptygraph`, `depinuse`, `samefileapp`, `without_constructor`
- Interface resolution: `iface`, `ifacedep`, `crossiface`, `unresiface`
- Typed/Named inject: `typed_inject`, `named_inject`
- Provide variations: `provide_typed`, `provide_named`, `provide_cross_type`
- Variable annotation: `variable_*` prefix (basic, typed, named, mixed, cross_package, alias_import, pkg_collision, idempotent, outdated, ident_ext_type, typed_named)
- Struct tag: `struct_tag_*` prefix (named, exclude, mixed, all_excluded, typed_fields, idempotent, outdated)
- Container mode: `container_*` prefix (basic, anonymous, named, named_field, cross_package, iface_field, mixed_option, transitive, variable, idempotent, outdated, provide_cross_type)
- Idempotent/outdated: `idempotent`, `outdated`
- Error cases: `error_*` prefix (error_cases, error_duplicate_name, error_nonliteral, error_provide_typed, error_variable_*, error_struct_tag_*, error_container_*)
- Constructor generation: `constructorgen/` (per-file test cases with .go/.golden pairs)
- Dependency-only smoke tests: `dep_basic`, `dep_missing_constructor`, `dep_cross_package`, `dep_interface_impl` (no App annotation, no golden files; verify dependency phase runs without unexpected diagnostics)

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

**Import Order**:
1. Standard library
2. External dependencies (third-party)
3. Internal packages

## Code Organization Principles

### Component-Based Architecture
The analyzer is built from composable components with clear responsibilities:
- **Detectors**: Find DI patterns (`InjectDetector`, `ProvideCallDetector`, `VariableCallDetector`, `AppDetector`, `AppOptionExtractor`, `StructDetector`, `FieldAnalyzer`, `ConstructorAnalyzer`, `OptionExtractor`, `NamerValidator`)
- **Generators**: Produce code via AST construction + `format.Node` (`ConstructorGenerator`, `BootstrapGenerator`)
- **Reporters**: Emit diagnostics (`SuggestedFixBuilder`, `DiagnosticEmitter`)
- **Registries**: Track state (`ProviderRegistry`, `InjectorRegistry`, `VariableRegistry`, `DuplicateRegistry`)

Components are wired in `cmd/braider/main.go` via braider's own DI annotations (dogfooding) and passed to analyzer constructors.

### Phased Pipeline Pattern
The project exposes two coordinated analyzers from `internal/analyzer/`, orchestrated via `phasedchecker.Main()`:
- **DependencyAnalyzer** (Phase "dependency"): Per-package; detects `Injectable[T]` structs, `Provide[T](fn)` calls, and `Variable[T](value)` calls; generates constructors; returns `*DependencyResult`
- **AppAnalyzer** (Phase "app"): Runs after dependency phase; generates bootstrap code using all registered providers, injectors, and variables
- **Aggregator**: `AfterDependencyPhase` callback that iterates per-package results via `checker.Graph` and populates shared registries between phases

State flows from DependencyAnalyzer to AppAnalyzer through shared registries populated by the Aggregator.

### Minimal Public API
Only the CLI entry point (`cmd/braider/main.go`) is user-facing. All implementation details are in `internal/` to prevent accidental external dependencies.

### Test Data Isolation
Test fixtures live in `testdata/e2e/` following checkertest conventions. Each test case is a separate Go package that can be analyzed independently.

### Internal Package Organization
The internal package is split into focused subpackages:
- `internal/analyzer/` - Analyzer definitions (`DependencyAnalyzer`, `AppAnalyzer`), `Aggregator` (AfterPhase callback), `DependencyResult` (per-package result type), and orchestration runners
- `internal/annotation/` - Marker interfaces (e.g., `Injectable`, `Provider`, `Variable`, `App`, `AppOption`, `AppDefault`, `AppContainer`) embedded by the public `pkg/annotation/` types; provides the type-level contracts that detectors match against
- `internal/detect/` - Detection logic for DI patterns (inject, provide call, variable call, app annotations, struct analysis, field analysis, constructor detection, option extraction, namer validation, app option extraction, container definition/field models, marker resolution via `MarkerInterfaces`/`ResolveMarkers`)
- `internal/generate/` - AST-based code generation (constructors, bootstrap IIFE) and utilities (AST builder helpers, import management, naming conventions, keyword checking, hash generation)
- `internal/report/` - Diagnostic and suggested fix building
- `internal/registry/` - Shared state management (provider registry, injector registry, variable registry, duplicate registry)
- `internal/graph/` - Dependency resolution (dependency graph, interface registry, topological sort, container validation, container field resolution)
- `internal/loader/` - Package loading utilities for cross-package dependency analysis

### Public API (`pkg/`)
**Location**: `pkg/annotation/` with subpackages `app/`, `inject/`, `provide/`, `variable/`, `namer/`
**Purpose**: Public annotation types and functions for users to mark DI targets
**Dependency**: Public types embed marker interfaces from `internal/annotation/` (e.g., `annotation.Injectable` embeds `internal/annotation.Injectable`), establishing the type-level contracts that detectors match against
**Pattern**: Four annotation mechanisms with generic option types:
- `Injectable[T inject.Option]` interface - Embed in structs to mark for constructor generation and DI registration
- `Provide[T provide.Option](fn)` function - Register provider functions via `var _ = annotation.Provide[T](fn)` (struct fields in bootstrap dependency struct)
- `Variable[T variable.Option](value)` function - Register existing variables/values via `var _ = annotation.Variable[T](value)` (expression assignments in bootstrap IIFE)
- `App[T app.Option](main)` function - Call in main package to mark entry point for bootstrap code generation; type parameter `T` configures output mode (default anonymous struct or user-defined container)

**Option subpackages**:
- `app/` - Options for App: `Default`, `Container[T]`
- `inject/` - Options for Injectable: `Default`, `Typed[I]`, `Named[N]`, `WithoutConstructor`
- `provide/` - Options for Provide: `Default`, `Typed[I]`, `Named[N]`
- `variable/` - Options for Variable: `Default`, `Typed[I]`, `Named[N]`
- `namer/` - `Namer` interface for Named option types (must return string literal from `Name()`)

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
