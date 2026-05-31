# Implementation Plan

- [x] 1. Foundation: shared registry and detection-result extension

- [x] 1.1 (P) Introduce the EntryPointRegistry as the cross-phase carrier of main-package and explicit-App presence
  - Provide a thread-safe registry that records the set of package import paths that are main packages and the set of package import paths that contain explicit annotation.App declarations
  - Guarantee idempotent registration (re-registering the same path is a no-op) using set semantics under a read-write mutex, matching the locking pattern used by DuplicateRegistry
  - Expose query operations that return a stable, lexicographically sorted snapshot of main-package paths and a boolean indicating whether any explicit App annotation was registered anywhere
  - Cover the registry with unit tests that assert idempotent registration, sorted snapshot output, the HasExplicitApp boolean transition, and safe concurrent registration from multiple goroutines
  - Observable completion: a new entry-point registry file exists with passing unit tests, and the registry can be constructed via a NewEntryPointRegistry constructor that braider's DI annotations can pick up
  - _Requirements: 1.1, 2.1, 2.3, 3.1, 3.3, 4.1, 4.2_
  - _Boundary: registry/EntryPointRegistry_

- [x] 1.2 (P) Extend AppAnnotation with an Inferred marker flag
  - Add a single boolean field on the AppAnnotation struct that marks synthetic, inference-originated values; the existing detector never sets it to true
  - Update doc comments on the struct to describe when the flag is true and what downstream consumers must skip when it is set (annotation validation, option extraction, duplicate-in-file dedup)
  - Observable completion: existing detector tests still pass with the new field defaulting to false; the new field is referenced in at least one godoc comment so its purpose is discoverable from `go doc`
  - _Requirements: 1.1, 5.1, 5.2_
  - _Boundary: detect/AppAnnotation_

- [x] 2. Detection: per-package main-package and explicit-App flags

- [x] 2.1 Extend the per-package DependencyResult payload with entry-point inputs
  - Add fields for the package import path, whether the package is a main package (package name `main` AND a top-level `func main` declaration is present), and whether the package contains at least one explicit annotation.App call
  - Keep these fields as plain value types (string and booleans) so the Aggregator can read them safely across goroutines without retaining AST nodes
  - Update or add a focused unit test on the dependency analyzer that confirms the result struct carries the new fields for a representative input
  - Observable completion: a DependencyResult produced from a sample `package main` test fixture with `func main` reports IsMainPackage=true and a populated PackagePath
  - _Requirements: 4.1, 4.3_
  - _Boundary: analyzer/DependencyResult_

- [x] 2.2 Populate the new DependencyResult fields inside DependencyAnalyzeRunner
  - Inject the existing AppDetector as a new constructor parameter on DependencyAnalyzeRunner so the same App-detection definition is reused across phases
  - During Run, after the existing detection loops, set PackagePath from the pass's package path, set IsMainPackage when the pass's package name is `main` and a top-level `func main` is present, and set HasExplicitApp when the App detector reports one or more annotations
  - Add (or move) a `findMainFunction` helper so it is reachable from both the dependency and app runners without duplication
  - Add a dependency-analyzer unit test that covers four representative cases: non-main package, main with func and no App, main with func and explicit App, package named main without func main
  - Observable completion: running the dependency analyzer on each of those four cases yields the documented flag combinations as visible test assertions
  - _Requirements: 4.1, 4.3, 2.1, 1.1_
  - _Boundary: analyzer/DependencyAnalyzeRunner_

- [x] 3. Aggregation: feed entry-point data into the shared registry

- [x] 3.1 Extend the Aggregator to write entry-point data into EntryPointRegistry
  - Accept the EntryPointRegistry as a new constructor parameter on Aggregator and store it as a field
  - Inside AfterDependencyPhase, in the same loop that registers providers/injectors/variables, register the package path as a main package when IsMainPackage is set and register the package as carrying explicit App when HasExplicitApp is set
  - Update the aggregator unit tests to cover: zero main packages, single main package, multiple main packages, and a mix of main packages with and without explicit App, asserting the resulting registry state for each
  - Observable completion: after AfterDependencyPhase runs over a synthetic graph of DependencyResults, EntryPointRegistry.MainPackagePaths returns the expected sorted slice and HasExplicitApp matches the input scenario
  - _Depends: 1.1, 2.1_
  - _Requirements: 1.1, 2.1, 3.1, 4.1, 4.2_
  - _Boundary: analyzer/Aggregator_

- [x] 4. Diagnostics: new emitter methods for inferred bootstrap and ambiguity

