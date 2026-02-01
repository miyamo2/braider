# braider

Braider resolves DI bindings and generates wiring via go vet.

## Overview

**braider** is a go vet analyzer that resolves Dependency Injection (DI) bindings and generates wiring code automatically. It leverages the `SuggestedFix` feature of the `go/analysis` package to auto-generate constructors and bootstrap code, integrating seamlessly with the standard Go toolchain.

Unlike runtime DI frameworks, braider provides compile-time safety with zero runtime overhead. All generated code is plain Go with no reflection or containers required.

## Key Features

- **DI Binding Detection**: Static analysis of Go code to detect DI patterns via annotations
- **Constructor Generation**: Automatic generation of constructor functions for structs with injectable dependencies
- **Bootstrap Code Generation**: Generate main function IIFE wiring code with topologically sorted initialization
- **Dependency Graph Resolution**: Analyzes dependency relationships and resolves initialization order
- **Interface Support**: Automatic resolution of interface dependencies to concrete implementations
- **Go Vet Integration**: Works as a standard go vet tool, enabling `go vet -fix` workflow for applying suggested fixes

## How It Works

### Marker Types

braider uses three marker types to define your dependency injection structure:

1. **`annotation.Inject`**: Marks structs as DI targets (become fields in dependency struct)
2. **`annotation.Provide`**: Marks structs as DI providers (local variables only)
3. **`annotation.App(main)`**: Triggers bootstrap code generation in main function

### Usage Flow

1. **Add Annotations**: Mark your structs with `annotation.Inject` or `annotation.Provide`, and add `annotation.App(main)` to trigger bootstrap generation.

```go
package main

import "github.com/miyamo2/braider/pkg/annotation"

// User code with annotations
type MyRepository struct {
    annotation.Inject
}

type MyService struct {
    annotation.Inject
    repo MyRepository
}

var _ = annotation.App(main)

func main() {}
```

2. **Run Analyzer**: The analyzer detects the annotations and generates constructor functions:
   - `NewMyRepository()`
   - `NewMyService(repo MyRepository)`

3. **Generate Bootstrap Code**: The analyzer generates bootstrap code in the `main` function with topologically sorted initialization, ensuring dependencies are created in the correct order.

## Installation

```bash
go install github.com/miyamo2/braider/cmd/braider@latest
```

## Usage

```bash
# Run analysis
go vet -vettool=$(which braider) ./...

# Apply suggested fixes (generates constructors and bootstrap code)
go vet -vettool=$(which braider) -fix ./...
```

## Tech Stack

- **Go Version**: 1.24
- **Core Package**: `golang.org/x/tools/go/analysis`
- **Key Feature**: `analysis.SuggestedFix` for code generation

## Inspiration

This project is inspired by [google/wire](https://github.com/google/wire), which provides compile-time dependency injection for Go.
