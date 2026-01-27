# Project Structure

## Organization Philosophy

braider follows a **standard Go project layout** with clear separation between public CLI entry point and internal implementation. The structure prioritizes simplicity and follows Go community conventions for analyzer tools.

## Directory Patterns

### CLI Entry Point
**Location**: `cmd/braider/`
**Purpose**: Minimal CLI wrapper that invokes the analyzer via singlechecker
**Pattern**: Single `main.go` that imports internal analyzer and calls `singlechecker.Main()`

### Internal Implementation
**Location**: `internal/`
**Purpose**: Core analyzer logic, not importable by external packages
**Pattern**:
- `analyzer.go` - Main analyzer definition and run function
- `analyzer_test.go` - Tests using analysistest framework

### Test Fixtures
**Location**: `internal/testdata/src/`
**Purpose**: Go source files used as test inputs for analysistest
**Pattern**: Each test scenario in its own package directory (e.g., `example/`)

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
- **Detectors**: Find DI patterns (`InjectDetector`, `StructDetector`, `FieldAnalyzer`)
- **Generators**: Produce code (`ConstructorGenerator`)
- **Reporters**: Emit diagnostics (`SuggestedFixBuilder`, `DiagnosticEmitter`)

Components are instantiated in `analyzer.go` and orchestrated through a phased pipeline.

### Single Analyzer Pattern
The project exposes one `analysis.Analyzer` variable from the internal package, following the standard pattern for go/analysis tools.

### Minimal Public API
Only the CLI entry point (`cmd/braider/main.go`) is user-facing. All implementation details are in `internal/` to prevent accidental external dependencies.

### Test Data Isolation
Test fixtures live in `testdata/src/` following analysistest conventions. Each test case is a separate Go package that can be analyzed independently.

### Internal Package Organization
The internal package is split into focused subpackages:
- `internal/detect/` - Detection logic for DI patterns (inject markers, struct candidates, fields)
- `internal/generate/` - Code generation logic (constructors, formatting)
- `internal/report/` - Diagnostic and suggested fix building

### Public API (`pkg/`)
**Location**: `pkg/annotation/`
**Purpose**: Public annotation types for users to mark DI targets
**Pattern**: Minimal marker types (e.g., `Inject` struct) that users embed in their structs

---
_Document patterns, not file trees. New files following patterns should not require updates_