- [x] 4.1 (P) Add diagnostic emitter methods for inferred bootstrap and multi-main ambiguity
  - Add three methods to the DiagnosticEmitter interface and its default implementation: one for a missing inferred bootstrap, one for an outdated inferred bootstrap (carrying a replacement SuggestedFix), and one for the multi-main ambiguity diagnostic
  - Use the existing bootstrap-generation category for the two inferred-bootstrap methods and the existing app-validation category for the ambiguity method; messages must clearly distinguish the inferred case from the explicit-App case and must list each candidate package path deterministically for the ambiguity case
  - Add unit tests that mirror the existing emitter test patterns: assert category, message text contains the inference marker phrasing, candidate paths appear in sorted order, and the SuggestedFix is attached for the bootstrap methods
  - Observable completion: `go test ./internal/report/...` passes with the three new test cases asserting the new messages and categories
  - _Requirements: 3.1, 3.3, 5.1, 5.2, 5.4_
  - _Boundary: report/DiagnosticEmitter_

- [x] 5. Decision: inference and ambiguity branch in AppAnalyzeRunner

- [x] 5.1 Add the inference branch and ambiguity branch to AppAnalyzeRunner.Run
  - Accept EntryPointRegistry as a new constructor parameter on AppAnalyzeRunner
  - When the per-package App detector returns zero annotations: if the registry reports an explicit App anywhere, return without action; otherwise locate the local `func main`, and if absent return without action; if the registry reports exactly one main package proceed by synthesizing an AppAnnotation with Inferred=true, File set to the file containing `func main`, and Pos set to that function's position; if the registry reports more than one main package, emit the ambiguity diagnostic at the local `func main` position with the sorted candidate-paths slice and return
  - Route the synthesized AppAnnotation through the existing default-mode bootstrap pipeline: skip ValidateAppAnnotations, skip DeduplicateAppsByFile (single-element slice makes it a no-op anyway), skip AppOptionExtractor (force default mode), and reuse GenerateBootstrap / CheckBootstrapCurrent / BuildBootstrapFix / BuildBootstrapReplacementFix unchanged
  - Branch diagnostic emission on the Inferred flag so missing bootstrap uses the new inferred-missing emitter, an existing-but-stale bootstrap uses the new inferred-update emitter, and all other generation/graph errors continue to use the existing emitters with the synthesized position
  - Add focused unit tests (or extend existing ones) covering: single main with no App produces an inferred bootstrap, two mains with no App produces the ambiguity diagnostic on each main, single main with explicit App in another package is suppressed (no inference), single main with stale existing inferred bootstrap produces the update diagnostic
  - Observable completion: `go test ./internal/analyzer/...` passes with new and updated tests showing the inference and ambiguity branches firing under the documented conditions
  - _Depends: 1.1, 1.2, 4.1_
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 2.1, 2.3, 3.1, 3.2, 3.3, 4.4, 5.1, 5.2, 5.3, 5.4_
  - _Boundary: analyzer/AppAnalyzeRunner_

- [x] 6. Wiring: integration test harness and dogfooded bootstrap

- [x] 6.1 Update the integration test harness wiring to construct and pass the new components
  - Add construction of EntryPointRegistry to the integration test helper, pass the new AppDetector parameter to the DependencyAnalyzeRunner constructor, pass the EntryPointRegistry to both the Aggregator and AppAnalyzeRunner constructors
  - Adjust any other constructor-side changes required so the harness compiles with the modified constructors from tasks 2.2, 3.1, and 5.1
  - Observable completion: `go build ./...` succeeds and the existing TestIntegration suite still runs (it may temporarily fail on the noapp golden — that golden update is task 7.1)
  - _Depends: 2.2, 3.1, 5.1_
  - _Requirements: 1.1, 2.1, 3.1, 4.1_
  - _Boundary: analyzer/integration_test_

- [x] 6.2 Regenerate the dogfooded bootstrap in cmd/braider/main.go
  - Update the App container struct in cmd/braider/main.go so it (indirectly) exposes the new components via the existing aggregator/analyzer constructors, then run the just-built `braider -fix ./cmd/braider/...` to regenerate the `dependency` IIFE
  - Verify the regenerated IIFE wires EntryPointRegistry into both the Aggregator and the AppAnalyzeRunner constructors and refreshes the `// braider:hash:` marker
  - Observable completion: `go build ./cmd/braider/...` succeeds, the regenerated bootstrap reflects the new constructor signatures, and `braider ./cmd/braider/...` reports no further bootstrap-update diagnostic
  - _Depends: 6.1_
  - _Requirements: 1.1, 2.1, 3.1_
  - _Boundary: cmd/braider/main.go_

- [x] 7. End-to-end test data and TestIntegration coverage

