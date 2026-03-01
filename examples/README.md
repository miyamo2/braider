# Examples

## Running an example

```bash
cd examples/<example>
braider -fix ./...
```

## Example list

- `typed-inject`: `Injectable[inject.Typed[I]]` registers dependencies as interface types.
- `named-inject`: `Injectable[inject.Named[N]]` registers dependencies with literal names from `N.Name()`.
- `without-constructor`: `Injectable[inject.WithoutConstructor]` skips constructor generation and relies on a manual `New<Type>`.
- `mixed-options`: combines `Typed[I]` and `Named[N]` via an embedded option interface.
- `provide-typed`: `Provide[provide.Typed[I]]` registers provider functions as interface types.
- `variable`: `Variable[variable.Default](value)` registers a pre-existing variable as a DI dependency.
- `struct-tag-named`: `braider:"<name>"` struct tag injects a named dependency into a specific field.
- `struct-tag-exclude`: `braider:"-"` struct tag excludes a field from dependency injection.
- `container-basic`: `app.Container[T]` with an anonymous struct as the bootstrap output.
- `container-named`: `app.Container[T]` with a named container type from another package.
