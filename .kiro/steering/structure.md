# Project Structure

## Organization Philosophy

braider follows a **standard Go project layout** with clear separation between public CLI entry point and internal implementation. The structure prioritizes simplicity and follows Go community conventions for analyzer tools.

## Directory Patterns

### CLI Entry Point
**Location**: `cmd/braider/`
**Purpose**: CLI wrapper that instantiates components and invokes multiple analyzers
**Pattern**: Single `main.go` that uses braider's own annotations (dogfooding):
1. Declares `annotation.Variable` for shared context values (`bootstrapCtx`, `bootstrapCancel`)
2. Declares `annotation.App[app.Default](main)` to trigger bootstrap generation
3. braider generates the `dependency` IIFE that wires all internal components (detectors, generators, reporters, registries, graph builders, analyzers)
4. `main()` calls `multichecker.Main()` with both analyzers extracted from the generated struct

This self-hosting pattern means braider's own `cmd/braider/main.go` contains braider-generated bootstrap code, validating the tool against its own codebase.

### Internal Implementation
**Location**: `internal/`
**Purpose**: Core analyzer logic, not importable by external packages
**Pattern**: Organized into focused subpackages by responsibility (see Internal Package Organization below)

### Test Fixtures
**Location**: `internal/analyzer/testdata/`
**Purpose**: Go source files used as test inputs for analysistest
**Pattern**: Organized by test category and analyzer:
- `testdata/e2e/` - App annotation scenarios (~78 cases: basic, typed_inject, named_inject, provide_typed, provide_named, provide_cross_type, struct_tag_*, container_*, circular, crosspackage, idempotent, without_constructor, error cases, etc.)
- `testdata/dependency/` - Dependency analysis scenarios (basic, abstrct, cross_package, missing_constructor)
- `testdata/constructorgen/` - Constructor generation scenarios (per-file test cases: simple, multifield, pointer, imported, aliasedimport, definedtypes, typealias, existing, struct_tag_*, uppercamel)
- `testdata/providefunc/` - Provider function detection scenarios (legacy, directories may be empty)



Variable-related test cases follow the same pattern with `variable_` prefix (variable_basic, variable_typed, variable_named, variable_mixed, variable_cross_package, variable_alias_import, variable_pkg_collision, variable_idempotent, variable_outdated) and `error_variable_` prefix for error scenarios. Struct tag test cases use `struct_tag_` prefix (struct_tag_mixed, struct_tag_named, struct_tag_typed_fields, struct_tag_exclude, struct_tag_all_excluded, struct_tag_idempotent, struct_tag_outdated) and `error_struct_tag_` prefix for error scenarios. Container test cases use `container_` prefix (container_basic, container_anonymous, container_named, container_named_field, container_cross_package, container_iface_field, container_mixed_option, container_transitive, container_variable, container_idempotent, container_outdated) and `error_container_` prefix for error scenarios (error_container_non_struct, error_container_ambiguous, error_container_unresolved, error_container_tag_empty, error_container_tag_exclude).

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
- **Registries**: Track state (`ProviderRegistry`, `InjectorRegistry`, `VariableRegistry`, `PackageTracker`)

Components are wired in `cmd/braider/main.go` via braider's own DI annotations (dogfooding) and passed to analyzer constructors.

### Multi-Analyzer Pattern
The project exposes two coordinated analyzers from `internal/analyzer/`:
- **DependencyAnalyzer**: First pass to detect `Injectable[T]` structs, `Provide[T](fn)` calls, and `Variable[T](value)` calls; generate constructors; register to global registries
- **AppAnalyzer**: Second pass to generate bootstrap code using all registered providers, injectors, and variables

Both analyzers share state through global registries, enabling cross-package dependency resolution.

### Minimal Public API
Only the CLI entry point (`cmd/braider/main.go`) is user-facing. All implementation details are in `internal/` to prevent accidental external dependencies.

### Test Data Isolation
Test fixtures live in `testdata/e2e/` following analysistest conventions. Each test case is a separate Go package that can be analyzed independently.

### Internal Package Organization
The internal package is split into focused subpackages:
- `internal/analyzer/` - Analyzer definitions (`DependencyAnalyzer`, `AppAnalyzer`) and orchestration
- `internal/annotation/` - Marker interfaces (e.g., `Injectable`, `Provider`, `Variable`, `App`, `AppOption`, `AppDefault`, `AppContainer`) embedded by the public `pkg/annotation/` types; provides the type-level contracts that detectors match against
- `internal/detect/` - Detection logic for DI patterns (inject, provide call, variable call, app annotations, struct analysis, field analysis, constructor detection, option extraction, namer validation, app option extraction, container definition/field models, marker resolution via `MarkerInterfaces`/`ResolveMarkers`)
- `internal/generate/` - AST-based code generation (constructors, bootstrap IIFE) and utilities (AST builder helpers, import management, naming conventions, keyword checking, hash generation)
- `internal/report/` - Diagnostic and suggested fix building
- `internal/registry/` - Global state management (provider registry, injector registry, variable registry, package tracker)
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
