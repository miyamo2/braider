# GitHub Copilot Instructions

This file provides guidance to GitHub Copilot when working with code in this repository.

## Project Overview

**braider** is a go vet analyzer that resolves DI (Dependency Injection) bindings and generates wiring code automatically. It leverages the `SuggestedFix` feature of the go/analysis package to auto-generate constructors and bootstrap code.

### Key Features
- Static analysis of Go code to detect DI patterns via annotations
- Automatic generation of constructor functions
- DI wiring code generation via suggested fixes
- Bootstrap code generation with IIFE pattern
- Integration with `go vet` workflow

### Inspiration
This project is inspired by [google/wire](https://github.com/google/wire), which provides compile-time dependency injection for Go.

## Tech Stack

- **Go Version**: 1.24
- **Core Package**: `golang.org/x/tools/go/analysis`
- **Key Feature**: `analysis.SuggestedFix` for code generation

## Architecture & Design

### Multi-Analyzer Architecture

braider uses two specialized analyzers that work together:

1. **DependencyAnalyzer** (`braider_dependency`):
   - Scans all packages for `annotation.Injectable` structs and `annotation.Provide[T](fn)` calls
   - Generates constructors for Injectable structs via SuggestedFix
   - Registers providers and injectors to global registries
   - Validates provider function signatures for Provide calls

2. **AppAnalyzer** (`braider_app`):
   - Detects `annotation.App(main)` annotations
   - Validates App annotation usage (single annotation, main function reference)
   - Generates bootstrap code using IIFE pattern
   - Creates dependency struct with all Injectable structs as fields

### Dependency Injection Flow

```go
// User code with annotations
type MyRepository struct {
    annotation.Injectable[inject.Default]
}

type MyService struct {
    annotation.Injectable[inject.Default]
    repo MyRepository
}

var _ = annotation.App(main)

func main() {}
```

Analyzer generates:
1. Constructors: `NewMyRepository()`, `NewMyService(repo MyRepository)`
2. Bootstrap code in main with topologically sorted initialization

### Core Components

#### Registries (shared state across analyzers)
- **ProviderRegistry**: Tracks Provide-annotated provider functions and their dependencies
- **InjectorRegistry**: Tracks Inject-annotated structs and their dependencies
- **PackageTracker**: Monitors which packages have been scanned

#### Detectors (pattern recognition)
- **AppDetector**: Detects and validates `annotation.App(main)` annotations
- **ProvideCallDetector**: Detects `var _ = annotation.Provide[T](fn)` package-level calls
- **InjectDetector**: Detects `annotation.Injectable` fields in structs
- **StructDetector**: Combines struct + Inject field detection
- **FieldAnalyzer**: Extracts injectable fields from structs (excluding annotation fields)
- **ConstructorAnalyzer**: Analyzes existing constructors and extracts dependencies

#### Generators
- **ConstructorGenerator**: Generates constructor function code
- **BootstrapGenerator**: Generates IIFE bootstrap code (TODO)

#### Reporters
- **SuggestedFixBuilder**: Builds `analysis.SuggestedFix` for code generation
- **DiagnosticEmitter**: Emits diagnostics with suggested fixes

#### Graph Processing
- **DependencyGraph**: Builds dependency graph from providers and injectors
- **TopologicalSort**: Resolves initialization order
- **InterfaceRegistry**: Maps interface types to implementing structs

### Dependency Injection System

The DI system uses three marker types:

1. **`annotation.Injectable[T]`**: Marks structs as DI targets (become fields in dependency struct)
2. **`annotation.Provide[T](fn)`**: Registers provider functions as DI sources (`var _ = annotation.Provide[provide.Default](NewRepo)`)
3. **`annotation.App(main)`**: Triggers bootstrap code generation in main function

## Development Guidelines

### Coding Standards
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Adhere to [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- Use meaningful variable and function names
- Keep functions focused and small

### Testing
- Use `analysistest` package for analyzer testing
- Create testdata directories with Go source files
- Test both positive cases (should report) and negative cases (should not report)
- Example testdata location: `internal/analyzer/testdata/`

### Commit Messages
- Follow [Conventional Commits](https://www.conventionalcommits.org/v1.0.0/) specification
- Format: `<type>(<scope>): <description>`
- When Copilot commits, please add the following `Co-authored-by trailer` to the end of the commit message to indicate which AI agent performed the work:
  `Co-authored-by: Copilot <175728472+Copilot@users.noreply.github.com>`

#### Types
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only changes
- `style`: Code style changes (formatting, missing semicolons, etc.)
- `refactor`: Code refactoring without feature changes or bug fixes
- `test`: Adding or modifying tests
- `chore`: Maintenance tasks (build process, dependencies, etc.)

## Build & Test Commands

```bash
# Build all packages
go build ./...

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a single test file
go test -v ./internal/analyzer/app_test.go

# Run a specific test
go test -v -run TestAppAnalyzer ./internal/analyzer

# Build the analyzer binary
go build -o braider ./cmd/braider

# Run the analyzer via go vet
go vet -vettool=$(which braider) ./...

# Run with suggested fixes applied
go vet -vettool=$(which braider) -fix ./...
```

## Directory Structure

```
braider/
├── cmd/
│   └── braider/
│       └── main.go              # CLI entry point with DI wiring
├── pkg/
│   └── annotation/
│       └── annotation.go        # Public annotation types
├── internal/
│   ├── analyzer/
│   │   ├── app.go              # AppAnalyzer implementation
│   │   ├── dependency.go       # DependencyAnalyzer implementation
│   │   └── testdata/           # Test fixtures
│   ├── detect/                 # Pattern detection components
│   ├── generate/               # Code generation components
│   ├── graph/                  # Dependency graph and sorting
│   ├── loader/                 # Package loading utilities
│   ├── registry/               # Global registries
│   └── report/                 # Diagnostic and fix reporting
```

## Usage Example

```bash
# Install
go install github.com/miyamo2/braider/cmd/braider@latest

# Run analysis
go vet -vettool=$(which braider) ./...

# Apply suggested fixes
go vet -vettool=$(which braider) -fix ./...
```
