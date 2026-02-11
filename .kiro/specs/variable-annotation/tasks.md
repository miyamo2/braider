# Implementation Plan

- [x] 1. Variable annotation public API and option package
- [x] 1.1 (P) Define the Variable option types and annotation function
  - The `pkg/annotation/variable/option.go` and `annotation.Variable[T]()` already exist; verify they compile and that `Default`, `Typed[I]`, and `Named[N]` option types match the patterns used by `inject` and `provide`
  - Ensure `variable.Option` is recognized by the OptionExtractor infrastructure (internal annotation marker types: `VariableOption`, `VariableDefault`, `VariableTyped`, `VariableNamed`)
  - Add or verify example tests for basic, typed, and named Variable usage in `pkg/annotation/`
  - _Requirements: 1.1, 2.1, 2.2, 2.3_

- [x] 2. Variable call detection
- [x] 2.1 Implement the VariableCallDetector
  - Create a detector that scans package-level `var _ = annotation.Variable[T](value)` declarations
  - Identify the call expression, resolve the argument expression type via type info, and extract canonical expression text using AST formatting
  - Collect all package paths referenced by the expression (for import handling) and determine whether the expression is already package-qualified
  - Reject non-Variable annotation calls (Provide, Injectable, App) and emit a diagnostic error when the argument is missing or the expression type cannot be resolved
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6_
  - _Contracts: VariableCallDetector Service_

- [x] 2.2 Extend option extraction for Variable options
  - Add an extraction method that parses Variable option metadata from the type parameter of `annotation.Variable[T]`
  - Support `Default`, `Typed[I]`, `Named[N]`, and mixed anonymous-interface option combinations
  - Reuse existing default-check, typed-interface, and namer-type extraction logic by recognizing the Variable option package path alongside inject and provide paths
  - Validate namer implementations (must return a string literal) and report diagnostic errors for validation failures
  - Do not check for `WithoutConstructor` (not applicable to Variables)
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_
  - _Contracts: OptionExtractor Service (extension)_

- [x] 3. Variable registry
- [x] 3.1 (P) Implement the VariableRegistry
  - Create a thread-safe registry for Variable annotation entries, using the same nested-map and mutex pattern as the existing provider and injector registries
  - Store all necessary metadata: fully qualified type name, package path, package name, local type name, expression text, expression packages, qualification status, interface implementations, registered type, and optional name
  - Detect and reject duplicate `(TypeName, Name)` pairs, returning an error with conflicting information
  - Provide retrieval methods: get all entries in deterministic order, get by type name (unnamed), and get by type name and name (named)
  - Implement the dependency-info interface so Variable entries can participate in graph edge building (dependencies always empty)
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6_
  - _Contracts: VariableRegistry State_

- [x] 4. Dependency graph integration
- [x] 4.1 Extend graph node structure for Variable metadata
  - Add fields to the graph node to carry expression text, expression package references, and qualification status; these fields remain empty for non-Variable nodes
  - _Requirements: 4.1, 4.2_

- [x] 4.2 Extend graph building to include Variable nodes
  - Accept Variable registry entries when constructing the dependency graph
  - Create zero-dependency nodes with no constructor name, `IsField` set to false, and expression metadata populated from the registry entry
  - Use composite keys for named Variable nodes (`TypeName#Name`), consistent with existing named dependency handling
  - Ensure edge building correctly handles Variables (no edges created since dependency list is always empty)
  - Depends on 4.1 for the extended node structure
  - _Requirements: 4.1, 4.2, 4.4, 4.5, 4.6_
  - _Contracts: DependencyGraphBuilder Service (extension)_

- [x] 4.3 Extend interface registry to include Variable implementations
  - When building the interface registry, process Variable entries alongside providers and injectors so that `Typed[I]` registrations map the interface to the Variable's concrete type
  - Without this, dependencies referencing the interface type would fail to resolve
  - Depends on 4.2 for the Variable data flowing into the graph builder
  - _Requirements: 4.3, 4.5_
  - _Contracts: InterfaceRegistry Service (extension)_

- [x] 5. Bootstrap code generation for Variables
- [x] 5.1 Extend bootstrap generation to emit expression assignments
  - When generating initialization code for Variable nodes, emit a local variable assignment using the stored expression text instead of a constructor call
  - Apply package qualification for local references from other packages, using the package alias when available
  - Ensure Variable expression assignment is checked before the existing constructor-name validation, so Variable nodes do not trigger the "requires a constructor" error
  - Include necessary imports for packages referenced by Variable expressions, handling import alias collisions
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 7.3_
  - _Contracts: BootstrapGenerator Service (extension), CollectImports (extension)_

