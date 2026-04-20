# CLAUDE.md

**braider** is a `go/analysis`-based analyzer that resolves DI bindings and generates wiring code automatically using `analysis.SuggestedFix`. Inspired by [google/wire](https://github.com/google/wire); produces plain Go with no runtime container.

Stack: **go 1.25** / `golang.org/x/tools/go/analysis` / `github.com/miyamo2/phasedchecker`.

## Build & Test

```bash
go build ./...                                           # Build all packages
go test ./...                                            # Run all tests
go test -v -run TestIntegration ./internal/analyzer      # All e2e tests
go test -v -run TestIntegration/basic ./internal/analyzer # Single e2e case
go build -o braider ./cmd/braider                        # Build binary
braider ./...                                            # Run analyzer
braider -fix ./...                                       # Apply suggested fixes
```

## Task → Paths Guide

Reverse lookup from intent to the paths you need to touch.

| Intent | Primary paths to edit |
|---|---|
| Add a new annotation type (a 5th kind after Injectable / Provide / Variable / App) | `pkg/annotation/<new>/`, `internal/annotation/` (marker interface), `internal/detect/` (detector), `internal/registry/` (new registry if needed) |
| Add an option type to an existing annotation (e.g. `inject.Foo` / `provide.Bar`) | `pkg/annotation/{inject,provide,variable,app}/`, `internal/detect/` (around `OptionExtractor`) |
| Add or change `braider:"..."` struct tag behavior | `internal/detect/` (tag parsing), plus `internal/generate/` if the tag affects emitted code |
| Change pipeline behavior, phase split, or cross-phase registry sharing | `internal/analyzer/`, `internal/registry/`, `cmd/braider/main.go` |
| Change constructor / bootstrap code generation (AST builders, import handling, hash idempotency) | `internal/generate/` |
| Change diagnostic wording, `SuggestedFix` structure, or severity | `internal/report/` |
| Change Namer / marker resolver / AST validation logic | `internal/detect/` (e.g. `NamerValidator`, `ResolveMarkers`) |
| Add / modify E2E test cases or update goldens | `internal/analyzer/integration_test.go`, `internal/analyzer/testdata/**` |
| Add a new `internal/*` package or change package responsibilities | The affected paths themselves, plus this `Rules Index` and the `paths:` of related rule files |
| Change CLI entry point / command-line behavior | `cmd/braider/main.go` |

New categories: update this table before starting implementation.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/v1.0.0/): `<type>(<scope>): <description>`.

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`.

## Cross-AI Sync Policy

`.claude/rules/*.md` is the **source of truth** for detailed rules. When updating a topic file, propagate the change to the corresponding files so Copilot and Gemini stay in sync:

| Topic | Claude Code | GitHub Copilot | Gemini CLI |
|---|---|---|---|
| architecture | `.claude/rules/architecture.md` | `.github/instructions/architecture.instructions.md` | section in `.gemini/GEMINI.md` |
| annotations | `.claude/rules/annotations.md` | `.github/instructions/annotations.instructions.md` | section in `.gemini/GEMINI.md` |
| struct-tags | `.claude/rules/struct-tags.md` | `.github/instructions/struct-tags.instructions.md` | section in `.gemini/GEMINI.md` |
| code-generation | `.claude/rules/code-generation.md` | `.github/instructions/code-generation.instructions.md` | section in `.gemini/GEMINI.md` |
| testing | `.claude/rules/testing.md` | `.github/instructions/testing.instructions.md` | section in `.gemini/GEMINI.md` |
| internal-layout | `.claude/rules/internal-layout.md` | `.github/instructions/internal-layout.instructions.md` | section in `.gemini/GEMINI.md` |

Commit trailers:
- Copilot commits: `Co-authored-by: Copilot <175728472+Copilot@users.noreply.github.com>`
- Gemini commits: `Co-Authored-By: gemini-cli <218195315+gemini-cli@users.noreply.github.com>`
