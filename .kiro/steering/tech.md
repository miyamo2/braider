# Technology Stack

## Architecture

braider implements a **phased pipeline architecture** using `phasedchecker.Main()` with two coordinated phases:
- **Phase "dependency"** (`DependencyAnalyzer`): Runs per-package; detects `annotation.Injectable[T]` structs, `annotation.Provide[T](fn)` calls, and `annotation.Variable[T](value)` calls; generates constructors; returns `*DependencyResult` per package
- **Phase "app"** (`AppAnalyzer`): Runs on main package after all dependency phase packages complete; detects `annotation.App[T](main)` and generates bootstrap IIFE code; supports default (anonymous struct) and container (user-defined struct) output modes via the App option type parameter

Between phases, `Aggregator.AfterDependencyPhase` iterates all per-package `DependencyResult` values (via `checker.Graph`) and populates shared registries. Duplicate registrations are collected into `DuplicateRegistry` for deferred reporting by AppAnalyzer. Each analyzer implements the `analysis.Analyzer` interface, performs static analysis on Go AST, and proposes code fixes via `SuggestedFix`.

The pipeline is configured via `phasedchecker.Config` with explicit `Pipeline` (phase ordering, per-phase analyzers, AfterPhase callbacks) and `DiagnosticPolicy` (category-to-severity mappings that can abort the pipeline).

## Core Technologies

- **Language**: go 1.25
- **Framework**: `golang.org/x/tools/go/analysis` (Go analyzer framework)
- **Pipeline**: `github.com/miyamo2/phasedchecker` (phased multi-analyzer orchestration)
- **Runtime**: Standard Go toolchain (`go vet`)

## Key Libraries

- **`golang.org/x/tools/go/analysis`**: Core analyzer interface and diagnostic reporting
- **`golang.org/x/tools/go/analysis/passes/inspect`**: AST inspection utilities
- **`golang.org/x/tools/go/ast/inspector`**: Efficient AST traversal
- **`github.com/miyamo2/phasedchecker`**: Phased pipeline orchestration with `phasedchecker.Main()`, `Config`, `Pipeline`, `Phase`, `DiagnosticPolicy`, `CategoryRule`, `SeverityCritical`
- **`golang.org/x/tools/go/analysis/checker`**: `checker.Graph` used by `Aggregator.AfterDependencyPhase` to iterate per-package analysis results

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
- Use `phasedchecker/checkertest` package for analyzer testing
- All e2e tests consolidated in a single table-driven `TestIntegration` using `checkertest.RunWithSuggestedFixes`
- Tests build a `phasedchecker.Config` with the same Pipeline/DiagnosticPolicy as production, using isolated registry instances
- Create testdata directories with Go source fixtures under `testdata/e2e/`
- Test both positive cases (should report) and negative cases (should not report)
- Dependency-only smoke tests (no App annotation, no golden files) share the same test runner; `RunWithSuggestedFixes` verifies no unexpected diagnostics

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
- **Annotation markers** (`internal/annotation/`): Marker interfaces (`Injectable`, `Provider`, `Variable`, `App`, `AppOption`, `AppDefault`, `AppContainer`) embedded by public `pkg/annotation/` types; defines the type-level contracts that detectors match against
- **Detectors** (`internal/detect/`): Pattern matching (inject, provide call, variable call, app, struct, field, constructor, option extraction, namer validation, app option extraction, container definition models, marker resolution)
- **Generators** (`internal/generate/`): AST-based code generation (constructors, bootstrap IIFE) via `go/ast` + `format.Node`; utilities (imports, naming, keyword checking, hash markers for idempotency, AST builder helpers)
- **Reporters** (`internal/report/`): Diagnostic and suggested fix building; diagnostic category constants (`CategoryOptionValidation`, `CategoryExpressionValidation`, `CategoryDependencyRegistration`) map to `SeverityCritical` in phasedchecker
- **Registries** (`internal/registry/`): Shared state for cross-phase dependency tracking (provider, injector, variable, duplicate)
- **Graph** (`internal/graph/`): Dependency graph construction, interface resolution, topological sorting, container validation, container field resolution; Variable nodes participate in graph as zero-dependency leaves
- **Loader** (`internal/loader/`): Package loading utilities for cross-package analysis
- **Analyzer** (`internal/analyzer/`): `Aggregator` (AfterPhase callback for registry population), `DependencyResult` (per-package result type), `DependencyAnalyzeRunner`, `AppAnalyzeRunner`

