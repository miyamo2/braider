---
paths:
  - "internal/analyzer/integration_test.go"
  - "internal/analyzer/testdata/**"
---

# Testing

_Read this when: adding/modifying test cases, debugging golden-file diffs, working with `checkertest`, or setting up new testdata directories._

## Out of Scope

Topics under this rule's `paths:` that are handled elsewhere:

- Internal structure of the analyzer pipeline under test → see `architecture.md`
- Rules governing how the generated code that appears in golden files is built → see `code-generation.md`
- Annotation / struct-tag specifications exercised by the test cases → see `annotations.md` / `struct-tags.md`

## phasedchecker/checkertest Framework

All e2e tests run through a single table-driven `TestIntegration` function in `internal/analyzer/integration_test.go`, using `checkertest.RunWithSuggestedFixes` to validate both diagnostics and generated code against `.golden` files.

Each test case builds a `phasedchecker.Config` with the **same Pipeline/DiagnosticPolicy as production**, using isolated registry instances.

Dependency-only smoke tests (no App annotation, no golden files) share the same test runner; `RunWithSuggestedFixes` verifies no unexpected diagnostics.

## Build & Test Commands

```bash
go build ./...                                           # Build all packages
go test ./...                                            # Run all tests
go test -v -run TestIntegration ./internal/analyzer      # Run all e2e tests
go test -v -run TestIntegration/basic ./internal/analyzer # Run a single e2e case
go build -o braider ./cmd/braider                        # Build analyzer binary
braider ./...                                            # Run analyzer
braider -fix ./...                                       # Apply suggested fixes
```

## Testdata Conventions

- Source files contain `// want "message"` directives to assert expected diagnostics
- `.golden` files must match the source file content **after** SuggestedFix application
- For idempotent tests (no `// want` on App annotation): existing bootstrap hash must match computed hash
- Testdata modules use `replace` directives in their `go.mod`:
  ```
  replace github.com/miyamo2/braider => ./../../../../..
  ```
  (count `..` levels carefully; points to the module root from `testdata/<case>/`)
- **Avoid `string`/primitive fields in testdata structs** — they become unresolvable DI dependencies

## Golden File Workflow

1. Create placeholder `.golden` file (e.g., copy source file as-is)
2. Run test → get diff output
3. Paste actual output as the golden

Golden file hashes must match the computed graph hash. For hash mismatches: run test first to get the actual hash from the diff, then update the golden.

## Testdata Directory Structure

All test cases unified under `internal/analyzer/testdata/e2e/` (82 cases), organized by category prefix:

### Core scenarios
`basic`, `crosspackage`, `simpleapp`, `modulewide`, `pkgcollision`, `emptygraph`, `depinuse`, `samefileapp`, `without_constructor`, `multitype`

### Interface resolution
`iface`, `ifacedep`, `crossiface`, `unresiface`

### Typed/Named inject
`typed_inject`, `named_inject`

### Provide variations
`provide_typed`, `provide_named`, `provide_cross_type`

### Variable annotation (`variable_*`)
`variable_basic`, `variable_named`, `variable_typed`, `variable_typed_named`, `variable_cross_package`, `variable_pkg_collision`, `variable_alias_import`, `variable_ident_ext_type`, `variable_mixed`, `variable_idempotent`, `variable_outdated`

### Container mode (`container_*`)
`container_basic`, `container_anonymous`, `container_named`, `container_named_field`, `container_cross_package`, `container_iface_field`, `container_mixed_option`, `container_transitive`, `container_variable`, `container_idempotent`, `container_outdated`, `container_provide_cross_type`

### Struct tag (`struct_tag_*`)
`struct_tag_named`, `struct_tag_exclude`, `struct_tag_mixed`, `struct_tag_all_excluded`, `struct_tag_typed_fields`, `struct_tag_idempotent`, `struct_tag_outdated`

### Idempotent/outdated
`idempotent`, `outdated`

### Error cases
- `error_*` prefix: `error_cases`, `error_duplicate_name`, `error_duplicate_provide_variable`, `error_nonliteral`, `error_provide_typed`, `error_variable_*`, `error_struct_tag_*`, `error_struct_tag_conflict`, `error_container_*`
- Other: `circular`, `ambiguous*`, `unresolvedparam`, `unresparam`, `unresolvedif`, `nonmainapp`, `noapp`, `multipleapp`

### Constructor generation
`constructorgen/` — per-file test cases with `.go`/`.golden` pairs inside a single test directory

### Dependency-only smoke tests
`dep_basic`, `dep_missing_constructor`, `dep_cross_package`, `dep_interface_impl` — no App annotation, no golden files; verify dependency phase runs without unexpected diagnostics

## Test Philosophy

- Test both **positive** cases (should report) and **negative** cases (should not report)
- Use `phasedchecker.Config` identical to production (Pipeline + DiagnosticPolicy), with isolated registry instances
- Each test case is a separate Go package that can be analyzed independently
