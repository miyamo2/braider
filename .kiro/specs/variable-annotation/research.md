# Research & Design Decisions

## Summary
- **Feature**: `variable-annotation`
- **Discovery Scope**: Extension — adds a new annotation type (`Variable`) to the existing detection → registration → graph → bootstrap pipeline
- **Key Findings**:
  - Expression text extraction via `go/format.Node()` is the most reliable approach for round-tripping AST expressions back to source text
  - Dedicated `VariableCallDetector` and `VariableRegistry` components (Option B) align best with braider's single-responsibility component architecture
  - Cross-package expression qualification requires distinguishing local Idents from already-qualified SelectorExprs at detection time

## Research Log

### Expression Text Extraction Approaches

- **Context**: Variable annotations register an existing expression (e.g., `os.Stdout`, `config.DefaultConfig`) instead of a constructor call. The bootstrap generator needs the original expression text to emit assignment code like `writer := os.Stdout`.

- **Sources Consulted**:
  - `go/format` package documentation (`format.Node()`)
  - `go/printer` package documentation (`printer.Fprint()`)
  - Position-based extraction from `token.FileSet`

- **Findings**:
  - **`go/format.Node()`**: Formats an AST node back to syntactically correct Go source. Handles comments, operator precedence, and import qualifiers. Returns `[]byte`. Works on any `ast.Node` or `ast.Expr`. This is the approach used internally by `gofmt`.
  - **`go/printer.Fprint()`**: Lower-level API that `go/format.Node()` wraps. Offers more configuration options via `printer.Config` but requires manual buffer management. No benefit over `format.Node()` for our use case.
  - **Position-based extraction**: Read file content and extract substring between `expr.Pos()` and `expr.End()`. Fragile — requires access to the original file bytes and fails if the file is modified or if the expression has been rewritten. Not portable across analysis passes.

- **Implications**: Use `go/format.Node()` for expression text extraction. It produces canonical Go source from the AST, which is exactly what we need for bootstrap code generation. The AST is always available during analysis, making this approach robust and deterministic.

### Cross-Package Expression Qualification

- **Context**: When a Variable annotation is declared in package `config` with expression `DefaultConfig` (a package-level variable), the bootstrap code in `main` must emit `config.DefaultConfig`. Conversely, `os.Stdout` (already a SelectorExpr) should be emitted as-is.

- **Sources Consulted**:
  - Existing `CollectImports()` and import alias handling in `internal/generate/imports.go`
  - `go/types.TypeOf()` behavior for expressions
  - Bootstrap generator Phase 2 in `internal/generate/bootstrap.go`

- **Findings**:
  - At detection time, the argument expression's AST node reveals its form:
    - `*ast.Ident` → local reference (e.g., `DefaultConfig` in package `config`) → needs qualification when used from another package
    - `*ast.SelectorExpr` → already qualified (e.g., `os.Stdout`) → use as-is, ensure import is present
    - `*ast.CompositeLit`, `*ast.CallExpr` → out of scope per Non-Goals (complex expressions)
  - Storing both the raw `ExpressionText` (from `format.Node()`) and the declaring package path allows the bootstrap generator to:
    1. For Ident expressions from another package: prepend `<pkgName>.` qualifier
    2. For SelectorExpr expressions: use `ExpressionText` as-is
    3. Add necessary imports via the existing `CollectImports()` infrastructure
  - The `ExpressionPkgs` field (list of package paths referenced by the expression) enables the import collector to add all required imports.

- **Implications**: Store structured expression metadata at detection time: `ExpressionText` (raw formatted text), `ExpressionPkgs` (referenced packages), and `IsQualified` (whether the expression is already package-qualified). The bootstrap generator uses this to emit correct code with proper imports.

### Architecture Pattern: Dedicated vs Overloaded Components

- **Context**: The plan needs to decide between extending existing Provide components or creating new Variable-specific ones.

- **Sources Consulted**:
  - Existing `ProvideCallDetector` and `ProviderRegistry` implementations
  - `DependencyAnalyzeRunner.Run()` phase structure
  - Component instantiation in `cmd/braider/main.go`

