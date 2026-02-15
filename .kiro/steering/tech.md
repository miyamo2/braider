# Technology Stack

## Architecture

braider implements a **multi-analyzer architecture** using `multichecker.Main()` with two coordinated analyzers:
- **DependencyAnalyzer**: Detects `annotation.Injectable[T]` structs, `annotation.Provide[T](fn)` calls, and `annotation.Variable[T](value)` calls; generates constructors; registers to global registries
- **AppAnalyzer**: Detects `annotation.App(main)` and generates bootstrap IIFE code using all registered providers, injectors, and variables

Each analyzer implements the `analysis.Analyzer` interface, performs static analysis on Go AST, and proposes code fixes via `SuggestedFix`. Analyzers share state through global registries for cross-package dependency resolution.

## Core Technologies

- **Language**: go 1.25
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
- go 1.25+
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
The analyzer uses composable components (detectors, generators, reporters) wired via braider's own DI annotations in `cmd/braider/main.go` (dogfooding). Each component has a single responsibility and is testable in isolation. Components are organized by concern:
- **Annotation markers** (`internal/annotation/`): Marker interfaces (`Injectable`, `Provider`, `Variable`, `App`) embedded by public `pkg/annotation/` types; defines the type-level contracts that detectors match against
- **Detectors** (`internal/detect/`): Pattern matching (inject, provide call, variable call, app, struct, field, constructor, option extraction, namer validation)
- **Generators** (`internal/generate/`): Code generation (constructors, bootstrap IIFE), utilities (imports, formatting, naming, keyword checking, hash markers for idempotency, AST utilities)
- **Reporters** (`internal/report/`): Diagnostic and suggested fix building
- **Registries** (`internal/registry/`): Global state for cross-package dependency tracking (provider, injector, variable, package tracker)
- **Graph** (`internal/graph/`): Dependency graph construction, interface resolution, topological sorting; Variable nodes participate in graph as zero-dependency leaves
- **Loader** (`internal/loader/`): Package loading utilities for cross-package analysis

### AST Inspector Pattern
Uses `inspect.Analyzer` as a dependency for efficient AST traversal, following the recommended pattern for go/analysis tools.

### Global Registry Pattern
Uses shared registries (`ProviderRegistry`, `InjectorRegistry`, `VariableRegistry`, `PackageTracker`) to accumulate DI information across multiple packages and analyzer passes. This enables cross-package dependency resolution and ensures the `AppAnalyzer` can access all bindings collected by `DependencyAnalyzer`.

### Bootstrap Struct Field vs Local Variable
In the dependency graph, `Node.IsField` determines how a dependency appears in the bootstrap IIFE:
- **IsField=true** (Injectable and Provide nodes): Become fields in the returned dependency struct, accessible to the caller
- **IsField=false** (Variable nodes): Become local variables within the IIFE, not exposed to the caller

### Variable Expression Handling
Variable annotations accept only simple identifiers (`myVar`) or package-qualified identifiers (`os.Stdout`). The detector normalizes aliased import qualifiers to declared package names (e.g., `import myos "os"` with `myos.Stdout` becomes `os.Stdout` in `ExpressionText`). During bootstrap generation, expression aliases are rewritten if package name collisions occur. Variable nodes that are not depended upon by other nodes use blank assignments (`_ =`) to avoid unused variable errors.

### Struct Tag Field Control
Fields in `Injectable[T]` structs can use `braider` struct tags for field-level DI customization:
- `braider:"name"` - Resolve the field as a named dependency (equivalent to `Named[N]` at the field level)
- `braider:"-"` - Exclude the field from DI resolution entirely (field is not wired in the constructor)
- Empty tag `braider:""` emits a diagnostic error (ambiguous intent)
- Conflicting tags (e.g., `braider:"name"` on a field whose type is already registered with a different name) emit a diagnostic error

### Idempotent Code Generation
Bootstrap code generation uses hash markers (`// braider:hash:<hash>`) to track dependency graph state. The generator compares current graph hash against existing hash comments to determine if regeneration is needed. This prevents unnecessary rewrites and preserves manual edits in unrelated code sections.

### Dogfooding (Self-Hosting)
braider uses its own annotations in `cmd/braider/main.go` to wire its internal components. The entry point declares `annotation.Variable` for shared values (e.g., `bootstrapCtx`, `bootstrapCancel`) and `annotation.App(main)` to trigger bootstrap generation. braider then generates the `dependency` IIFE that constructs and wires all detectors, generators, reporters, registries, graph builders, and analyzers. This ensures braider validates its own code generation against a real, non-trivial dependency graph.

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

_Updated: 2026-02-02 - Added loader component, expanded generator utilities to include naming and keyword checking_
_Updated: 2026-02-11 - Sync: Corrected annotation names to current API; added OptionExtractor, NamerValidator to detect components; added AST utilities to generate components_
_Updated: 2026-02-12 - Sync: Added Variable annotation support (VariableCallDetector, VariableRegistry, Variable expression handling, blank assignment for unused Variable nodes)_
_Updated: 2026-02-15 - Sync: Added Bootstrap struct field vs local variable pattern (Provide now IsField=true); added struct tag field control pattern (braider:"name", braider:"-")_
_Updated: 2026-02-15 - Sync: Added internal/annotation marker interface layer; added dogfooding (self-hosting) pattern for cmd/braider/main.go_
