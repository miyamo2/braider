# Research & Design Decisions: bootstrap-with-app-annotation

---
**Purpose**: Capture discovery findings, architectural investigations, and rationale that inform the technical design.
---

## Summary
- **Feature**: `bootstrap-with-app-annotation`
- **Discovery Scope**: Extension (builds on existing constructor generation feature)
- **Key Findings**:
  - Topological sort via Kahn's algorithm is preferred for deterministic, cycle-detecting ordering
  - go/analysis Facts mechanism enables cross-package dependency discovery
  - Bootstrap code follows IIFE pattern with anonymous struct for clean dependency access

## Research Log

### Topological Sort Algorithms for Dependency Graphs

- **Context**: Need to order constructor calls such that dependencies are initialized before dependents
- **Sources Consulted**:
  - [Sorting a Dependency Graph in Go](https://kendru.github.io/go/2021/10/26/sorting-a-dependency-graph-in-go/)
  - [gammazero/toposort package](https://pkg.go.dev/github.com/gammazero/toposort)
  - [gonum/graph/topo package](https://pkg.go.dev/gonum.org/v1/gonum/graph/topo)
  - [stevenle/topsort package](https://pkg.go.dev/github.com/stevenle/topsort)
- **Findings**:
  - Kahn's algorithm: O(V+E) time complexity, naturally detects cycles
  - DFS-based approach: Requires additional cycle detection pass
  - gonum provides comprehensive graph utilities but adds significant dependency
  - Custom implementation avoids external dependencies for minimal analyzer tool
- **Implications**: Implement Kahn's algorithm internally; alphabetical secondary sort for determinism

### Cross-Package Dependency Discovery in go/analysis

- **Context**: Bootstrap generation requires discovering Inject-annotated structs across multiple packages
- **Sources Consulted**:
  - [go/analysis package documentation](https://pkg.go.dev/golang.org/x/tools/go/analysis)
  - [Writing multi-package analysis tools](https://eli.thegreenplace.net/2020/writing-multi-package-analysis-tools-for-go/)
  - [Analysis Framework DeepWiki](https://deepwiki.com/golang/tools/4-analysis-framework)
- **Findings**:
  - Facts are serializable findings shared between packages via gob encoding
  - `FactTypes` field establishes vertical dependency (same analyzer, different packages)
  - Facts can be exported for package-level or object-level entities
  - Analyzer can import facts from any import dependency of current package
- **Implications**: Export `InjectableFact` for each Inject-annotated struct; import facts from dependencies during bootstrap analysis

### Bootstrap Code Pattern Selection

- **Context**: Determine the structure of generated bootstrap code
- **Sources Consulted**:
  - google/wire patterns and generated code examples
  - annotation.App documentation in pkg/annotation/annotation.go
- **Findings**:
  - IIFE pattern provides clean scope for intermediate variables
  - Anonymous struct allows grouping all dependencies under single `dependency` variable
  - Deterministic field ordering (alphabetical by type name) ensures reproducible output
  - `_ = dependency` in main ensures the variable is referenced (prevents unused variable error)
- **Implications**: Generate IIFE returning anonymous struct; use camelCase field names derived from type names

## Architecture Pattern Evaluation

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|---------------------|-------|
| Extend existing pipeline | Add Phase 3 for bootstrap after constructor generation | Minimal architecture change, reuses components | Increases complexity of run function | Selected approach |
| Separate analyzer | Create new analyzer for bootstrap | Clean separation | Requires coordination between analyzers, duplicate detection | Rejected |
| Plugin architecture | Composable analysis passes | Maximum flexibility | Over-engineering for current scope | Future consideration |

## Design Decisions

### Decision: Kahn's Algorithm for Topological Sort

- **Context**: Need cycle detection and deterministic ordering for dependency initialization
- **Alternatives Considered**:
  1. DFS-based topological sort - Requires separate cycle detection
  2. External library (gonum, topsort) - Adds dependency to analyzer
  3. Kahn's algorithm - In-degree based, detects cycles naturally
- **Selected Approach**: Custom Kahn's algorithm implementation
- **Rationale**:
  - O(V+E) complexity matches DFS performance
  - Naturally detects cycles (non-zero in-degree nodes after sort indicate cycle)
  - No external dependencies keeps analyzer lightweight
  - Well-documented algorithm with clear implementation path
- **Trade-offs**:
  - Benefit: Zero external dependencies
  - Compromise: Must implement and maintain algorithm code
- **Follow-up**: Verify cycle detection produces meaningful error paths

### Decision: Facts for Cross-Package Discovery

- **Context**: Bootstrap generation requires knowledge of Inject structs in imported packages
- **Alternatives Considered**:
  1. Re-analyze all packages during bootstrap pass
  2. Use analysis.Fact mechanism for exported information
  3. Require all Inject structs in same package as main
- **Selected Approach**: Export `InjectableFact` from each package containing Inject structs
- **Rationale**:
  - Leverages go/analysis built-in mechanism for cross-package data
  - Efficient: Facts computed once, reused across dependent packages
  - Follows established patterns in go/analysis ecosystem
- **Trade-offs**:
  - Benefit: Correct handling of multi-package applications
  - Compromise: More complex implementation than single-package analysis
- **Follow-up**: Design fact schema to include constructor signature and dependencies

### Decision: IIFE Bootstrap Pattern

- **Context**: Bootstrap code structure for initializing dependencies
- **Alternatives Considered**:
  1. Global variable assignments - Simple but pollutes namespace
  2. Init function pattern - Implicit ordering concerns
  3. IIFE with anonymous struct - Clean scoping, explicit variable
- **Selected Approach**: IIFE returning anonymous struct assigned to `dependency` variable
- **Rationale**:
  - Matches pattern documented in annotation.App examples
  - Intermediate variables scoped within IIFE
  - Single exported `dependency` variable provides clean access
  - Anonymous struct naturally groups related dependencies
- **Trade-offs**:
  - Benefit: Clean, readable generated code
  - Compromise: Slightly more complex generation logic
- **Follow-up**: None

### Decision: Package-Scoped Bootstrap Generation

- **Context**: Where to perform bootstrap code generation
- **Alternatives Considered**:
  1. Generate in main package only (where App annotation exists)
  2. Generate partial wiring in each package
- **Selected Approach**: Generate complete bootstrap code in main package only
- **Rationale**:
  - App annotation marks the bootstrap target location
  - Single point of code generation simplifies implementation
  - All constructor information available via Facts from imported packages
- **Trade-offs**:
  - Benefit: Simpler code generation logic
  - Compromise: All wiring visible in main package (may be large)
- **Follow-up**: Consider partial generation for very large applications in future

## Risks & Mitigations

- **Risk 1: Circular dependencies across packages** - Kahn's algorithm detects cycles; report full cycle path in diagnostic message
- **Risk 2: Constructor signature mismatch** - Verify constructor exists and parameters match injectable types during graph construction
- **Risk 3: Import path resolution** - Use qualified type names in dependency graph; handle aliased imports via TypesInfo
- **Risk 4: Large dependency graphs** - Kahn's algorithm is O(V+E); unlikely to be performance issue for typical applications
- **Risk 5: Non-injectable parameters** - Skip primitives and external types; only wire Inject-annotated dependencies

## References

- [go/analysis package](https://pkg.go.dev/golang.org/x/tools/go/analysis) - Core analyzer framework
- [Writing multi-package analysis tools](https://eli.thegreenplace.net/2020/writing-multi-package-analysis-tools-for-go/) - Multi-package patterns
- [gammazero/toposort](https://pkg.go.dev/github.com/gammazero/toposort) - Reference Kahn's algorithm implementation
- [Sorting a Dependency Graph in Go](https://kendru.github.io/go/2021/10/26/sorting-a-dependency-graph-in-go/) - Practical guide
- [google/wire](https://github.com/google/wire) - Inspiration for DI patterns