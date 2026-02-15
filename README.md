# braider — compile-time DI via go/analysis

![Coverage](https://github.com/miyamo2/braider/blob/main/.assets/test_cov.svg?raw=true)
![Code to Test Ratio](https://github.com/miyamo2/braider/blob/main/.assets/ratio.svg?raw=true)
![Test Execution Time](https://github.com/miyamo2/braider/blob/main/.assets/time.svg?raw=true)

braider is a `go vet` analyzer that resolves dependency injection (DI) bindings and generates constructors and bootstrap wiring using `analysis.SuggestedFix`. It integrates with the standard Go toolchain, produces plain Go code with no runtime container, and is inspired by google/wire.

## Overview

- Compile-time DI validation with actionable diagnostics
- Constructor generation for structs annotated with `annotation.Injectable[T]`
- Provider registration via `annotation.Provide[T](fn)`
- Interface-typed dependencies via `inject.Typed[I]` and `provide.Typed[I]`
- Named dependencies via `inject.Named[N]` and `provide.Named[N]`
- Custom constructors via `inject.WithoutConstructor`
- Field-level DI control via `braider` struct tags (`braider:"name"` / `braider:"-"`)
- Bootstrap wiring generated from a dependency graph in topological order
- Works with `go vet -fix` for one-shot application of suggested fixes

## Installation

Requires go 1.25+.

```bash
go install github.com/miyamo2/braider/cmd/braider@latest
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
- `annotation.App(main)` — marks the entry point where bootstrap code is generated.

### Options

`Injectable[T]` and `Provide[T]` accept option interfaces to customize registration:

| Option | Injectable | Provide | Description |
|--------|-----------|---------|-------------|
| `Default` | `inject.Default` | `provide.Default` | Default registration. Constructor returns `*StructType`. |
| `Typed[I]` | `inject.Typed[I]` | `provide.Typed[I]` | Register as interface type `I` instead of the concrete type. |
| `Named[N]` | `inject.Named[N]` | `provide.Named[N]` | Register with name `N.Name()`. `N` must implement `namer.Namer` and return a string literal. |
| `WithoutConstructor` | `inject.WithoutConstructor` | N/A | Skip constructor generation. You must provide a manual `New<Type>` function. |

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
type MixedService struct {
    annotation.Injectable[interface {
        inject.Typed[Repository]
        inject.Named[ServiceName]
    }]
}
```

**Custom Namer types** must return a hardcoded string literal from `Name()`:

```go
type PrimaryDBName struct{}

func (PrimaryDBName) Name() string { return "primaryDB" }
```

## Example

```go
package main

import (
    "time"

    "github.com/miyamo2/braider/pkg/annotation"
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

var _ = annotation.App(main)

func main() {}
```

**Following `braider -fix ./...`, the generated bootstrap code will look like this**

```go
package main

import (
    "time"

    "github.com/miyamo2/braider/pkg/annotation"
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

var _ = annotation.App(main)

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
- [Struct tag named](examples/struct-tag-named) -- inject a named dependency into a specific field with `braider:"<name>"`
- [Struct tag exclude](examples/struct-tag-exclude) -- exclude a field from DI with `braider:"-"`

## Contributing

Issues and pull requests are welcome. Please keep changes focused and add tests where applicable (`go test ./...`).

## License

MIT. See [LICENSE](LICENSE).