- [x] 5.2 Extend hash computation to include Variable expression data
  - Include Variable expression text in the graph hash so that adding, removing, or changing a Variable registration triggers bootstrap regeneration
  - Ensure the hash contribution is empty for non-Variable nodes, preserving existing hash values for projects that do not use Variables
  - _Requirements: 5.5, 5.6_
  - _Contracts: ComputeGraphHash (extension)_

- [x] 6. DependencyAnalyzer integration (Phase 2.5)
  - Wire the Variable call detector and Variable registry into the DependencyAnalyzer
  - Add a new phase (Phase 2.5, after Provide detection and before Inject re-detection) that detects Variable annotations, extracts options, determines the registered type, and registers entries in the Variable registry
  - Emit diagnostic errors for unresolvable expression types, namer validation failures, and type incompatibility with `Typed[I]`
  - Emit warning diagnostics for duplicate named Variable registrations, including both conflicting package paths
  - Cancel bootstrap generation on validation errors, consistent with existing Provide error handling
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 2.1, 2.2, 2.3, 2.4, 2.5, 3.1, 3.4, 3.5, 6.1, 6.2, 6.3, 6.4, 7.1_
  - _Contracts: DependencyAnalyzer (extension)_

- [x] 7. AppAnalyzer integration
  - Wire the Variable registry into the AppAnalyzer
  - Retrieve all Variable entries alongside providers and injectors when building the dependency graph
  - Pass Variable data to the graph builder and ensure generated bootstrap code correctly includes Variables from any scanned package
  - Add a diagnostic hint for unresolvable dependencies when the missing type matches a Variable registration with a name mismatch
  - _Requirements: 4.1, 4.5, 5.4, 6.5, 7.2_
  - _Contracts: AppAnalyzer (extension)_

- [ ] 8. Entry point wiring
  - Instantiate the VariableCallDetector and VariableRegistry in the CLI entry point
  - Pass both new components to the DependencyAnalyzer and AppAnalyzer constructors
  - Verify the complete analyzer chain builds and runs without errors on a project with no Variable annotations (backward compatibility)
  - _Requirements: 1.1, 3.1, 7.1, 7.2_

- [ ] 9. Integration tests
- [ ] 9.1 Basic Variable bootstrap tests
  - Create test cases for a basic Variable with default options, verifying the bootstrap IIFE contains the expected expression assignment
  - Create test cases for a Variable with `Typed[I]`, verifying the dependency is registered under the interface type and resolves correctly
  - Create test cases for a Variable with `Named[N]`, verifying the named composite key is used and the bootstrap assigns the correct variable name
  - Use the two-phase analysistest pattern: DependencyAnalyzer diagnostics validation followed by AppAnalyzer golden-file bootstrap validation
  - _Requirements: 1.1, 1.2, 1.3, 2.1, 2.2, 2.3, 4.1, 4.2, 4.3, 4.4, 5.1, 5.2_

- [ ] 9.2 Mixed and cross-package Variable tests
  - Create test cases where Variable, Provide, and Injectable dependencies coexist, verifying correct topological ordering in the generated bootstrap
  - Create test cases for mixed options (Typed and Named combined via anonymous interface embedding)
  - Create test cases for Variable annotations declared in non-main packages, verifying cross-package expression qualification and correct imports
  - _Requirements: 2.5, 4.5, 5.3, 5.4, 7.1, 7.2, 7.3_

- [ ] 9.3 Error and edge-case tests
  - Create test cases for missing or unresolvable Variable argument expressions, verifying diagnostic error emission
  - Create test cases for namer validation failures (non-literal return from `Name()`)
  - Create test cases for `Typed[I]` where the argument type does not implement the interface
  - Create test cases for duplicate named Variable registrations, verifying warning diagnostics
  - Create test cases for unresolvable dependency with name-mismatch hint
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

- [ ] 9.4 Idempotency tests
  - Create a test case with pre-existing bootstrap code whose hash matches the current Variable graph, verifying no regeneration occurs
  - Create a test case where the Variable expression changes, verifying hash mismatch triggers regeneration with an updated hash
  - _Requirements: 5.5, 5.6_

- [ ]* 9.5 Acceptance-criteria-focused unit tests
  - Unit tests for VariableCallDetector: basic detection, rejection of non-Variable calls, missing arguments, unresolvable types
  - Unit tests for VariableRegistry: register/retrieve, duplicate detection, thread safety, deterministic ordering, named lookup
  - Unit tests for OptionExtractor Variable extension: Default, Typed, Named, mixed options, validation errors
  - Unit tests for hash computation: verify existing hashes unchanged when expression text is empty, verify hash changes when expression text varies
  - _Requirements: 1.1, 1.2, 1.5, 1.6, 2.1, 2.2, 2.3, 2.4, 2.5, 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 5.5, 5.6_
