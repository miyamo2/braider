# Requirements Document

## Project Description (Input)
bootstrap-with-app-annotation

## Introduction

This feature implements the analyzer logic for detecting `annotation.App(main)` markers and generating application bootstrap code. When a Go package contains `var _ = annotation.App(main)`, braider analyzes all structs marked with `annotation.Inject` across the package, resolves their dependency graph, and generates initialization code that wires up all dependencies in the correct order.

The generated bootstrap code creates a `dependency` variable containing an anonymous struct with all injectable dependencies, initializing them via their constructors in topological order based on the dependency graph.

## Requirements

### Requirement 1: App Annotation Detection
**Objective:** As a developer, I want braider to detect `annotation.App(main)` markers in my code, so that the analyzer knows where to generate bootstrap code.

#### Acceptance Criteria
1. When a package contains `var _ = annotation.App(main)`, the Analyzer shall identify this as a bootstrap target.
2. When a package contains multiple `annotation.App(main)` declarations, the Analyzer shall proceed without error using the first detected App annotation as the bootstrap target. If multiple App annotations appear in the same file, the Analyzer shall emit a warning diagnostic for each duplicate and ignore it.
3. When a package does not contain any `annotation.App` declaration, the Analyzer shall skip bootstrap generation for that package.
4. When `annotation.App` is called with a function other than `main`, the Analyzer shall report an error diagnostic indicating App must reference the main function.

### Requirement 2: Inject-Annotated Struct Discovery
**Objective:** As a developer, I want braider to discover all structs marked with `annotation.Inject`, so that they can be included in the bootstrap wiring.

#### Acceptance Criteria
1. When a struct embeds `annotation.Inject`, the Analyzer shall identify it as an injectable dependency.
2. When a struct does not embed `annotation.Inject`, the Analyzer shall exclude it from the dependency graph.
3. When multiple packages contain `annotation.Inject` structs within the same module, the Analyzer shall discover all of them for bootstrap generation.
4. When an `annotation.Inject` struct is defined but no constructor exists, the Analyzer shall report an error diagnostic indicating the struct requires a constructor.

### Requirement 3: Dependency Graph Construction
**Objective:** As a developer, I want braider to analyze constructor parameters and build a dependency graph, so that dependencies are initialized in the correct order.

#### Acceptance Criteria
1. When parsing a constructor function, the Analyzer shall extract its parameter types as dependencies.
2. When a constructor parameter type matches an `annotation.Inject` struct, the Analyzer shall add a dependency edge from the parameter type to the struct being constructed.
3. When constructor parameters include non-injectable types (primitives, external types), the Analyzer shall report a dependency graph build error indicating the unresolvable dependency type.
4. When a constructor returns multiple values, the Analyzer shall use the first return value as the provided type.
5. When a constructor parameter is an interface type, the Analyzer shall find an `annotation.Inject` struct that implements that interface and add it as a dependency edge.
6. When multiple `annotation.Inject` structs implement the same interface required by a constructor parameter, the Analyzer shall report an error diagnostic listing the ambiguous implementations.
7. When no `annotation.Inject` struct implements a required interface parameter, the Analyzer shall report an error diagnostic indicating the unresolved interface dependency.
8. When resolving interface implementations, the Analyzer shall search across all packages in the module via the global injector/provider registries populated during analysis.

### Requirement 4: Circular Dependency Detection
**Objective:** As a developer, I want braider to detect circular dependencies, so that I can fix invalid dependency graphs before runtime.

#### Acceptance Criteria
1. If a circular dependency is detected in the dependency graph, the Analyzer shall report an error diagnostic listing the cycle path.
2. If the dependency graph contains no cycles, the Analyzer shall proceed with code generation.
3. When reporting a circular dependency, the Analyzer shall include the full cycle path (e.g., "A -> B -> C -> A").

### Requirement 5: Topological Sort for Initialization Order
**Objective:** As a developer, I want dependencies to be initialized in topological order, so that each dependency is available when needed by its dependents.

#### Acceptance Criteria
1. When generating initialization code, the Analyzer shall order constructor calls such that a dependency is initialized before any struct that depends on it.
2. When multiple valid topological orderings exist, the Analyzer shall produce a deterministic order (alphabetical by type name for ties).
3. When the dependency graph is empty (no injectable structs), the Analyzer shall generate an empty bootstrap block.

### Requirement 6: Bootstrap Code Generation
**Objective:** As a developer, I want braider to generate the bootstrap code as a SuggestedFix, so that I can apply it via `go vet -fix`.

#### Acceptance Criteria
1. When an `annotation.App(main)` is detected and the dependency graph is valid, the Analyzer shall generate a SuggestedFix containing the bootstrap code.
2. The generated bootstrap code shall define a package-level variable named `dependency` that initializes all injectable structs.
3. When the `dependency` variable is not referenced in the main function body, the generated bootstrap code shall include `_ = dependency` inside the main function body to ensure the dependency variable is referenced. When the `dependency` variable is already referenced in the main function body, the Analyzer shall not add `_ = dependency`.
4. When applying the SuggestedFix, the generated code shall be valid Go code that compiles without errors.
5. When the bootstrap code already exists and is up-to-date, the Analyzer shall not report a diagnostic (idempotent behavior).
6. If the bootstrap code exists but is outdated (dependency graph changed), the Analyzer shall report a diagnostic with a SuggestedFix to update the bootstrap code.

### Requirement 7: Generated Code Structure
**Objective:** As a developer, I want the generated bootstrap code to follow a predictable structure, so that it is readable and maintainable.

#### Acceptance Criteria
1. The generated `dependency` variable shall be an immediately-invoked function returning an anonymous struct.
2. The anonymous struct shall contain fields for each injectable dependency, named in camelCase derived from the type name.
3. The generated constructor calls shall use the field names as variable names for intermediate values.
4. When a dependency is from an external package, the Analyzer shall include the appropriate import statement in the generated code.
5. The generated code shall be formatted according to `gofmt` standards.

### Requirement 8: Diagnostic Messages
**Objective:** As a developer, I want clear diagnostic messages, so that I can understand and fix issues in my DI configuration.

#### Acceptance Criteria
1. When reporting an error, the Analyzer shall include the source position (file, line, column) of the problematic code.
2. When reporting an error, the Analyzer shall provide a message describing what is wrong and how to fix it.
3. When suggesting a fix, the Analyzer shall provide a clear description of what the fix will do.

### Requirement 9: Integration with Existing Constructor Generation
**Objective:** As a developer, I want the bootstrap feature to work alongside the existing constructor generation feature, so that I can use both features together.

#### Acceptance Criteria
1. When both App annotation and Inject annotations are present, the Analyzer shall generate both constructor code and bootstrap code.
2. The Analyzer shall ensure constructors are generated before bootstrap code references them.
3. When the Analyzer runs multiple times (incremental analysis), it shall produce consistent results.
