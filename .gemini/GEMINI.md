# CLAUDE.md - braider

## Project Overview

**braider** is a go vet analyzer that resolves DI (Dependency Injection) bindings and generates wiring code automatically. It leverages the `SuggestedFix` feature of the go/analysis package to auto-generate constructors and DI container wiring.

### Key Features
- Static analysis of Go code to detect DI patterns
- Automatic generation of constructor functions
- DI wiring code generation via suggested fixes
- Integration with `go vet` workflow

### Inspiration
This project is inspired by [google/wire](https://github.com/google/wire), which provides compile-time dependency injection for Go. braider aims to provide similar functionality through the go vet analyzer interface.

## Tech Stack

- **Go Version**: 1.24
- **Core Package**: `golang.org/x/tools/go/analysis`
- **Key Feature**: `analysis.SuggestedFix` for code generation

## Architecture & Design

### Analyzer Implementation

braider implements the `analysis.Analyzer` interface:

```go
var Analyzer = &analysis.Analyzer{
    Name: "braider",
    Doc:  "resolves DI bindings and generates wiring",
    Run:  run,
}
```

### Code Generation via SuggestedFix

The analyzer uses `analysis.SuggestedFix` to propose code changes that can be automatically applied:

```go
analysis.Diagnostic{
    Pos:     pos,
    Message: "missing constructor",
    SuggestedFixes: []analysis.SuggestedFix{
        {
            Message: "generate constructor",
            TextEdits: []analysis.TextEdit{
                {
                    Pos:     pos,
                    End:     end,
                    NewText: []byte(generatedCode),
                },
            },
        },
    },
}
```

### DI Wiring Patterns

braider analyzes struct dependencies and generates wiring code that:
- Identifies injectable dependencies
- Resolves dependency graphs
- Generates initialization code in topological order

## Development Guidelines

### Coding Standards
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Adhere to [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- Use meaningful variable and function names
- Keep functions focused and small

### Error Messages
- Error messages should be clear and actionable
- Include position information for easy navigation
- Suggest fixes when possible

### Testing
- Use `analysistest` package for analyzer testing
- Create testdata directories with Go source files
- Test both positive cases (should report) and negative cases (should not report)

### Commit Messages
- Follow [Conventional Commits](https://www.conventionalcommits.org/v1.0.0/) specification
- Format: `<type>(<scope>): <description>`
- When Gemini CLI commits, please add the following `Co-authored-by trailer` to the end of the commit message to indicate which AI agent performed the work.
  `Co-Authored-By: gemini-cli <218195315+gemini-cli@users.noreply.github.com>`

#### Types
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only changes
- `style`: Code style changes (formatting, missing semicolons, etc.)
- `refactor`: Code refactoring without feature changes or bug fixes
- `test`: Adding or modifying tests
- `chore`: Maintenance tasks (build process, dependencies, etc.)

#### Examples
```
feat(analyzer): add support for interface injection
fix(wiring): resolve circular dependency detection
docs: update README with installation instructions
test(analyzer): add test cases for nested structs
refactor(core): simplify dependency graph traversal
chore: update golang.org/x/tools dependency
```

## Build & Test Commands

```bash
# Build all packages
go build ./...

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run the analyzer via go vet
go vet -vettool=$(which braider) ./...

# Run with suggested fixes applied
go vet -vettool=$(which braider) -fix ./...
```

## Directory Structure

```
braider/
├── .claude/
│   └── CLAUDE.md        # This file
├── README.md            # Project readme
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── cmd/
│   └── braider/
│       └── main.go      # CLI entry point
├── internal/
│   ├── analyzer.go      # Main analyzer implementation
│   └── analyzer_test.go # Analyzer tests
└── testdata/
    └── src/
        └── example/     # Test fixtures
            └── *.go
```

## Usage Example

Once implemented, braider can be used as follows:

```bash
# Install
go install github.com/miyamo2/braider/cmd/braider@latest

# Run analysis
go vet -vettool=$(which braider) ./...

# Apply suggested fixes
go vet -vettool=$(which braider) -fix ./...
```
