# Product Overview

**braider** is a go vet analyzer that resolves Dependency Injection (DI) bindings and generates wiring code automatically. It integrates seamlessly with the standard Go toolchain via `go vet`, providing compile-time DI resolution without runtime overhead.

## Core Capabilities

- **DI Binding Detection**: Static analysis of Go code to identify structs, functions, and variables with annotation markers (`Injectable[T]`, `Provide[T](fn)`, `Variable[T](value)`, `App[T](main)`)
- **Constructor Generation**: Auto-generate constructor functions for all structs embedding `Injectable[T]`, including zero-dependency structs
- **Bootstrap Code Generation**: Generate main function IIFE wiring code with `App[T]` annotation; supports both default (anonymous struct) and user-defined container output modes via App option type parameter
- **Dependency Graph Resolution**: Analyzes dependency relationships and generates initialization code in topological order
- **Variable Registration**: Register existing variables/values as dependencies via `Variable[T](value)` (e.g., `annotation.Variable[variable.Default](os.Stdout)`)
- **Interface Support**: Automatic resolution of interface dependencies to concrete implementations via `Provide[provide.Typed[I]](fn)`, `Injectable[inject.Typed[I]]`, and `Variable[variable.Typed[I]](value)`
- **Struct Tag Control**: Field-level DI customization via `braider` struct tags (`braider:"name"` for named resolution, `braider:"-"` to exclude fields from DI)
- **Container Definition**: User-defined container structs via `App[app.Container[T]]` option, enabling typed access to resolved dependencies through a custom struct type rather than an anonymous struct
- **Go Vet Integration**: Works as a standard go vet tool, enabling `go vet -fix` workflow for applying suggested fixes

## Target Use Cases

- **Application Bootstrap**: Generate wiring code for service initialization with complex dependency graphs
- **Constructor Generation**: Auto-generate constructor functions for all `Injectable[T]` structs, regardless of field count
- **Refactoring Support**: Detect missing or broken DI bindings during code changes

## Value Proposition

Unlike runtime DI frameworks, braider provides:
- **Compile-time Safety**: Errors detected during analysis, not at runtime
- **Zero Runtime Overhead**: Generated code is plain Go, no reflection or containers
- **Toolchain Integration**: Works with existing `go vet` workflow, no custom build steps
- **Inspired by Wire**: Similar philosophy to google/wire, but leveraging go/analysis for broader tooling integration

---
_Focus on patterns and purpose, not exhaustive feature lists_

_Updated: 2026-02-02 - Added ProvideFunc annotation support for function-based dependency providers_
_Updated: 2026-02-11 - Sync: Corrected annotation names to match current API (Injectable[T], Provide[T](fn)); removed obsolete ProvideFunc references_
_Updated: 2026-02-12 - Sync: Added Variable[T](value) annotation as a core capability for registering existing variables as dependencies_
_Updated: 2026-02-14 - Sync: Constructor generation now applies to all Injectable[T] structs, including zero-dependency structs (no HasInjectableFields guard)_
_Updated: 2026-02-15 - Sync: Added struct tag support for field-level DI control; Provide annotations now included as bootstrap struct fields_
_Updated: 2026-02-16 - Sync: App annotation now uses generic form App[T](main) with app option type parameter; added container definition support via app.Container[T] option_
