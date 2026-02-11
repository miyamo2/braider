# Project Structure

## Organization Philosophy

braider follows a **standard Go project layout** with clear separation between public CLI entry point and internal implementation. The structure prioritizes simplicity and follows Go community conventions for analyzer tools.

## Directory Patterns

### CLI Entry Point
**Location**: `cmd/braider/`
**Purpose**: CLI wrapper that instantiates components and invokes multiple analyzers
**Pattern**: Single `main.go` that:
1. Creates shared registries (provider, injector, package tracker)
2. Instantiates detectors, generators, and reporters
3. Constructs `DependencyAnalyzer` and `AppAnalyzer` with dependencies
4. Calls `multichecker.Main()` with both analyzers

### Internal Implementation
**Location**: `internal/`
**Purpose**: Core analyzer logic, not importable by external packages
**Pattern**: Organized into focused subpackages by responsibility (see Internal Package Organization below)

### Test Fixtures
**Location**: `internal/analyzer/testdata/`
**Purpose**: Go source files used as test inputs for analysistest
**Pattern**: Organized by test category and analyzer:
- `testdata/bootstrapgen/` - App annotation scenarios (~35 cases: basic, typed_inject, named_inject, provide_typed, provide_named, circular, crosspackage, idempotent, without_constructor, error cases, etc.)
- `testdata/dependency/` - Dependency analysis scenarios (basic, abstrct, cross_package, missing_constructor)
- `testdata/constructorgen/` - Constructor generation scenarios (simple, multifield, pointer, imported, aliasedimport, definedtypes, typealias, existing, negative)
- `testdata/providefunc/` - Provider function detection scenarios (legacy, directories may be empty)

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
- **Detectors**: Find DI patterns (`InjectDetector`, `ProvideCallDetector`, `AppDetector`, `StructDetector`, `FieldAnalyzer`, `ConstructorAnalyzer`, `OptionExtractor`, `NamerValidator`)
- **Generators**: Produce code (`ConstructorGenerator`, `BootstrapGenerator`)
- **Reporters**: Emit diagnostics (`SuggestedFixBuilder`, `DiagnosticEmitter`)
- **Registries**: Track state (`ProviderRegistry`, `InjectorRegistry`, `PackageTracker`)

Components are instantiated in `main.go` and passed to analyzer constructors via dependency injection.

### Multi-Analyzer Pattern
The project exposes two coordinated analyzers from `internal/analyzer/`:
- **DependencyAnalyzer**: First pass to detect `Injectable[T]` structs and `Provide[T](fn)` calls, generate constructors, register to global registries
- **AppAnalyzer**: Second pass to generate bootstrap code using registered dependencies

Both analyzers share state through global registries, enabling cross-package dependency resolution.

### Minimal Public API
Only the CLI entry point (`cmd/braider/main.go`) is user-facing. All implementation details are in `internal/` to prevent accidental external dependencies.

### Test Data Isolation
Test fixtures live in `testdata/bootstrapgen/` following analysistest conventions. Each test case is a separate Go package that can be analyzed independently.

### Internal Package Organization
The internal package is split into focused subpackages:
- `internal/analyzer/` - Analyzer definitions (`DependencyAnalyzer`, `AppAnalyzer`) and orchestration
- `internal/detect/` - Detection logic for DI patterns (inject, provide call, app annotations, struct analysis, field analysis, constructor detection, option extraction, namer validation)
- `internal/generate/` - Code generation logic (constructors, bootstrap IIFE) and utilities (AST utilities, code formatting, import management, naming conventions, keyword checking, hash generation)
- `internal/report/` - Diagnostic and suggested fix building
- `internal/registry/` - Global state management (provider registry, injector registry, package tracker)
- `internal/graph/` - Dependency resolution (dependency graph, interface registry, topological sort)
- `internal/loader/` - Package loading utilities for cross-package dependency analysis

### Public API (`pkg/`)
**Location**: `pkg/annotation/` with subpackages `inject/`, `provide/`, `namer/`
**Purpose**: Public annotation types and functions for users to mark DI targets
**Pattern**: Three annotation mechanisms with generic option types:
- `Injectable[T inject.Option]` interface - Embed in structs to mark for constructor generation and DI registration
- `Provide[T provide.Option](fn)` function - Register provider functions via `var _ = annotation.Provide[T](fn)` (local variables in bootstrap IIFE)
- `App(main)` function - Call in main package to mark entry point for bootstrap code generation

**Option subpackages**:
- `inject/` - Options for Injectable: `Default`, `Typed[I]`, `Named[N]`, `WithoutConstructor`
- `provide/` - Options for Provide: `Default`, `Typed[I]`, `Named[N]`
- `namer/` - `Namer` interface for Named option types (must return string literal from `Name()`)

---
_Document patterns, not file trees. New files following patterns should not require updates_

_Updated: 2026-02-02 - Added ProvideFunc annotation, expanded generate package utilities, clarified loader package purpose_
_Updated: 2026-02-11 - Sync: Updated annotation API to current generics-based design (Injectable[T], Provide[T](fn)); added inject/provide/namer subpackages; updated component lists (ProvideCallDetector, BootstrapGenerator, OptionExtractor, NamerValidator); corrected testdata categories_
