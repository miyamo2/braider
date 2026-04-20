# Technology Stack

## Architecture

braider implements a **phased pipeline architecture** using `phasedchecker.Main()` with two coordinated phases (`dependency` → `app`). `Aggregator.AfterDependencyPhase` populates shared registries between phases from per-package `*DependencyResult` values. Each analyzer implements `analysis.Analyzer`, performs static analysis on Go AST, and proposes code fixes via `SuggestedFix`.

Pipeline is configured via `phasedchecker.Config` with explicit `Pipeline` and `DiagnosticPolicy`. `DiagnosticPolicy` uses `DefaultSeverity: SeverityWarn` for diagnostics outside explicit category rules.

> Implementation details: see `.claude/rules/architecture.md`, `.claude/rules/internal-layout.md`.

## Core Technologies

- **Language**: go 1.25
- **Framework**: `golang.org/x/tools/go/analysis` (Go analyzer framework)
- **Pipeline**: `github.com/miyamo2/phasedchecker` (phased multi-analyzer orchestration)
- **Runtime**: Standalone binary via `phasedchecker.Main()`

## Key Libraries

- `golang.org/x/tools/go/analysis` — core analyzer interface and diagnostic reporting
- `golang.org/x/tools/go/analysis/passes/inspect` — AST inspection utilities
- `golang.org/x/tools/go/ast/inspector` — efficient AST traversal
- `golang.org/x/tools/go/analysis/checker` — `checker.Graph` used by `Aggregator.AfterDependencyPhase` to iterate per-package results
- `github.com/miyamo2/phasedchecker` — `Main()`, `Config`, `Pipeline`, `Phase`, `DiagnosticPolicy`, `CategoryRule`, `SeverityCritical`, `SeverityWarn`

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
- Use `phasedchecker/checkertest` with isolated registry instances
- All e2e tests consolidated in a single table-driven `TestIntegration`
- Test both positive and negative cases
- See `.claude/rules/testing.md` for testdata conventions, golden workflow, and the 82-case catalog

## Development Environment

### Required Tools
- go 1.25+
- Standard Go toolchain

### Common Commands
```bash
go build ./...
go test ./...
braider ./...
braider -fix ./...
```

## Key Technical Decisions

Each decision below records **why** the approach was chosen. Implementation details live in `.claude/rules/`.

### SuggestedFix for Code Generation
Code generation is delivered as `analysis.SuggestedFix` rather than separate codegen tooling. Enables `braider -fix` workflow, IDE quick actions, and atomic related-change application. See `.claude/rules/code-generation.md`.

### Component-Based Architecture
The analyzer uses composable components (detectors, generators, reporters, registries, graph, analyzer) wired via braider's own DI annotations in `cmd/braider/main.go` (dogfooding). Each component has a single responsibility and is testable in isolation. Component inventory: `.claude/rules/internal-layout.md`.

### AST-Based Code Generation
Both constructor and bootstrap code build `go/ast` trees programmatically and render via `format.Node`, rather than string concatenation. Produces correctly formatted Go without a separate formatting pass; eliminates `CodeFormatter` from the dependency graph; removes whole classes of string-interpolation bugs. Helpers and rendering functions: `.claude/rules/code-generation.md`.

### AST Inspector Pattern
Uses `inspect.Analyzer` as a dependency for efficient AST traversal, following the recommended pattern for go/analysis tools.

### Cross-Phase State via Registries and Aggregator
Two phases share state via thread-safe registries populated by the `Aggregator.AfterDependencyPhase` callback. `DependencyAnalyzer.Run()` returns `*DependencyResult` per package rather than writing directly to registries — isolates per-package analysis from shared-state mutation. Duplicate registrations are deferred via `DuplicateRegistry` for reporting by `AppAnalyzer`. See `.claude/rules/architecture.md`.

### Bootstrap Struct Field vs Local Variable
`Node.IsField` determines how a dependency appears in the bootstrap IIFE. Injectable/Provide become struct fields (caller-accessible); Variable nodes become local variables (not exposed). Rationale: Variable annotations often reference pre-existing external state (`os.Stdout`) that need not leak into the returned container.

### Cross-Package Constructor Qualification
When a `Provide[T](fn)` returns a type from a package **different** than where the function is defined, bootstrap uses two separate qualifiers — return-type package (for field types) and constructor package (for call sites). Both are tracked on `Node`. `ConstructorPkgPath` is included in hash computation only when it differs from `PackagePath`. See `.claude/rules/code-generation.md`.