### AST-Based Code Generation
Both constructor and bootstrap code generation build `go/ast` trees programmatically and render them via `format.Node`, rather than using string concatenation (`fmt.Sprintf`, `strings.Builder`). This approach:
- Produces correctly formatted Go code without a separate formatting pass
- Eliminates an entire component (`CodeFormatter`) from the dependency graph
- Makes structural code manipulation safer (no string interpolation bugs)
- Uses `ast_builder.go` helpers (`astIdent`, `astSelector`, `astStructType`, `astShortVar`, `astFuncDecl`, `astVarDecl`, etc.) to construct AST nodes concisely
- Renders declarations via `renderDecl` (wraps in dummy file, assigns synthetic positions, strips package prefix) and expressions via `renderNode`
- Import blocks use `RenderImportBlock` (shared by both generate and report packages)

### AST Inspector Pattern
Uses `inspect.Analyzer` as a dependency for efficient AST traversal, following the recommended pattern for go/analysis tools.

### Cross-Phase State via Registries and Aggregator
The two phases share state via shared registries (thread-safe, `sync.RWMutex`), populated by the `Aggregator.AfterDependencyPhase` callback between phases:
- `ProviderRegistry` / `InjectorRegistry` / `VariableRegistry`: nested `map[TypeName]map[Name]*Info`
- `DuplicateRegistry`: collects duplicate named dependency registrations across packages; reported by AppAnalyzer as Critical diagnostics

`DependencyAnalyzer.Run()` returns `*DependencyResult` per package (providers, injectors, variables) instead of writing directly to registries. After all packages in the dependency phase complete, the Aggregator iterates the `checker.Graph`, extracts each result, and populates the shared registries.

### Bootstrap Struct Field vs Local Variable
In the dependency graph, `Node.IsField` determines how a dependency appears in the bootstrap IIFE:
- **IsField=true** (Injectable and Provide nodes): Become fields in the returned dependency struct, accessible to the caller
- **IsField=false** (Variable nodes): Become local variables within the IIFE, not exposed to the caller

### Cross-Package Constructor Qualification
When a `Provide[T](fn)` registers a function that returns a type from a different package than where the function is defined, the bootstrap generator must use two separate package qualifiers:
- **Return type's package** (`PackagePath`/`PackageName`): Used for struct field type qualification (e.g., `analysis.Analyzer`)
- **Constructor function's package** (`ConstructorPkgPath`/`ConstructorPkgName`): Used for function call qualification (e.g., `analyzer.NewAppAnalyzer(...)`)

The `Node` struct carries both sets of fields (`PackagePath`/`PackageName` and `ConstructorPkgPath`/`ConstructorPkgName`/`ConstructorPkgAlias`). Import collection and collision detection consider both packages. `ConstructorPkgPath` is included in hash computation only when it differs from `PackagePath`, preserving backward compatibility for same-package providers.

### Variable Expression Handling
Variable annotations accept only simple identifiers (`myVar`) or package-qualified identifiers (`os.Stdout`). The detector normalizes aliased import qualifiers to declared package names (e.g., `import myos "os"` with `myos.Stdout` becomes `os.Stdout` in `ExpressionText`). During bootstrap generation, expression aliases are rewritten if package name collisions occur. Variable nodes that are not depended upon by other nodes use blank assignments (`_ =`) to avoid unused variable errors.

