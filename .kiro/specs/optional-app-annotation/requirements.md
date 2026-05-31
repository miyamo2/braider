# Requirements Document

## Introduction

The `annotation.App[T](main)` declaration is currently required to identify a project's entry point and trigger bootstrap code generation. For projects with exactly one `main` package, this declaration adds no information — the entry point is unambiguous from package structure alone. This feature makes the `App` annotation optional in that single-main-package case: braider infers the unique main package as the entry point and generates the same bootstrap wiring that an explicit `annotation.App[app.Default](main)` would produce. When the entry point is ambiguous (multiple main packages without explicit `App`), braider emits a diagnostic asking for an explicit declaration instead of guessing.

Source: https://github.com/miyamo2/braider/issues/64

## Boundary Context

- **In scope**:
  - Inferring `annotation.App[app.Default](main)` semantics for the single-main-package case when no explicit `App` annotation exists in the analyzed scope.
  - Emitting an actionable diagnostic when more than one main package is present and no explicit `App` annotation exists.
  - Preserving all existing behavior when at least one explicit `App` annotation is present (explicit always wins).
  - Updating `pkg/annotation/app` documentation to describe optional/inferred behavior.
- **Out of scope**:
  - Inferring container mode (`app.Container[T]`); inference only covers `app.Default` because no user-defined container type can be inferred.
  - Changing the validation rules for explicit `App` annotations (e.g., non-main reference, duplicate-in-file dedup).
  - Discovering main packages outside the analyzer's invocation scope.
  - Modifying the dependency phase (`Injectable` / `Provide` / `Variable` detection) semantics.
- **Adjacent expectations**:
  - The phased pipeline (`dependency` → `app`) already aggregates per-package state via `Aggregator.AfterDependencyPhase`. Main-package discovery must integrate with that aggregation so the `app` phase can decide whether to infer.
  - The `braider -fix` workflow must continue to apply generated bootstrap fixes for both explicit and inferred App cases.

## Requirements

### Requirement 1: Implicit App Inference for a Single Main Package

**Objective:** As a developer with a project containing exactly one main package, I want braider to infer the entry point automatically, so that I do not have to write boilerplate `App[T]` declarations for simple projects.

#### Acceptance Criteria
1. When the analyzed scope contains exactly one main package and no explicit `annotation.App` annotation in any analyzed package, the braider analyzer shall treat that main package as the application entry point and generate bootstrap wiring equivalent to an explicit `annotation.App[app.Default](main)` declared in that package.
2. When inference is active, the braider analyzer shall use `app.Default` semantics (anonymous-struct return value) for the generated bootstrap.
3. When inference is active and zero `Injectable`, `Provide`, or `Variable` registrations have been discovered, the braider analyzer shall generate an empty bootstrap (no struct fields, no local variables) equivalent to what an explicit `App[app.Default]` would produce.
4. While an existing generated bootstrap with a matching hash marker is already present in the inferred main package, the braider analyzer shall skip regeneration (idempotent behavior identical to the explicit-App case).
5. When an existing bootstrap in the inferred main package is stale (hash mismatch), the braider analyzer shall produce a replacement fix using the same update mechanism as the explicit-App case.

### Requirement 2: Explicit App Annotation Always Wins

**Objective:** As a developer who wants explicit control of the bootstrap entry point, I want my `App[T]` declaration to be honored, so that explicit code is never overridden by inference.

#### Acceptance Criteria
1. When at least one explicit `annotation.App` annotation is present in the analyzed scope, the braider analyzer shall use that annotation (or annotations) as the entry point and shall not perform main-package inference.
2. When an explicit `App` annotation is present, the braider analyzer shall preserve all pre-existing App behavior unchanged, including duplicate-in-file warnings, non-main reference validation, and `app.Default` / `app.Container[T]` option semantics.
3. When an explicit `App` annotation is present in some packages but not others, the braider analyzer shall not infer additional App entry points for the remaining main packages.

### Requirement 3: Multiple Main Packages Without Explicit App

**Objective:** As a developer working on a project with multiple main packages, I want a clear diagnostic when the entry point is ambiguous, so that I am never surprised by a guessed bootstrap target.

#### Acceptance Criteria
1. If the analyzed scope contains more than one main package and no explicit `annotation.App` annotation is present in any of them, then the braider analyzer shall emit a diagnostic stating that the entry point is ambiguous and instructing the developer to add an explicit `annotation.App[T](main)` declaration.
2. If the analyzed scope contains more than one main package and no explicit `annotation.App` annotation is present, then the braider analyzer shall not generate bootstrap code for any of those main packages.
3. The ambiguity diagnostic shall be actionable: it shall identify the candidate main packages by import path so the developer can choose where to add the explicit declaration.

### Requirement 4: Definition of "Main Package" for Inference

**Objective:** As a developer, I want a precise definition of what counts as a main package for inference, so that the behavior is predictable across project layouts.

#### Acceptance Criteria
1. The braider analyzer shall classify a package as a main package for inference purposes only when its declared package name is `main` and it contains a top-level `main` function declaration.
2. While performing inference, the braider analyzer shall consider only packages included in the analyzer's invocation scope and shall not load or inspect packages outside that scope.
3. If the analyzed scope contains exactly one package named `main` but that package does not declare a `main` function, then the braider analyzer shall not infer an App annotation and shall not emit a multi-main ambiguity diagnostic.
4. If the analyzed scope contains zero main packages and no explicit `App` annotation, then the braider analyzer shall not generate bootstrap code and shall not emit any App-related diagnostic.

### Requirement 5: Diagnostic and Fix Behavior for Inferred App

**Objective:** As a developer, I want inferred bootstrap generation to integrate cleanly with the existing `braider -fix` workflow, so that the experience is consistent with explicit-annotation usage.

#### Acceptance Criteria
1. When the braider analyzer generates bootstrap code for an inferred App, the analyzer shall report a diagnostic that carries the bootstrap as an applicable suggested fix, positioned at the inferred main package's `main` function declaration.
2. The diagnostic message for the inferred case shall be distinguishable from the explicit-annotation diagnostic, communicating that the entry point was inferred from a single main package.
3. When `braider -fix` is applied to the inferred bootstrap diagnostic, the resulting source file shall compile, and the generated bootstrap shall be structurally equivalent to the bootstrap that an explicit `annotation.App[app.Default](main)` declaration in the same package would produce.
4. When an inferred bootstrap update is required (hash mismatch on existing bootstrap), the braider analyzer shall emit a bootstrap-update diagnostic using the same update fix mechanism as the explicit-annotation case.

### Requirement 6: Documentation of Optional Behavior

**Objective:** As a developer reading the public annotation documentation, I want the inference rule documented, so that I know when I can omit the `App` annotation and what happens if I do.

#### Acceptance Criteria
1. The package documentation for `pkg/annotation/app` shall describe the inference rule: when exactly one main package is in scope and no explicit `App` annotation is declared, braider infers `annotation.App[app.Default](main)` for that package.
2. The package documentation shall state that explicit `annotation.App[T]` always takes precedence over inference.
3. The package documentation shall describe the multi-main ambiguity behavior: when multiple main packages are in scope and no explicit `App` annotation exists, braider emits a diagnostic requesting an explicit declaration.