### Variable Expression Handling
Variable annotations accept only identifiers or package-qualified selectors. Aliased imports are normalized to declared package names for consistent hashing and later rewritten with collision-safe aliases during bootstrap generation. See `.claude/rules/annotations.md`.

### App Options and Container Definition
`App[T](main)` type parameter selects between `app.Default` (anonymous struct output) and `app.Container[T]` (user-defined container). Container fields use the same `braider` tag convention as Injectable fields, with stricter rules (`-` / empty not permitted). Detect-validate-resolve-generate pipeline documented in `.claude/rules/annotations.md`. Tag rules: `.claude/rules/struct-tags.md`.

### Struct Tag Field Control
`Injectable[T]` fields can use `braider` tags for field-level DI customization (`name` / `-`). Empty and conflicting tags emit diagnostic errors. See `.claude/rules/struct-tags.md`.

### Idempotent Code Generation
Bootstrap uses hash markers (`// braider:hash:<hash>`) to skip regeneration when the dependency graph is unchanged. Preserves manual edits in unrelated sections. Hash inputs and idempotency semantics: `.claude/rules/code-generation.md`.

### Dogfooding (Self-Hosting)
braider uses its own annotations in `cmd/braider/main.go`. `App[app.Container[T]](main)` generates the dependency IIFE that wires all internal components. Validates braider against a real, non-trivial dependency graph on every build.

### Conventional Commits
Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/v1.0.0/): `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `style`.

---
_Document standards and patterns, not every dependency_

_Updated: 2026-02-02 - Added loader component, expanded generator utilities to include naming and keyword checking_
_Updated: 2026-02-11 - Sync: Corrected annotation names to current API; added OptionExtractor, NamerValidator to detect components; added AST utilities to generate components_
_Updated: 2026-02-12 - Sync: Added Variable annotation support (VariableCallDetector, VariableRegistry, Variable expression handling, blank assignment for unused Variable nodes)_
_Updated: 2026-02-15 - Sync: Added Bootstrap struct field vs local variable pattern (Provide now IsField=true); added struct tag field control pattern (braider:"name", braider:"-")_
_Updated: 2026-02-15 - Sync: Added internal/annotation marker interface layer; added dogfooding (self-hosting) pattern for cmd/braider/main.go_
_Updated: 2026-02-16 - Sync: App annotation now generic App[T](main) with app option type parameter; added App Options and Container Definition pattern (app.Default, app.Container[T]); added AppOptionExtractor, ContainerValidator, ContainerResolver to component lists; added AppOption/AppDefault/AppContainer marker interfaces_
_Updated: 2026-02-18 - Sync: Added cross-package constructor qualification pattern (ConstructorPkgPath/ConstructorPkgName separation from PackagePath/PackageName for Provide nodes returning types from different packages); added marker resolution to detect component list_
_Updated: 2026-02-20 - Sync: Code generation refactored from string concatenation to AST-based approach (go/ast + format.Node); CodeFormatter component removed; added AST-Based Code Generation technical decision; generate package now uses ast_builder.go helpers and renderDecl/renderNode; report package delegates import rendering to generate.RenderImportBlock_
_Updated: 2026-02-25 - Sync: Migrated from multichecker.Main() to phasedchecker.Main() phased pipeline architecture; replaced analysistest with phasedchecker/checkertest; added Aggregator/DependencyResult cross-phase coordination pattern; replaced Global Registry Pattern with Cross-Phase State section; removed PackageTracker (no longer exists); added DuplicateRegistry; updated dogfooding to use app.Container[T] and phasedchecker.Config; added analyzer package to component list_
_Updated: 2026-02-26 - Sync: Updated testing section to reflect consolidated table-driven TestIntegration pattern (all e2e tests in single test function using RunWithSuggestedFixes); added dependency-only smoke test pattern_
_Updated: 2026-03-02 - Sync: DiagnosticPolicy now includes DefaultSeverity (SeverityWarn) for diagnostics not matching explicit category rules; added SeverityWarn to phasedchecker library references_
_Updated: 2026-04-21 - Sync: Extracted implementation details into .claude/rules/ (architecture, annotations, struct-tags, code-generation, testing, internal-layout); steering now retains intent/WHY, references rules for details_