### App Options and Container Definition
The `App[T](main)` annotation uses a generic type parameter to configure bootstrap output mode:
- **`app.Default`**: Standard mode; bootstrap returns an anonymous struct with all dependencies as fields (default behavior)
- **`app.Container[T]`**: Container mode; `T` is a user-defined struct type (named or anonymous) whose fields map to resolved dependencies; bootstrap returns an instance of `T`

Container fields use the same `braider` struct tag convention as Injectable fields:
- `braider:"name"` - Resolve the field to a named dependency
- No tag - Resolve by type (concrete or interface)
- `braider:"-"` and `braider:""` are not permitted on container fields (emit diagnostic errors)

The container pipeline follows a detect-validate-resolve-generate pattern:
- **AppOptionExtractor** classifies the type argument as Default or Container and extracts the `ContainerDefinition` model
- **ContainerValidator** validates all container fields are resolvable against the dependency graph before generation
- **ContainerResolver** maps each container field to its dependency graph node key and bootstrap variable name
- **BootstrapGenerator.GenerateContainerBootstrap** produces the typed IIFE code with a struct literal return

Mixed options (combining Container with other App options) are supported via anonymous interface embedding.

### Struct Tag Field Control
Fields in `Injectable[T]` structs can use `braider` struct tags for field-level DI customization:
- `braider:"name"` - Resolve the field as a named dependency (equivalent to `Named[N]` at the field level)
- `braider:"-"` - Exclude the field from DI resolution entirely (field is not wired in the constructor)
- Empty tag `braider:""` emits a diagnostic error (ambiguous intent)
- Conflicting tags (e.g., `braider:"name"` on a field whose type is already registered with a different name) emit a diagnostic error

### Idempotent Code Generation
Bootstrap code generation uses hash markers (`// braider:hash:<hash>`) to track dependency graph state. The generator compares current graph hash against existing hash comments to determine if regeneration is needed. This prevents unnecessary rewrites and preserves manual edits in unrelated code sections.

### Dogfooding (Self-Hosting)
braider uses its own annotations in `cmd/braider/main.go` to wire its internal components. The entry point declares `annotation.App[app.Container[T]](main)` with a container struct that exposes the two analyzers and the Aggregator. braider then generates the `dependency` IIFE that constructs and wires all detectors, generators, reporters, registries, graph builders, and analyzers. The generated container is consumed by `phasedchecker.Main()` to configure the phased pipeline (Pipeline with phases, AfterPhase callbacks, DiagnosticPolicy). This ensures braider validates its own code generation against a real, non-trivial dependency graph.

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
_Updated: 2026-02-16 - Sync: App annotation now generic App[T](main) with app option type parameter; added App Options and Container Definition pattern (app.Default, app.Container[T]); added AppOptionExtractor, ContainerValidator, ContainerResolver to component lists; added AppOption/AppDefault/AppContainer marker interfaces_
_Updated: 2026-02-18 - Sync: Added cross-package constructor qualification pattern (ConstructorPkgPath/ConstructorPkgName separation from PackagePath/PackageName for Provide nodes returning types from different packages); added marker resolution to detect component list_
_Updated: 2026-02-20 - Sync: Code generation refactored from string concatenation to AST-based approach (go/ast + format.Node); CodeFormatter component removed; added AST-Based Code Generation technical decision; generate package now uses ast_builder.go helpers and renderDecl/renderNode; report package delegates import rendering to generate.RenderImportBlock_
_Updated: 2026-02-25 - Sync: Migrated from multichecker.Main() to phasedchecker.Main() phased pipeline architecture; replaced analysistest with phasedchecker/checkertest; added Aggregator/DependencyResult cross-phase coordination pattern; replaced Global Registry Pattern with Cross-Phase State section; removed PackageTracker (no longer exists); added DuplicateRegistry; updated dogfooding to use app.Container[T] and phasedchecker.Config; added analyzer package to component list_
_Updated: 2026-02-26 - Sync: Updated testing section to reflect consolidated table-driven TestIntegration pattern (all e2e tests in single test function using RunWithSuggestedFixes); added dependency-only smoke test pattern_
