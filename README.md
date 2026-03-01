# braider — compile-time DI via go/analysis

[![Go Reference](https://pkg.go.dev/badge/github.com/miyamo2/braider.svg)](https://pkg.go.dev/github.com/miyamo2/braider)
[![GitHub go.mod Go version (subdirectory of monorepo)](https://img.shields.io/github/go-mod/go-version/miyamo2/braider?logo=go)](https://img.shields.io/github/go-mod/go-version/miyamo2/braider?logo=go)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/miyamo2/braider)](https://img.shields.io/github/v/release/miyamo2/braider)
![Coverage](https://github.com/miyamo2/braider/blob/main/.assets/test_cov.svg?raw=true)
![Code to Test Ratio](https://github.com/miyamo2/braider/blob/main/.assets/ratio.svg?raw=true)
![Test Execution Time](https://github.com/miyamo2/braider/blob/main/.assets/time.svg?raw=true)
[![Go Report Card](https://goreportcard.com/badge/github.com/miyamo2/braider)](https://goreportcard.com/report/github.com/miyamo2/braider)
[![GitHub License](https://img.shields.io/github/license/miyamo2/braider?&color=blue)](https://img.shields.io/github/license/miyamo2/braider?&color=blue)

braider is a `go/analysis` analyzer that resolves dependency injection (DI) bindings and generates constructors and bootstrap wiring using `analysis.SuggestedFix`. It integrates with the standard Go toolchain, produces plain Go code with no runtime container, and is inspired by google/wire.

## Overview

- Compile-time DI validation with actionable diagnostics
- Constructor generation for structs annotated with `annotation.Injectable[T]`
- Provider registration via `annotation.Provide[T](fn)`
- Variable registration via `annotation.Variable[T](value)` for pre-existing variables
- Interface-typed dependencies via `inject.Typed[I]`, `provide.Typed[I]`, and `variable.Typed[I]`
- Named dependencies via `inject.Named[N]`, `provide.Named[N]`, and `variable.Named[N]`
- Custom constructors via `inject.WithoutConstructor`
- Field-level DI control via `braider` struct tags (`braider:"name"` / `braider:"-"`)
- App options: `app.Default` (anonymous struct) and `app.Container[T]` (user-defined container)
- Bootstrap wiring generated from a dependency graph in topological order
- Works with `braider -fix` for one-shot application of suggested fixes

## Installation

Requires go 1.25+.

### CLI

```bash
go install github.com/miyamo2/braider/cmd/braider@latest
```

Or, add it as a [tool dependency](https://go.dev/doc/modules/managing-dependencies#tools)

```bash
go get -tool github.com/miyamo2/braider/cmd/braider@latest
```

Or, install via [Homebrew](https://brew.sh/)

```bash
brew install miyamo2/tap/braider
```

### Module

Annotation types and option packages used in your code are provided by this module.

```bash
go get github.com/miyamo2/braider
```

## Usage

1. Add annotations in your code.
2. (Optional) Run the analyzer to check for configuration issues

```bash
braider ./...
```

3. Apply generated constructors and bootstrap wiring

```bash
braider -fix ./...
```

### Annotations

- `annotation.Injectable[inject.Default]` — marks structs that need constructor generation and will be exposed from the bootstrap dependency struct.
- `annotation.Provide[provide.Default](fn)` — registers provider functions exposed as fields in the bootstrap dependency struct.
- `annotation.Variable[variable.Default](value)` — registers a pre-existing variable or package-qualified identifier (e.g., `os.Stdout`) as a DI dependency without generating a constructor.
- `annotation.App[app.Default](main)` — marks the entry point where bootstrap code is generated. Use `app.Default` for an anonymous struct or `app.Container[T]` for a user-defined container type.

### Options

`Injectable[T]`, `Provide[T]`, and `Variable[T]` accept option interfaces to customize registration:

| Option | Injectable | Provide | Variable | Description |
|--------|-----------|---------|----------|-------------|
| `Default` | `inject.Default` | `provide.Default` | `variable.Default` | Default registration. |
| `Typed[I]` | `inject.Typed[I]` | `provide.Typed[I]` | `variable.Typed[I]` | Register as interface type `I` instead of the concrete type. |
| `Named[N]` | `inject.Named[N]` | `provide.Named[N]` | `variable.Named[N]` | Register with name `N.Name()`. `N` must implement `namer.Namer` and return a string literal. |
| `WithoutConstructor` | `inject.WithoutConstructor` | N/A | N/A | Skip constructor generation. You must provide a manual `New<Type>` function. |

`App[T]` accepts option interfaces to customize bootstrap output:

| Option | Description |
|--------|-------------|
| `app.Default` | Generate an anonymous struct with all dependencies as fields (default behavior). |
| `app.Container[T]` | Generate a bootstrap function that returns the user-defined container type `T`. Container fields are matched by type; use `braider:"name"` tags for named dependencies and `braider:"-"` to exclude fields. |

### Struct Tags

`Injectable[T]` struct fields can use `braider` struct tags for field-level DI control:

| Tag | Description |
|-----|-------------|
| `braider:"<name>"` | Resolve this field using the named dependency matching `<name>`. |
| `braider:"-"` | Exclude this field from dependency injection entirely. |

Fields without a `braider` tag are resolved by type as usual.

**Named dependency injection** — use `braider:"<name>"` to wire a specific named provider/injector/variable to a field:

```go
type PrimaryRepoName struct{}

func (PrimaryRepoName) Name() string { return "primaryRepo" }

var _ = annotation.Provide[provide.Named[PrimaryRepoName]](NewUserRepository)

func NewUserRepository() *UserRepository { return &UserRepository{} }

type AppService struct {
    annotation.Injectable[inject.Default]
    repo *UserRepository `braider:"primaryRepo"`
}
```

**Field exclusion** — use `braider:"-"` to keep a field out of DI:

```go
type AppService struct {
    annotation.Injectable[inject.Default]
    logger   Logger
    debugger Debugger `braider:"-"`
}
```

The generated constructor will only accept `logger` as a parameter; `debugger` is ignored.

Struct tags can be combined freely — some fields tagged with names, some excluded, and others resolved by type:

```go
type AppService struct {
    annotation.Injectable[inject.Default]
    repo     *UserRepository `braider:"primaryRepo"`
    logger   Logger
    debugger Debugger        `braider:"-"`
}
```

**Mixed options** are supported by embedding multiple option interfaces in a single anonymous interface:

```go
type MixedRepository struct {
    annotation.Injectable[interface {
        inject.Typed[Repository]
        inject.Named[RepositoryName]
    }]
}
```

**Custom Namer types** must return a hardcoded string literal from `Name()`:

```go
type PrimaryDBName struct{}

func (PrimaryDBName) Name() string { return "primaryDB" }
```

### Variable

`annotation.Variable[T](value)` registers a pre-existing variable or package-qualified identifier as a DI dependency. Unlike `Injectable` and `Provide`, no constructor is generated or invoked — the value is assigned directly in the bootstrap code.

Supported argument expressions: identifiers (`myVar`) and package-qualified selectors (`os.Stdout`). Literals, function calls, and composite literals are not supported.

```go
import (
    "os"

    "github.com/miyamo2/braider/pkg/annotation"
    "github.com/miyamo2/braider/pkg/annotation/variable"
)

// Register os.Stdout as a *os.File dependency.
var _ = annotation.Variable[variable.Default](os.Stdout)
```

In the generated bootstrap code, this becomes a direct assignment: `stdout := os.Stdout`.

### App Options

`annotation.App[T](main)` is generic — the type parameter `T` controls the shape of the bootstrap output.

**`app.Default`** — generates an anonymous struct with all dependencies as fields:

```go
var _ = annotation.App[app.Default](main)
```

**`app.Container[T]`** — generates a bootstrap function that returns a user-defined container type `T`:

```go
var _ = annotation.App[app.Container[struct {
    Svc *Service
}]](main)
```

Container fields are matched against registered dependencies by type. Use `braider:"name"` tags to match named dependencies. Note that `braider:"-"` is not permitted on container fields (it is a validation error):

```go
var _ = annotation.App[app.Container[struct {
    Primary *Database `braider:"primaryDB"`
    Replica *Database `braider:"replicaDB"`
}]](main)
```

The container type can also be a named struct from another package:

```go
var _ = annotation.App[app.Container[container.AppContainer]](main)
```

## Example

```go
package main

import (
    "time"

    "github.com/miyamo2/braider/pkg/annotation"
    "github.com/miyamo2/braider/pkg/annotation/app"
    "github.com/miyamo2/braider/pkg/annotation/inject"
    "github.com/miyamo2/braider/pkg/annotation/provide"
)

type Clock interface {
    Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

var _ = annotation.Provide[provide.Typed[Clock]](NewClock)

func NewClock() *realClock { return &realClock{} }

type Service struct {
    annotation.Injectable[inject.Default]
    Clock Clock
}

var _ = annotation.App[app.Default](main)

func main() {}
```

**Following `braider -fix ./...`, the generated bootstrap code will look like this**

```go
package main

import (
    "time"

    "github.com/miyamo2/braider/pkg/annotation"
    "github.com/miyamo2/braider/pkg/annotation/app"
    "github.com/miyamo2/braider/pkg/annotation/inject"
    "github.com/miyamo2/braider/pkg/annotation/provide"
)

type Clock interface {
    Now() time.Time
}

type realClock struct{}

func (r realClock) Now() time.Time {
	return time.Now()
}

var _ = annotation.Provide[provide.Typed[Clock]](NewClock)

func NewClock() *realClock { return &realClock{} }

type Service struct {
    annotation.Injectable[inject.Default]
    Clock Clock
}

// NewService is a constructor for Service.
//
// Generated by braider. DO NOT EDIT.
func NewService(c Clock) Service {
    return Service{Clock: c}
}

var _ = annotation.App[app.Default](main)

func main() {
    _ = dependencies
}

// braider:hash:blurblurblur
var dependencies = func() struct {
    Service Service
} {
    clock := NewClock()
    service := NewService(clock)
    return struct {
        Service Service
    }{
        Service: service,
    }
}
```

### Examples

- [Typed inject](examples/typed-inject) -- register a struct as an interface type with `inject.Typed[I]`
- [Named inject](examples/named-inject) -- register multiple instances with different names via `inject.Named[N]`
- [Without constructor](examples/without-constructor) -- skip constructor generation with `inject.WithoutConstructor`
- [Mixed options](examples/mixed-options) -- combine `Typed[I]` and `Named[N]` in a single annotation
- [Provide typed](examples/provide-typed) -- register a provider function as an interface type with `provide.Typed[I]`
- [Variable](examples/variable) -- register a pre-existing variable as a DI dependency with `annotation.Variable[T]`
- [Struct tag named](examples/struct-tag-named) -- inject a named dependency into a specific field with `braider:"<name>"`
- [Struct tag exclude](examples/struct-tag-exclude) -- exclude a field from DI with `braider:"-"`
- [Container (anonymous)](examples/container-basic) -- use `app.Container[T]` with an anonymous struct as the bootstrap output
- [Container (named type)](examples/container-named) -- use `app.Container[T]` with a named container type from another package

## Contributing

Issues and pull requests are welcome. Please keep changes focused and add tests where applicable (`go test ./...`).

## License

MIT. See [LICENSE](LICENSE).
