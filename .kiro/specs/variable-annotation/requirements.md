# Requirements Document

## Introduction

The `variable-annotation` feature adds a new `annotation.Variable[T variable.Option](value)` annotation to braider. While `annotation.Provide[T](fn)` registers a **function** as a dependency provider (calling it during bootstrap to produce a value), `annotation.Variable[T](value)` registers an **existing variable or expression** directly as a dependency without invoking a constructor. This enables users to inject pre-existing values such as `os.Stdout`, configuration objects, or third-party singletons into the dependency graph. The Variable annotation follows the same option patterns (`Default`, `Typed[I]`, `Named[N]`) and integrates with the existing detection, registration, graph-building, and bootstrap code-generation pipeline.

## Requirements

### Requirement 1: Variable Call Detection

**Objective:** As a braider user, I want the analyzer to detect `var _ = annotation.Variable[T](value)` declarations in my source code, so that pre-existing variables can participate in the DI graph.

#### Acceptance Criteria

1. When a package-level `var _ = annotation.Variable[T](value)` declaration is present, the DependencyAnalyzer shall identify it as a Variable annotation call.
2. When the Variable call argument is an expression (e.g., `os.Stdout`, a package-level variable, or a composite literal), the DependencyAnalyzer shall resolve the expression's type and extract the type name.
3. When the Variable call uses a single type parameter (e.g., `annotation.Variable[variable.Default](...)`), the DependencyAnalyzer shall extract the option type parameter for option processing.
4. When the Variable call uses a mixed-option anonymous interface type parameter (e.g., `annotation.Variable[interface{ variable.Typed[io.Writer]; variable.Named[N] }](...)`), the DependencyAnalyzer shall extract all embedded option interfaces.
5. If the Variable call argument is missing or the expression type cannot be resolved, the DependencyAnalyzer shall emit a diagnostic error at the call site.
6. The DependencyAnalyzer shall not treat non-Variable annotation calls (e.g., `annotation.Provide`) as Variable annotations.

### Requirement 2: Variable Option Extraction

**Objective:** As a braider user, I want Variable annotations to support `Default`, `Typed[I]`, and `Named[N]` options, so that I can control how the variable is registered in the DI container.

#### Acceptance Criteria

1. When `variable.Default` is specified, the DependencyAnalyzer shall register the variable under the declared type of the argument expression.
2. When `variable.Typed[I]` is specified, the DependencyAnalyzer shall register the variable under the interface type `I` instead of the argument's declared type.
3. When `variable.Named[N]` is specified and `N` implements `namer.Namer` with a `Name()` method returning a string literal, the DependencyAnalyzer shall register the variable under the name returned by `N.Name()`.
4. If `variable.Named[N]` is specified and `N.Name()` does not return a string literal, the DependencyAnalyzer shall emit a diagnostic error indicating the Namer validation failure.
5. When mixed options are specified via an anonymous interface embedding (e.g., `interface{ variable.Typed[I]; variable.Named[N] }`), the DependencyAnalyzer shall apply all embedded options.

### Requirement 3: Variable Registry

**Objective:** As a braider developer, I want Variable annotations to be stored in a registry accessible to the AppAnalyzer, so that variable dependencies can be resolved during bootstrap code generation.

#### Acceptance Criteria

1. When a Variable annotation is detected and its options are successfully extracted, the DependencyAnalyzer shall register the variable information in a registry that is accessible across packages.
2. The Variable registry shall store: the fully qualified type name, the package path, the local type name, the argument expression source text, the registered type (from `Typed[I]` or the declared type), and the optional name (from `Named[N]`).
3. The Variable registry shall be thread-safe using read-write mutex synchronization, consistent with the existing `ProviderRegistry` and `InjectorRegistry`.
4. If a duplicate `(TypeName, Name)` pair is registered, the Variable registry shall return an error indicating the conflict.
5. When a duplicate named variable dependency is detected, the DependencyAnalyzer shall emit a warning diagnostic with the conflicting package paths.
6. The Variable registry shall provide retrieval methods: `GetAll()` returning all entries sorted deterministically, `Get(typeName)` for unnamed lookup, and `GetByName(typeName, name)` for named lookup.