- **Findings**:
  - **Option A (Overload Provide)**: Add Variable detection to `ProvideCallDetector`, store Variable entries in `ProviderRegistry`. Pros: fewer new files. Cons: `ProviderInfo` must accommodate both function-based providers (with `ConstructorName`, `Dependencies`) and expression-based variables (with `ExpressionText`, no dependencies). Naming confusion (`ProviderInfo` for non-providers). `isProvideCall()` check logic becomes more complex.
  - **Option B (Dedicated Components)**: New `VariableCallDetector` and `VariableRegistry`. Pros: clean separation of concerns, each component has clear semantics, no risk of breaking existing Provide logic, testable in isolation. Cons: more files, more constructor parameters in analyzers.
  - **Option C (Shared Base)**: Extract common interface (e.g., `DependencySource`), have both Provider and Variable implement it. Pros: unified graph builder interface. Cons: premature abstraction for two concrete types, complicates existing working code.

- **Implications**: Option B is the best fit. braider already follows a component-based architecture with single-responsibility detectors and registries. Adding 2 new files (detector + registry) follows the established pattern and avoids coupling Variable semantics to Provider semantics.

### Expression Type Resolution

- **Context**: The Variable annotation's argument expression must have its type resolved to determine the fully qualified type name for registry and graph.

- **Sources Consulted**:
  - `analysis.Pass.TypesInfo.TypeOf()` API
  - Existing type resolution in `ProvideCallDetector.extractCandidate()`

- **Findings**:
  - `pass.TypesInfo.TypeOf(argExpr)` resolves the type of any expression at analysis time. For `os.Stdout`, this returns `*os.File`. For a package-level `var DefaultConfig Config`, it returns the concrete type.
  - Pointer unwrapping (`*types.Pointer` → `Elem()`) and named type extraction (`*types.Named` → `Obj()`) follow the same pattern used in `ProvideCallDetector`.
  - Edge cases: untyped constants, unexported types, type aliases — these should be handled by the type checker automatically.
  - For `Typed[I]` validation, `types.Implements(concreteType, iface)` or `types.Implements(types.NewPointer(concreteType), iface)` checks interface satisfaction, following the pattern in `OptionExtractor`.

- **Implications**: Type resolution reuses established patterns from `ProvideCallDetector`. No new type resolution infrastructure needed.

## Architecture Pattern Evaluation

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|---------------------|-------|
| A: Overload Provide | Add Variable to ProvideCallDetector & ProviderRegistry | Fewer new files, reuse existing infra | Semantic confusion (ProviderInfo for non-providers), coupling risk, ConstructorName required but irrelevant for Variables | **Rejected** |
| B: Dedicated Components | New VariableCallDetector & VariableRegistry | Clean separation, single-responsibility, testable, follows existing patterns | More files, more constructor parameters | **Selected** |
| C: Shared Base Abstraction | Extract DependencySource interface | Unified graph builder interface | Premature abstraction, complicates working code, over-engineering | **Rejected** |

## Design Decisions

### Decision: Dedicated VariableCallDetector and VariableRegistry

- **Context**: Variable annotations have fundamentally different semantics from Provide annotations (expression vs function, zero dependencies vs function parameters). The architecture must accommodate this cleanly.
- **Alternatives Considered**:
  1. Overload existing `ProvideCallDetector` and `ProviderRegistry` (Option A)
  2. Create dedicated `VariableCallDetector` and `VariableRegistry` (Option B)
  3. Extract shared `DependencySource` abstraction (Option C)
- **Selected Approach**: Option B — dedicated components
- **Rationale**: Follows braider's established component-based architecture. Each component has a clear, single responsibility. Variable semantics (expression assignment, zero dependencies) differ from Provider semantics (constructor invocation, function parameters). Keeping them separate prevents accidental coupling and simplifies testing.
- **Trade-offs**: Two additional source files and slightly more constructor parameters in the analyzers, but this is consistent with how Provide was added alongside Injectable.
- **Follow-up**: Ensure `VariableInfo` implements the `dependencyInfo` interface for graph edge building, even though `GetDependencies()` always returns empty.

### Decision: Node.ExpressionText Field for Bootstrap Differentiation

- **Context**: The bootstrap generator's Phase 2 currently assumes every node has a `ConstructorName` and emits `varName := pkgQualifier.ConstructorName(args)`. Variable nodes have no constructor — they need `varName := expression`.
- **Alternatives Considered**:
  1. Add `ExpressionText` to `graph.Node` and branch in Phase 2
  2. Create separate `VariableNode` type extending `Node`
  3. Use `ConstructorName` field to store expression text with a flag
