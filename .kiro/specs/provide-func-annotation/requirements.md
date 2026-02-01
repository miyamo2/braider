# Requirements Document

## Project Description (Input)
provide-func-annotation

## Introduction

This feature adds support for annotating constructor or factory functions with `annotation.ProvideFunc` instead of embedding `annotation.Provide` in structs. By marking provider functions directly, developers can register factory methods for standard libraries or their own packages as dependency providers without modifying the types themselves.

ProvideFunc-annotated functions participate in dependency resolution, interface mapping, and bootstrap generation. The analyzer treats these functions as providers whose parameters are dependencies and whose return values are the provided types.

## Requirements

### Requirement 1: ProvideFunc Annotation Detection
**Objective:** As a developer, I want braider to detect `annotation.ProvideFunc` markers, so that I can register constructor or factory functions as dependency providers.

#### Acceptance Criteria
1. When a package contains `var _ = annotation.ProvideFunc(fn)`, the Analyzer shall detect the ProvideFunc annotation and register `fn` as a provider function.
2. When multiple ProvideFunc annotations are present in a package, the Analyzer shall register each referenced function once.
3. When a package contains no ProvideFunc annotations, the Analyzer shall not register provider functions from that package.

### Requirement 2: Provider Function Validation
**Objective:** As a developer, I want braider to validate provider functions, so that invalid ProvideFunc usage is caught early.

#### Acceptance Criteria
1. If the argument to ProvideFunc does not resolve to a function or method expression, the Analyzer shall report an error diagnostic indicating ProvideFunc requires a function reference.
2. If the referenced function has zero return values, the Analyzer shall report an error diagnostic indicating a provider function must return a value.
3. When the referenced function returns multiple values, the Analyzer shall treat the first return value as the provided dependency type.
4. The Analyzer shall accept provider functions that return either concrete types or interface types.

### Requirement 3: Dependency Extraction for Provider Functions
**Objective:** As a developer, I want provider functions to participate in dependency resolution, so that their parameters are wired correctly.

#### Acceptance Criteria
1. When a provider function declares parameters, the Analyzer shall treat each parameter type as a dependency required to invoke the function.
2. When a provider function has no parameters, the Analyzer shall treat it as a leaf provider with no dependencies.
3. When a parameter type matches a registered injector or provider, the Analyzer shall include it as a dependency edge.
4. If a parameter type cannot be resolved to any registered provider or injector, the Analyzer shall report an error diagnostic indicating an unresolved dependency.

### Requirement 4: Interface Resolution Using ProvideFunc
**Objective:** As a developer, I want ProvideFunc providers to support interface dependencies, so that interfaces can be wired to concrete implementations.

#### Acceptance Criteria
1. When a provider function returns a concrete type that implements an interface, the Analyzer shall register that type as an implementation for the interface.
2. When an interface-typed dependency is required by a constructor or provider function, the Analyzer shall resolve it to a single implementing provider or injector.
3. If multiple implementations are registered for the same interface, the Analyzer shall report an ambiguity diagnostic listing the implementations.
4. If no implementation is registered for a required interface, the Analyzer shall report an unresolved interface diagnostic.

### Requirement 5: Cross-Package Provider Functions
**Objective:** As a developer, I want to register constructors from other packages, so that I can provide standard-library or external dependencies.

#### Acceptance Criteria
1. When ProvideFunc references a function defined in an imported package (including the Go standard library), the Analyzer shall accept it as a provider function.
2. When generated bootstrap code calls a provider function from another package, the Analyzer shall include the required import.
3. When ProvideFunc references a function defined in the current package, the Analyzer shall reference it directly without adding a new import.

### Requirement 6: Bootstrap Wiring Integration
**Objective:** As a developer, I want ProvideFunc providers to appear in bootstrap wiring, so that generated code can construct dependencies from provider functions.

#### Acceptance Criteria
1. When ProvideFunc providers are registered and the dependency graph is valid, the Analyzer shall generate bootstrap wiring that calls provider functions in dependency order.
2. When a provider function depends on other dependencies, the Analyzer shall pass the resolved variables as arguments in the generated call.
3. When a provider function's return value is required by downstream dependencies, the Analyzer shall store it in a local variable for reuse.
4. If a provider function's return value is not required by any dependency, the Analyzer shall omit it from bootstrap wiring.

### Requirement 7: Diagnostics and Suggested Fix Behavior
**Objective:** As a developer, I want clear diagnostics for ProvideFunc usage, so that I can fix configuration errors quickly.

#### Acceptance Criteria
1. When ProvideFunc usage is invalid (non-function, missing return, unresolved dependency, or ambiguous interface), the Analyzer shall report a diagnostic at the ProvideFunc call site.
2. When ProvideFunc usage is valid, the Analyzer shall not emit diagnostics for the annotation itself.
3. When ProvideFunc contributes to bootstrap generation, the Analyzer shall include the provider function calls in the SuggestedFix output so they can be applied via `go vet -fix`.
4. The Analyzer shall include source position information and a clear correction message in ProvideFunc-related diagnostics.