- [x] 7.1 (P) Update the existing noapp e2e fixture to reflect inferred-bootstrap behavior
  - Recognize that the noapp case (one `package main` with `func main` and no annotations) now triggers inference and must produce an empty inferred bootstrap IIFE
  - Update the `main.go.golden` for the noapp fixture so it contains the expected empty inferred bootstrap with the correct hash marker, and update the `// want` comment on the source `main.go` to expect the inferred-missing diagnostic
  - Observable completion: the existing TestIntegration `NoAppAnnotation` subtest passes with the updated golden under the new behavior
  - _Depends: 6.2_
  - _Requirements: 1.3, 5.1, 5.2_
  - _Boundary: testdata/e2e/noapp_

- [x] 7.2 (P) Add an inferred_app_basic e2e fixture exercising single-main inference with a real dependency
  - Create a fixture directory with one `package main` containing only `func main`, a sibling service package providing an Injectable struct, and no annotation.App anywhere
  - Provide a `main.go.golden` that contains the inferred default-mode bootstrap IIFE wiring the service exactly as an explicit App[app.Default] declaration would produce
  - Observable completion: a new TestIntegration entry referencing this fixture passes once added (the test-wiring entry is the responsibility of task 7.6)
  - _Depends: 6.2_
  - _Requirements: 1.1, 1.2, 5.3_
  - _Boundary: testdata/e2e/inferred_app_basic_

- [x] 7.3 (P) Add an inferred_app_idempotent e2e fixture asserting hash-match skip
  - Create a fixture with an inferred bootstrap already present in `main.go` whose hash matches the dependency graph, so the analyzer must emit no diagnostic and produce no fix
  - Provide a `main.go.golden` identical to `main.go`
  - Observable completion: a new TestIntegration entry for this fixture passes once added (wiring in 7.6) and emits no diagnostic
  - _Depends: 6.2_
  - _Requirements: 1.4_
  - _Boundary: testdata/e2e/inferred_app_idempotent_

- [x] 7.4 (P) Add an inferred_app_outdated e2e fixture exercising stale inferred bootstrap update
  - Create a fixture with an existing inferred bootstrap whose hash no longer matches the current dependency graph (e.g., a new Injectable was added)
  - Provide a `main.go.golden` showing the updated inferred bootstrap with refreshed hash and the appropriate `// want` comment expecting the inferred-update diagnostic
  - Observable completion: a new TestIntegration entry for this fixture passes once added (wiring in 7.6), emitting the inferred-update diagnostic and applying the replacement fix
  - _Depends: 6.2_
  - _Requirements: 1.5, 5.4_
  - _Boundary: testdata/e2e/inferred_app_outdated_

- [x] 7.5 (P) Add an ambiguous_entry_point e2e fixture covering the multi-main ambiguity case
  - Create a fixture directory with two `package main` subdirectories (each containing only `func main`) and no annotation.App anywhere in the module
  - Provide `main.go.golden` files for each cmd subdirectory that are identical to their sources (no bootstrap generated), and place `// want` comments asserting the ambiguity diagnostic with both candidate package paths
  - Observable completion: a new TestIntegration entry for this fixture passes once added (wiring in 7.6); each main package emits the ambiguity diagnostic, no bootstrap is generated, and no inferred-missing diagnostic appears
  - _Depends: 6.2_
  - _Requirements: 3.1, 3.2, 3.3_
  - _Boundary: testdata/e2e/ambiguous_entry_point_

- [x] 7.6 Wire the new e2e fixtures into the TestIntegration table
  - Add table entries for InferredAppBasic, InferredAppIdempotent, InferredAppOutdated, and AmbiguousEntryPoint pointing at the directories created in 7.2–7.5
  - Confirm the existing NoAppAnnotation entry remains and now passes against the updated golden from 7.1
  - Confirm the existing MultipleEntryPoints entry remains and continues to pass unchanged (explicit App in each package preserves prior behavior)
  - Observable completion: `go test -v -run TestIntegration ./internal/analyzer` passes with all entries including the new ones, with no regressions in unrelated subtests
  - _Depends: 7.1, 7.2, 7.3, 7.4, 7.5_
  - _Requirements: 1.1, 1.3, 1.4, 1.5, 2.1, 2.2, 3.1, 5.3, 5.4_
  - _Boundary: analyzer/integration_test_

- [x] 8. Documentation

- [x] 8.1 (P) Document optional-App behavior in pkg/annotation/app
  - Update the package-level doc comment (and/or the Default / Option doc comments as appropriate) to describe the inference rule, that explicit App always takes precedence, and that multi-main scope without explicit App produces an ambiguity diagnostic
  - Ensure the documentation is rendered via `go doc ./pkg/annotation/app/...` and remains consistent with existing examples
  - Observable completion: `go doc github.com/miyamo2/braider/pkg/annotation/app` displays the new inference, precedence, and ambiguity statements
  - _Requirements: 6.1, 6.2, 6.3_
  - _Boundary: pkg/annotation/app_