- **Selected Approach**: Add `ExpressionText string` field to `graph.Node`
- **Rationale**: Minimal change to the `Node` struct. Phase 2 branching logic is simple: `if node.ExpressionText != "" → expression assignment; else → constructor call`. No type assertions needed, no breaking changes to existing interfaces.
- **Trade-offs**: `Node` struct grows by one field (empty for non-Variable nodes). This is acceptable — the struct is not serialized or stored long-term.
- **Follow-up**: Hash computation must include `ExpressionText` when non-empty.

### Decision: Expression Extraction via go/format.Node()

- **Context**: Need to convert the Variable argument AST expression back to source text for bootstrap code generation.
- **Alternatives Considered**:
  1. `go/format.Node()` — AST to canonical source
  2. Position-based file substring extraction
  3. Manual AST reconstruction (hand-written expression printer)
- **Selected Approach**: `go/format.Node()`
- **Rationale**: Standard library, produces canonical Go source, handles all expression types, no file I/O needed, deterministic output.
- **Trade-offs**: None significant. `format.Node()` is the standard approach for this use case.
- **Follow-up**: Verify that `format.Node()` produces importable package qualifiers (e.g., `os.Stdout` not just `Stdout`) — confirmed by checking that SelectorExpr nodes preserve the qualifier in formatting.

### Decision: OptionExtractor Extension with variableOptionsPath

- **Context**: Variable options (`variable.Default`, `variable.Typed[I]`, `variable.Named[N]`) follow the same pattern as inject/provide options but live in a separate package path (`pkg/annotation/variable`).
- **Alternatives Considered**:
  1. Extend existing `OptionExtractor` with `variableOptionsPath` and `ExtractVariableOptions()` method
  2. Create a separate `VariableOptionExtractor`
- **Selected Approach**: Extend existing `OptionExtractor`
- **Rationale**: The option extraction logic (Default/Typed/Named parsing, namer validation, interface search) is identical — only the package path differs. Duplicating this logic would violate DRY and create maintenance burden.
- **Trade-offs**: `isDefaultOptionDirect`, `extractTypedInterfaceDirect`, `extractNamerTypeDirect` must accept the variable options path in addition to inject/provide paths. This is a minor change (adding `|| pkg.Path() == variableOptionsPath` checks).
- **Follow-up**: No `WithoutConstructor` option for Variable (doesn't make sense — Variables never generate constructors).

### Decision: DependencyAnalyzer Phase 2.5 Placement

- **Context**: Variable detection needs a phase in the DependencyAnalyzer's `Run()` method.
- **Alternatives Considered**:
  1. Phase 2.5 (after Provide, before Inject re-detection)
  2. Phase 5 (after package tracking)
  3. Merge with Phase 2 (Provide detection)
- **Selected Approach**: Phase 2.5
- **Rationale**: Variables are "value providers" like Provide — logically grouped after Provide detection. They must be registered before Phase 3 (Inject re-detection) since no ordering dependency exists, but placing them before Phase 4 (package tracking) ensures all registrations complete before the package is marked as scanned.
- **Trade-offs**: Phase numbering becomes non-integer (2.5), but this is documentation-only — the code is sequential.
- **Follow-up**: None.

## Risks & Mitigations

- **Risk 1**: Expression text may contain package-qualified references that break when moved to a different package context.
  - **Mitigation**: Store `ExpressionPkgs` alongside `ExpressionText`. The bootstrap generator uses this to verify imports are present and re-qualify local Idents when generating from a different package.

- **Risk 2**: Hash computation changes could invalidate all existing bootstrap files (mass regeneration).
  - **Mitigation**: `ExpressionText` is only included in hash when non-empty. Existing nodes (Provider, Injector) have empty `ExpressionText`, so their hash contribution is unchanged.

- **Risk 3**: Complex expressions (composite literals, function calls) may be passed as Variable arguments despite being out of scope.
  - **Mitigation**: The `VariableCallDetector` can detect and reject complex expressions by checking the argument AST node type, emitting a diagnostic error for unsupported expression forms.

## References
- [`go/format.Node()`](https://pkg.go.dev/go/format#Node) — Standard library function for formatting AST nodes to Go source
- [`go/types.Implements()`](https://pkg.go.dev/go/types#Implements) — Standard library function for checking interface satisfaction
- [`golang.org/x/tools/go/analysis`](https://pkg.go.dev/golang.org/x/tools/go/analysis) — Go analysis framework
- braider architecture: `CLAUDE.md`, `.kiro/steering/tech.md`
