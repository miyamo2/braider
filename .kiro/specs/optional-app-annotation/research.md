# Research Log: optional-app-annotation

## Discovery Scope
- **Type**: Extension to existing braider pipeline.
- **Focus**: Locate the seam where App annotation presence drives bootstrap generation; identify the cross-package aggregation surface that can detect "exactly one main package" and "no explicit App in scope."

## Key Findings

### Finding 1: App detection happens twice across phases is acceptable
The `appDetector.DetectAppAnnotations(pass)` call is idempotent and cheap (AST inspector pre-order). Calling it once in the **dependency phase** (per package, to populate a global presence flag) and once in the **app phase** (per package, for processing) is the simplest path to feeding cross-package state into `AppAnalyzeRunner.Run`.

Source: `internal/detect/app.go:81-104`.

### Finding 2: Cross-phase state is already aggregated via `Aggregator.AfterDependencyPhase`
The existing `Aggregator` reads each package's `*DependencyResult` and registers providers/injectors/variables into shared thread-safe registries. The same pattern fits exactly for "main package present" and "explicit App present" booleans: extend `DependencyResult` with two flags and one `PackagePath` string, and have `Aggregator` push them into a new `EntryPointRegistry`.

Source: `internal/analyzer/aggregator.go:36-95`, `internal/analyzer/result.go`.

### Finding 3: `BuildBootstrapFix` only reads `app.File` from the `AppAnnotation`
The default-mode bootstrap pipeline (`AppAnalyzeRunner.Run` lines 184-251) consumes `apps[0]` as: `apps[0].Pos` (diagnostic position), `apps[0].File` (in `BuildBootstrapFix` for import handling). `BuildBootstrapReplacementFix` does not take `*AppAnnotation` at all. **Implication**: an "inferred" synthetic `AppAnnotation{File: <main file>, Pos: <mainFunc.Pos()>, Inferred: true}` is sufficient to drive the existing bootstrap generation flow without touching the fix builder API.

Source: `internal/report/fix.go:131-220`.

### Finding 4: Validation and option extraction must be skipped for inferred App
- `ValidateAppAnnotations` requires `app.MainFunc` (the `*ast.Ident` argument). For inferred App there is no call expression.
- `appOptionExtractor.ExtractAppOption` walks the `CallExpr.Fun.(*ast.IndexExpr)`. For inferred App there is no `CallExpr`.
- **Resolution**: short-circuit both for inferred App and force `app.Default` semantics (no container).

Source: `internal/detect/app.go:208-262`, `internal/detect/app_option_extractor.go`.

### Finding 5: `findMainFunction` already exists in `internal/analyzer/app.go`
Top-level package walk for `FuncDecl` with `Name.Name == "main"`. Reusable for both: detecting "is main package" in the dependency phase, and locating the `*ast.FuncDecl` for the inferred App position in the app phase. Move to a shared helper file or duplicate logic — both acceptable.

### Finding 6: Existing `noapp` integration test exercises the path that will change
`internal/analyzer/testdata/e2e/noapp/` is a `package main` with `func main(){}` and no annotation. Today its golden equals its source (no fix emitted). With inference enabled this case should emit an empty inferred bootstrap. **Implication**: the noapp golden must be updated when implementing this feature — it is no longer a "no-op" case.

### Finding 7: `multipleapp` test exercises the unchanged path
`testdata/e2e/multipleapp/cmd/{1,2}/main.go` each have an explicit `App[app.Default]`. Since explicit App is present in each, no inference runs and the test remains unchanged.

## Architecture Pattern Evaluation

| Option | Approach | Pros | Cons | Decision |
|---|---|---|---|---|
| A | Detect main packages + App presence in dependency phase, aggregate via `Aggregator`, decide in app phase | Reuses existing cross-phase aggregation mechanism; minimal new infrastructure | Adds one new registry and two flags to `DependencyResult` | **Chosen** |
| B | Add a third phase between dependency and app for "entry point resolution" | Cleaner conceptual separation | Adds a phase for a single boolean decision; over-engineered | Rejected |
| C | Query main packages on-demand inside `AppAnalyzeRunner` by scanning `pass.Pkg.Imports()` and sibling packages | Localized | Cannot see packages outside the current package's import graph; would require manual cross-pass coordination | Rejected (incorrect scope) |

## Design Decisions

1. **`EntryPointRegistry` (new)** — thread-safe; stores `mainPackagePaths` set and `explicitAppPackagePaths` set; provides `MainPackagePaths()`, `HasExplicitApp()`, `RegisterMainPackage(pkgPath)`, `RegisterExplicitApp(pkgPath)`. Idempotent registration (set semantics) so duplicate registrations across iterations are safe.

2. **`DependencyResult` extension** — add `IsMainPackage bool`, `HasExplicitApp bool`, `PackagePath string`. These are populated by `DependencyAnalyzeRunner.Run` so the aggregator can fold them into the new registry without an extra pass.

3. **`AppDetector` injection into `DependencyAnalyzeRunner`** — reuse the same detector that runs in app phase; ensures the definition of "App annotation" is identical across phases. Self-hosted DI in `cmd/braider/main.go` regenerates the constructor wiring; the manual e2e test setup needs a single new wiring line.

4. **Synthetic `AppAnnotation` for inference** — add an `Inferred bool` flag on `AppAnnotation`. When true, the app runner skips dedup, validation, and option extraction; forces default mode; emits inferred-specific diagnostics.

5. **Diagnostic surface** — three new methods on `DiagnosticEmitter`:
   - `EmitInferredBootstrapFix(reporter, pos, fix)`: "bootstrap code is missing (entry point inferred from single main package; add `annotation.App` to declare explicitly)".
   - `EmitInferredBootstrapUpdateFix(reporter, pos, fix)`: "bootstrap code is outdated (inferred entry point)".
   - `EmitAmbiguousEntryPoint(reporter, pos, candidatePaths)`: "entry point ambiguous: multiple main packages found ({path1, path2, ...}); add explicit `annotation.App[T](main)` to one package".

6. **Multi-main ambiguity is emitted per main package** — every main package processed in the app phase gets the diagnostic so the user sees it regardless of which main they were investigating. The position is `mainFunc.Pos()`.

## Synthesis Outcomes

- **No new abstractions beyond a single registry** — the EntryPointRegistry is the only new shared type. All other changes are extensions to existing structures (`DependencyResult`, `AppAnnotation`) or new emitter methods.
- **No changes to public `pkg/` API surface** — `app.Default` semantics drive inference; no new option types or runtime behavior change.
- **Generation pipeline is reused, not duplicated** — synthetic `AppAnnotation` lets us route inferred entries through the existing default-mode branch in `AppAnalyzeRunner.Run`.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Existing `noapp` e2e test silently changes behavior | Update test data + golden in the same task; add explicit assertion that the inferred bootstrap is emitted. |
| User-defined main-like helpers (a `main` ident that is not the package's true `func main()`) | `findMainFunction` already filters `FuncDecl.Name.Name == "main"` at top-level; same filter applies to inference. |
| Ambiguity diagnostic noise in monorepos with many `cmd/...` mains | Diagnostic explicitly lists candidate paths so the user can choose; emitted only when no explicit App exists anywhere — explicit declaration in one main suppresses inference everywhere. |
| Race conditions in `EntryPointRegistry` writes | Use `sync.RWMutex` + set semantics, identical pattern to `DuplicateRegistry`. |
| `AppDetector` cost added to dependency phase | Single AST inspector pre-order; same cost as the existing call in app phase. Empirically negligible. |
