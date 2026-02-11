# Examples

## Running an example

```bash
cd examples/<example>
braider -f ./...
```

## Example list

- `typed-inject`: `Injectable[inject.Typed[I]]` registers dependencies as interface types.
- `named-inject`: `Injectable[inject.Named[N]]` registers dependencies with literal names from `N.Name()`.
- `without-constructor`: `Injectable[inject.WithoutConstructor]` skips constructor generation and relies on a manual `New<Type>`.
- `mixed-options`: combines `Typed[I]` and `Named[N]` via an embedded option interface.
- `provide-typed`: `Provide[provide.Typed[I]]` registers provider functions as interface types.
