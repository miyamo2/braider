# Implementation Plan

- [x] 1. Define container data models and extend App annotation detection for the generic form
- [x] 1.1 Define the container-related data models used throughout the pipeline
  - Introduce the `ContainerDefinition`, `ContainerField`, `ResolvedContainerField`, and `AppOptionMetadata` data structures in the detect layer
  - `ContainerDefinition` captures whether the container is named or anonymous, its type metadata, and an ordered list of fields
  - `ContainerField` captures the field name, type, stringified type, `braider` struct tag value, and source position
  - `ResolvedContainerField` maps a container field name to its dependency graph node key and bootstrap variable name
  - `AppOptionMetadata` carries either the default flag or a pointer to the container definition
  - _Requirements: 1.1, 1.2, 1.3_

- [x] 1.2 Extend the App annotation detector to handle generic `annotation.App[T](main)` calls
  - Extend the App call detection logic to unwrap `*ast.IndexExpr` from the call expression's function position, reaching the underlying selector expression
  - Preserve the type argument expression from the index expression so that downstream option extraction can classify it
  - Add the type argument expression field to the App annotation result structure
  - Ensure non-generic `annotation.App(main)` calls continue to work unchanged (type argument expression is nil)
  - _Requirements: 1.1, 1.4_
  - _Contracts: AppDetector Service_

- [x] 1.3 Add unit tests for generic App call detection
  - Test that `annotation.App[app.Container[T]](main)` is detected and the type argument expression is populated
  - Test that `annotation.App[app.Default](main)` is detected and the type argument expression is populated
  - Test that `annotation.App(main)` (non-generic) continues to be detected with a nil type argument expression
  - _Requirements: 1.1, 1.4_