### Requirement 4: Dependency Graph Integration

**Objective:** As a braider user, I want Variable-provided dependencies to be resolvable in the dependency graph, so that Injectable structs and Provide functions can depend on values registered via Variable.

#### Acceptance Criteria

1. When the AppAnalyzer builds the dependency graph, it shall include nodes for all registered Variable dependencies.
2. The DependencyGraphBuilder shall create Variable nodes with zero dependencies (no constructor parameters), since variables are pre-existing values.
3. When a Variable is registered with `variable.Typed[I]`, the DependencyGraphBuilder shall register the Variable node under the interface type `I` in the InterfaceRegistry, enabling resolution of dependencies that reference `I`.
4. When a Variable is registered with `variable.Named[N]`, the DependencyGraphBuilder shall use the composite key `TypeName#Name` for the graph node, consistent with existing named dependency handling.
5. When an Injectable struct or Provide function depends on a type that is satisfied by a Variable registration, the dependency graph shall resolve it to the Variable node.
6. The Variable node shall have `IsField` set to `false`, since Variable dependencies are expressed as local variable assignments in the bootstrap IIFE, not as struct fields (unlike Provide and Injectable nodes which are struct fields).

### Requirement 5: Bootstrap Code Generation for Variables

**Objective:** As a braider user, I want the generated bootstrap IIFE to include assignments for Variable dependencies, so that the wired application correctly initializes with pre-existing values.

#### Acceptance Criteria

1. When Variable dependencies are present in the sorted dependency graph, the BootstrapGenerator shall emit a local variable assignment using the argument expression (e.g., `writer := os.Stdout`).
2. The BootstrapGenerator shall not emit a constructor call for Variable nodes; it shall directly assign the argument expression as the value.
3. When the Variable's argument expression references a type from another package, the BootstrapGenerator shall include the necessary import in the generated code.
4. When Variable, Provide, and Injectable dependencies coexist in the graph, the BootstrapGenerator shall produce assignments in correct topological order, with Variable nodes appearing before any nodes that depend on them.
5. The generated bootstrap code shall be idempotent: the hash computation shall include Variable entries so that unchanged Variable registrations do not trigger regeneration.
6. When the only change in the dependency graph is a Variable being added or removed, the BootstrapGenerator shall regenerate the bootstrap code with an updated hash.

### Requirement 6: Error Handling and Diagnostics

**Objective:** As a braider user, I want clear error messages when Variable annotations are misconfigured, so that I can quickly correct issues.

#### Acceptance Criteria

1. If a Variable annotation's argument expression type cannot be resolved, the DependencyAnalyzer shall emit a diagnostic error with the position of the Variable call and a message indicating the unresolvable type.
2. If a Variable's `Named[N]` option fails namer validation (non-literal return from `Name()`), the DependencyAnalyzer shall emit a diagnostic error with the position and a clear description of the validation failure.
3. If a Variable is registered with a `Typed[I]` interface that the argument's type does not implement, the DependencyAnalyzer shall emit a diagnostic error indicating the type incompatibility.
4. If a duplicate named Variable dependency is detected, the DependencyAnalyzer shall emit a warning diagnostic including both the existing and conflicting package paths.
5. If a dependency in the graph cannot be resolved and the missing type matches a Variable registration with a name mismatch, the AppAnalyzer shall include a hint in the unresolvable dependency error message suggesting a named lookup.

### Requirement 7: Cross-Package Variable Support

**Objective:** As a braider user, I want to register Variable annotations in any package (not just the main package), so that shared dependencies like loggers or configuration can be declared where they logically belong.

#### Acceptance Criteria

1. When a Variable annotation is declared in a non-main package, the DependencyAnalyzer shall detect and register it in the global Variable registry during the package scan phase.
2. When the AppAnalyzer generates bootstrap code, it shall resolve Variable dependencies from any scanned package, not only the main package.
3. When a cross-package Variable's argument expression references a symbol from its declaring package, the BootstrapGenerator shall emit the fully qualified expression (e.g., `config.DefaultConfig`) with the correct import.

