# Technology Stack

## Architecture

braider implements a **multi-analyzer architecture** using `multichecker.Main()` with two coordinated analyzers:
- **DependencyAnalyzer**: Detects and registers `Provide` and `Inject` structs, generates constructors
- **AppAnalyzer**: Detects `App` annotations and generates bootstrap IIFE code

Each analyzer implements the `analysis.Analyzer` interface, performs static analysis on Go AST, and proposes code fixes via `SuggestedFix`. Analyzers share state through global registries for cross-package dependency resolution.

## Core Technologies

- **Language**: Go 1.24
- **Framework**: `golang.org/x/tools/go/analysis` (Go analyzer framework)
- **Runtime**: Standard Go toolchain (`go vet`)

## Key Libraries

- **`golang.org/x/tools/go/analysis`**: Core analyzer interface and diagnostic reporting
- **`golang.org/x/tools/go/analysis/passes/inspect`**: AST inspection utilities
- **`golang.org/x/tools/go/ast/inspector`**: Efficient AST traversal
- **`golang.org/x/tools/go/analysis/multichecker`**: CLI wrapper for running multiple analyzers with shared state

## Development Standards

### Code Quality
- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Adhere to [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- Keep functions focused and small
- Use meaningful variable and function names

### Error Handling
- Error messages should be clear and actionable
- Include position information for easy navigation in IDEs
- Suggest fixes when possible via `SuggestedFix`

### Testing
- Use `analysistest` package for analyzer testing
- Create testdata directories with Go source fixtures
- Test both positive cases (should report) and negative cases (should not report)

## Development Environment

### Required Tools
- Go 1.24+
- Standard Go toolchain

### Common Commands
```bash
# Build all packages
go build ./...

# Run all tests
go test ./...

# Run analyzer via go vet
go vet -vettool=$(which braider) ./...

# Apply suggested fixes
go vet -vettool=$(which braider) -fix ./...
```

## Key Technical Decisions

### SuggestedFix for Code Generation
Code generation is implemented via `analysis.SuggestedFix` rather than separate codegen tools. This enables:
- Integration with `go vet -fix` workflow
- IDE integration (fixes appear as quick actions)
- Atomic application of related changes

### Component-Based Architecture
The analyzer uses composable components (detectors, generators, reporters) instantiated in `main.go` and passed to analyzer constructors. Each component has a single responsibility and is testable in isolation. Components are organized by concern:
- **Detectors** (`internal/detect/`): Pattern matching (inject, provide, app, struct, field, constructor)
- **Generators** (`internal/generate/`): Code generation (constructors)
- **Reporters** (`internal/report/`): Diagnostic and suggested fix building
- **Registries** (`internal/registry/`): Global state for cross-package dependency tracking
- **Graph** (`internal/graph/`): Dependency graph construction, interface resolution, topological sorting

### AST Inspector Pattern
Uses `inspect.Analyzer` as a dependency for efficient AST traversal, following the recommended pattern for go/analysis tools.

### Global Registry Pattern
Uses shared registries (`ProviderRegistry`, `InjectorRegistry`, `PackageTracker`) to accumulate DI information across multiple packages and analyzer passes. This enables cross-package dependency resolution and ensures the `AppAnalyzer` can access all bindings collected by `DependencyAnalyzer`.

### Conventional Commits
Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/v1.0.0/) specification:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions/modifications
- `refactor`: Code refactoring
- `chore`: Maintenance tasks

---
_Document standards and patterns, not every dependency_

_Updated: 2026-01-30 - Added multi-analyzer architecture, global registry pattern, expanded component organization_