- [x] 2. Implement the App option extractor to classify App type parameters
- [x] 2.1 (P) Implement the App option extractor component
  - Create a new component that resolves the type argument in `App[T]` using type-checker information
  - Determine whether the resolved type implements the `AppContainer` marker interface from the internal annotation package, using the same `types.Implements` pattern as existing option extraction
  - When the type implements `AppContainer`, extract the inner type parameter (the user's struct) by unwrapping the named type's type arguments
  - Support mixed options via anonymous interface embedding by searching embedded interfaces for `AppContainer` implementations
  - For named struct types, extract the fully qualified type name, package path, package name, and local name
  - For anonymous struct types, extract field definitions (name, type, struct tag) directly from the `types.Struct`
  - For both named and anonymous structs, populate the ordered field list in the container definition
  - Return default metadata when the type argument is nil (non-generic call) or when the type argument is `app.Default`
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 7.1, 7.2_
  - _Contracts: AppOptionExtractor Service_

- [x] 2.2 (P) Add unit tests for the App option extractor
  - Test extraction of `app.Default` returns default metadata
  - Test extraction of `app.Container[NamedStruct]` returns container definition with correct type info and fields
  - Test extraction of `app.Container[struct{...}]` returns anonymous container definition with field definitions
  - Test extraction of mixed options via anonymous interface embedding (e.g., `interface{ app.Container[T] }`) correctly identifies the container option
  - Test that non-generic calls (nil type argument) return default metadata
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 7.1, 7.2_

- [x] 3. Implement container struct validation
- [x] 3.1 (P) Implement the container validator component
  - Create a new component that validates all container struct fields against the dependency graph
  - Verify that the container type parameter is a struct type; reject non-struct types with a validation error
  - For each field, check for the `braider` struct tag: reject `braider:"-"` (exclusion not permitted in containers) and `braider:""` (ambiguous intent) with specific validation errors
  - For fields with `braider:"name"` tags, verify that a dependency registered under that name exists in the graph
  - For fields without a named tag, resolve the field type against graph nodes and the interface registry; detect unresolved types
  - Detect ambiguous resolution when a field type matches multiple registered dependencies and no struct tag disambiguates
  - Include source position information in all validation errors for IDE navigation
  - Return an empty error list when all fields are valid
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 6.1, 6.2, 6.4, 6.5_
  - _Contracts: ContainerValidator Service_

- [x] 3.2 (P) Add unit tests for the container validator
  - Test acceptance of a valid container struct with resolvable field types
  - Test rejection of non-struct type parameter (e.g., `int`) with appropriate error message
  - Test rejection of `braider:"-"` struct tag with error indicating exclusion is not permitted
  - Test rejection of empty `braider:""` struct tag with error indicating ambiguous intent
  - Test detection of unresolved concrete field type
  - Test detection of unresolved interface field type
  - Test detection of ambiguous field resolution (multiple candidates, no disambiguating tag)
  - Test that `braider:"name"` tag resolves correctly when the named dependency exists
  - Test that `braider:"name"` tag produces error when the named dependency does not exist
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 6.1, 6.2, 6.4_

- [x] 4. Implement container field resolution
- [x] 4.1 (P) Implement the container resolver component
  - Create a new component that maps each validated container field to its dependency graph node key
  - For fields with `braider:"name"` struct tags, construct the composite key (`TypeName#Name`) and look up in the graph
  - For interface-typed fields, use the interface registry to resolve to the implementing concrete node
  - For concrete-typed fields, match directly against graph node type names
  - Return an ordered list of resolved container fields preserving the container struct's field order
  - Each resolved field includes the dependency variable name used in bootstrap initialization code
  - _Requirements: 2.3, 2.4, 2.7, 3.4, 3.5, 3.6_
  - _Contracts: ContainerResolver Service_

- [x] 4.2 (P) Add unit tests for the container resolver
  - Test resolution of concrete type fields to graph nodes
  - Test resolution of interface type fields via the interface registry
  - Test resolution of named fields (`braider:"name"`) using composite keys
  - Test that resolution order matches the container struct field order
  - Test that resolved variable names are correct for downstream code generation
  - _Requirements: 2.3, 2.4, 2.7, 3.4, 3.5, 3.6_

- [x] 5. Extend container-specific diagnostics
- [x] 5.1 (P) Add container-specific diagnostic emission methods
  - Add a method to report non-struct container type parameter errors with the type name and source position
  - Add a method to report unresolvable container field errors identifying the field name, expected type, and reason
  - Add a method to report ambiguous container field resolution listing the field name, type, and candidate dependency names
  - Add a method to report forbidden or invalid struct tags in container definitions with the field name and reason
  - All diagnostic methods must include source position for IDE navigation
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_
  - _Contracts: DiagnosticEmitter Service_

- [x] 5.2 (P) Add unit tests for container diagnostic emission
  - Test each new diagnostic method emits the expected message format and includes position information
  - Verify error messages are actionable and identify the specific field or type causing the issue
  - _Requirements: 6.1, 6.2, 6.4, 6.5_

- [x] 6. Extend hash computation to include container field definitions
- [x] 6.1 (P) Extend the graph hash computation to incorporate container struct metadata
  - When a container definition is provided, append field data (field name, type string, struct tag) to the hash input after graph node data
  - When no container definition is provided (nil), produce the exact same hash as the current implementation to avoid any regression
  - The hash must change when container fields are added, removed, reordered, renamed, or when their types or tags change
  - _Requirements: 5.1, 5.2, 5.3_
  - _Contracts: ComputeGraphHash Service_

- [x] 6.2 (P) Add unit tests for extended hash computation
  - Test that the hash without a container definition matches the existing `ComputeGraphHash` output
  - Test that the hash changes when a container field is added or removed
  - Test that the hash changes when a container field is renamed or its type changes
  - Test that the hash changes when a container field's struct tag changes
  - Test that the hash changes when container field order changes
  - Test that the hash remains stable when nothing changes
  - _Requirements: 5.1, 5.2, 5.3_

- [x] 7. Extend import collection for container types
- [x] 7.1 (P) Extend the import collector to include container type and field type packages
  - When the container is a named struct from an external package, add its package import path to the collected imports
  - For each field in the container struct that references a type from an external package, add that package import path
  - Reuse the existing collision-safe alias generation logic for all new imports
  - When no container definition is provided, the import collection behavior remains unchanged
  - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_
  - _Contracts: CollectImports Service_

- [x] 7.2 (P) Add unit tests for container import collection
  - Test that a named container struct from an external package triggers an import for that package
  - Test that fields referencing external package types trigger the corresponding imports
  - Test that anonymous container fields with external types also trigger imports
  - Test that collision-safe aliases are applied when container imports conflict with existing imports
  - Test that nil container definition produces the same import set as the current behavior
  - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

- [x] 8. Implement container-aware bootstrap code generation
- [x] 8.1 Implement the container bootstrap generation method
  - Add a new generation method that produces a bootstrap IIFE returning the user-defined container struct instead of the auto-generated anonymous struct
  - For named container types, use the qualified type name as the IIFE return type (e.g., `func() pkg.MyContainer { ... }()`)
  - For anonymous container types, use the struct type literal as the IIFE return type (e.g., `func() struct { Field Type } { ... }()`)
  - Generate initialization code for all dependencies required to populate container fields, following topological order
  - Generate initialization code for transitive dependencies that are not container fields but are needed to construct container field values
  - Generate a return statement that populates each container struct field with its corresponding resolved dependency value
  - Include the `// braider:hash:<hash>` comment on the generated bootstrap variable for idempotency
  - Dependencies not in the container struct (transitive only) remain as local variables within the IIFE
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7_
  - _Contracts: BootstrapGenerator Service_

- [x] 8.2 Implement the container-aware idempotency check method
  - Add a method that checks whether the existing bootstrap hash matches the computed hash including container field definitions
  - When the hash matches, signal that regeneration should be skipped
  - When the hash differs, signal that regeneration is needed
  - Delegates to the extended hash computation from task 6
  - _Requirements: 5.1, 5.2, 5.3_
  - _Contracts: BootstrapGenerator Service_

- [x] 8.3 Add unit tests for container bootstrap generation
  - Test that a named container type produces an IIFE with the qualified type name as return type
  - Test that an anonymous container type produces an IIFE with the struct literal as return type
  - Test that initialization code follows topological order
  - Test that transitive dependencies not in the container are generated as local variables
  - Test that the return statement correctly maps container fields to resolved dependency values
  - Test that the hash comment is present in the generated output
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 5.1, 5.2, 5.3_

- [x] 9. Extend the App analyzer orchestration to support the container pipeline
- [x] 9.1 Wire new components into the App analyzer runner
  - Add the App option extractor, container validator, and container resolver as dependencies of the App analyzer runner
  - After App detection, invoke the option extractor to classify the detected annotation as default or container mode
  - In container mode: invoke container validation, then container resolution, then container-aware bootstrap generation
  - In default mode: follow the existing generation path unchanged
  - If container validation returns errors, emit diagnostics via the diagnostic emitter and skip code generation
  - Generate the `_ = dependency` reference in the main function body, consistent with default App behavior
  - _Requirements: 1.1, 1.4, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 3.1, 3.4, 3.5, 3.6, 3.7, 3.8, 5.1, 5.2, 5.3, 6.1, 6.2, 6.3, 6.4, 6.5, 7.1, 7.2_
  - _Contracts: AppAnalyzeRunner Service_

- [x] 9.2 Update braider's own DI bootstrap to wire the new components
  - Register the new components (App option extractor, container validator, container resolver) in braider's own DI annotations in the CLI entry point
  - Regenerate braider's bootstrap code to include the new dependencies in the analyzer runner's constructor
  - _Requirements: 1.1_

- [x] 10. Add integration tests for container bootstrap scenarios
- [x] 10.1 Add integration test for basic named container struct
  - Create a testdata case where `annotation.App[app.Container[MyContainer]](main)` uses a named container struct with simple fields
  - Verify the bootstrap IIFE returns the named type and initializes all fields correctly
  - Include `.golden` file with expected output after SuggestedFix application
  - _Requirements: 1.1, 1.2, 2.1, 3.1, 3.2, 3.4, 3.6, 3.7, 3.8_

- [x] 10.2 Add integration test for anonymous container struct
  - Create a testdata case where `annotation.App[app.Container[struct{...}]](main)` uses an inline anonymous struct
  - Verify the bootstrap IIFE returns the struct literal type and initializes all fields correctly
  - _Requirements: 1.2, 1.3, 3.1, 3.3, 3.4, 3.6_

- [x] 10.3 Add integration test for named field resolution via struct tag
  - Create a testdata case where a container field has a `braider:"name"` struct tag and the corresponding named dependency is registered
  - Verify the field is resolved via the named key and the generated code is correct
  - _Requirements: 2.4, 3.4, 3.6_

- [x] 10.4 Add integration test for interface-typed container field
  - Create a testdata case where a container field has an interface type and a concrete implementation is registered
  - Verify the interface resolution works and the bootstrap correctly initializes the field
  - _Requirements: 2.3, 3.4, 3.6, 6.1_

- [x] 10.5 Add integration test for cross-package container struct
  - Create a testdata case where the container struct is defined in an external package
  - Verify the import for the container's package is generated and the qualified type name is used in the return type
  - _Requirements: 3.2, 4.3, 4.4, 4.5_

- [x] 10.6 Add integration test for transitive dependencies
  - Create a testdata case where container fields require transitive dependencies that are not themselves container fields
  - Verify all transitive initialization code is generated and transitive dependencies appear as local variables
  - _Requirements: 3.4, 3.5, 3.6_

- [x] 10.7 Add integration test for container with Variable dependencies
  - Create a testdata case where a container field is satisfied by a Variable-registered dependency
  - Verify the Variable expression assignment is generated correctly in the bootstrap
  - _Requirements: 3.4, 3.6, 4.2_

- [x] 10.8 Add integration test for container idempotent regeneration
  - Create a testdata case where the existing bootstrap hash matches the computed hash including container field definitions
  - Verify that no `// want` diagnostic is emitted (regeneration is skipped)
  - _Requirements: 5.1, 5.3_

- [x] 10.9 Add integration test for container outdated regeneration
  - Create a testdata case where the existing bootstrap hash differs from the computed hash (e.g., container struct was modified)
  - Verify that regeneration produces updated bootstrap code
  - _Requirements: 5.2, 5.3_

- [x] 10.10 Add integration test for mixed options via anonymous interface embedding
  - Create a testdata case using `annotation.App[interface{ app.Container[T] }](main)`
  - Verify that the container option is correctly extracted and the bootstrap is generated as expected
  - _Requirements: 7.1, 7.2_

- [x] 11. Add integration tests for container error scenarios
- [x] 11.1 (P) Add integration test for non-struct container type parameter
  - Create a testdata case using `app.Container[int]` and verify a diagnostic error is emitted indicating a struct type is required
  - _Requirements: 2.2, 6.4, 6.5_

- [x] 11.2 (P) Add integration test for unresolved container field
  - Create a testdata case where a container field type has no registered dependency
  - Verify a diagnostic error is emitted identifying the unresolved field and type
  - _Requirements: 2.3, 6.1, 6.2, 6.5_

- [x] 11.3 (P) Add integration test for ambiguous container field
  - Create a testdata case where a container field type matches multiple registered dependencies without a disambiguating struct tag
  - Verify a diagnostic error is emitted listing the ambiguous candidates
  - _Requirements: 2.7, 6.5_

- [x] 11.4 (P) Add integration test for forbidden `braider:"-"` struct tag
  - Create a testdata case where a container field has `braider:"-"` and verify a diagnostic error is emitted
  - _Requirements: 2.5, 6.5_

- [x] 11.5 (P) Add integration test for empty `braider:""` struct tag
  - Create a testdata case where a container field has `braider:""` and verify a diagnostic error is emitted
  - _Requirements: 2.6, 6.5_

- [x] 12. Verify full build and test suite pass
  - Run `go build ./...` to confirm all packages compile without errors
  - Run `go test ./...` to confirm all existing and new tests pass
  - Verify that the default (non-container) App behavior is not regressed by running all existing bootstrap generation tests
  - Verify that braider's own self-hosting bootstrap in `cmd/braider/main.go` regenerates correctly with the new components wired
  - _Requirements: 1.4, 3.8, 6.3_
