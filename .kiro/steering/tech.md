# Technology Stack

## Architecture

braider implements the `analysis.Analyzer` interface from `golang.org/x/tools/go/analysis`, enabling integration with the Go toolchain as a vet tool. The analyzer performs static analysis on Go AST and proposes code fixes via `SuggestedFix`.

## Core Technologies

- **Language**: Go 1.24
- **Framework**: `golang.org/x/tools/go/analysis` (Go analyzer framework)
- **Runtime**: Standard Go toolchain (`go vet`)

## Key Libraries

- **`golang.org/x/tools/go/analysis`**: Core analyzer interface and diagnostic reporting
- **`golang.org/x/tools/go/analysis/passes/inspect`**: AST inspection utilities
- **`golang.org/x/tools/go/ast/inspector`**: Efficient AST traversal
- **`golang.org/x/tools/go/analysis/singlechecker`**: CLI wrapper for single-analyzer tools

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

### AST Inspector Pattern
Uses `inspect.Analyzer` as a dependency for efficient AST traversal, following the recommended pattern for go/analysis tools.

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
