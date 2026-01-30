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
**Pattern**: Organized by test category:
- `testdata/src/` - App annotation scenarios (noapp, simpleapp, multipleapp, etc.)
- `testdata/dependency/` - Dependency analysis scenarios (basic, cross_package, missing_constructor, etc.)
- `testdata/constructorgen/` - Constructor generation scenarios

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
- **Detectors**: Find DI patterns (`InjectDetector`, `ProvideDetector`, `AppDetector`, `StructDetector`, `FieldAnalyzer`, `ConstructorAnalyzer`)
- **Generators**: Produce code (`ConstructorGenerator`)
- **Reporters**: Emit diagnostics (`SuggestedFixBuilder`, `DiagnosticEmitter`)
- **Registries**: Track state (`ProviderRegistry`, `InjectorRegistry`, `PackageTracker`)

Components are instantiated in `main.go` and passed to analyzer constructors via dependency injection.

### Multi-Analyzer Pattern
The project exposes two coordinated analyzers from `internal/analyzer/`:
- **DependencyAnalyzer**: First pass to register all `Provide` and `Inject` structs
- **AppAnalyzer**: Second pass to generate bootstrap code using registered dependencies

Both analyzers share state through global registries, enabling cross-package dependency resolution.

### Minimal Public API
Only the CLI entry point (`cmd/braider/main.go`) is user-facing. All implementation details are in `internal/` to prevent accidental external dependencies.

### Test Data Isolation
Test fixtures live in `testdata/src/` following analysistest conventions. Each test case is a separate Go package that can be analyzed independently.

### Internal Package Organization
The internal package is split into focused subpackages:
- `internal/analyzer/` - Analyzer definitions (`DependencyAnalyzer`, `AppAnalyzer`) and orchestration
- `internal/detect/` - Detection logic for DI patterns (inject, provide, app annotations, struct analysis, field analysis, constructor detection)
- `internal/generate/` - Code generation logic (constructors, bootstrap code formatting)
- `internal/report/` - Diagnostic and suggested fix building
- `internal/registry/` - Global state management (provider registry, injector registry, package tracker)
- `internal/graph/` - Dependency resolution (dependency graph, interface registry, topological sort)
- `internal/loader/` - Package loading utilities

### Public API (`pkg/`)
**Location**: `pkg/annotation/`
**Purpose**: Public annotation types and functions for users to mark DI targets
**Pattern**: Three annotation mechanisms:
- `Inject` struct - Embed in structs to mark for constructor generation and DI registration
- `Provide` struct - Embed in structs to mark as providers (local variables in bootstrap IIFE)
- `App(main)` function - Call in main package to mark entry point for bootstrap code generation

---
_Document patterns, not file trees. New files following patterns should not require updates_

_Updated: 2026-01-30 - Added multi-analyzer pattern, expanded internal package organization, updated testdata structure, added App and Provide annotations_
