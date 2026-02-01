# Product Overview

**braider** is a go vet analyzer that resolves Dependency Injection (DI) bindings and generates wiring code automatically. It integrates seamlessly with the standard Go toolchain via `go vet`, providing compile-time DI resolution without runtime overhead.

## Core Capabilities

- **DI Binding Detection**: Static analysis of Go code to identify structs and functions with annotation markers (`Inject`, `Provide`, `ProvideFunc`, `App`)
- **Constructor Generation**: Auto-generate constructor functions for structs with injectable dependencies
- **Bootstrap Code Generation**: Generate main function IIFE wiring code with `App` annotation
- **Dependency Graph Resolution**: Analyzes dependency relationships and generates initialization code in topological order
- **Interface Support**: Automatic resolution of interface dependencies to concrete implementations via `ProvideFunc`
- **Go Vet Integration**: Works as a standard go vet tool, enabling `go vet -fix` workflow for applying suggested fixes

## Target Use Cases

- **Application Bootstrap**: Generate wiring code for service initialization with complex dependency graphs
- **Constructor Generation**: Auto-generate constructor functions for structs with injectable dependencies
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
