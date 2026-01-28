# Implementation Plan

## Task Overview

This implementation plan covers the bootstrap-with-app-annotation feature, which extends braider to detect `annotation.App(main)` markers and generate application bootstrap code. The feature introduces a multichecker architecture with two analyzers (InjectAnalyzer and AppAnalyzer), global injectable registry, dependency graph construction, and IIFE-based bootstrap code generation.

---

## Tasks

- [ ] 1. Implement the global InjectableRegistry for cross-package injectable discovery
- [ ] 1.1 Create the InjectableInfo data structure
  - Define the struct to hold injectable metadata including type name, package path, constructor name, dependencies, pending status, and implemented interfaces
  - Support fully qualified type names as map keys for unique identification
  - Include IsPending flag to track constructors being generated in the current pass
  - _Requirements: 2.3, 9.1, 9.2, 9.3_

- [ ] 1.2 Implement the thread-safe InjectableRegistry
  - Create a global singleton registry with mutex-protected operations
  - Implement Register method to add or update injectable entries
  - Implement GetAll method to retrieve all registered injectables as a snapshot
  - Implement Get method to retrieve a specific injectable by type name
  - Implement Clear method for test isolation
  - _Requirements: 2.3, 9.3_

- [ ] 1.3 Add unit tests for InjectableRegistry
  - Test registration of new injectables with various metadata
  - Test retrieval by type name and full list retrieval
  - Test update behavior when registering the same type twice
  - Test thread safety with concurrent access patterns
  - Test Clear functionality for test isolation
  - _Requirements: 2.3, 9.3_

- [ ] 2. Refactor existing analyzer into InjectAnalyzer with registry integration
- [ ] 2.1 Split the existing analyzer into InjectAnalyzer
  - Extract inject detection and constructor generation logic into a dedicated InjectAnalyzer
  - Configure the analyzer to run on all packages and detect annotation.Inject structs
  - Maintain existing detection components (InjectDetector, StructDetector, FieldAnalyzer)
  - _Requirements: 2.1, 2.2_

- [ ] 2.2 Integrate InjectAnalyzer with InjectableRegistry
  - Register discovered injectables to GlobalRegistry after detection
  - Set IsPending flag based on whether constructor exists on disk or is being generated
  - Populate Dependencies field by analyzing struct fields
  - Detect and populate Implements field using types.Implements for interface matching
  - _Requirements: 2.1, 2.3, 9.1, 9.2_

- [ ] 2.3 Update CLI entry point for multichecker architecture
  - Replace singlechecker.Main with multichecker.Main
  - Configure InjectAnalyzer and AppAnalyzer with proper dependency ordering
  - Ensure AppAnalyzer declares InjectAnalyzer as a required dependency
  - _Requirements: 9.1_

- [ ] 2.4 Add integration tests for InjectAnalyzer with registry
  - Test that injectables are registered from single package
  - Test that injectables are registered from multiple packages
  - Test pending vs existing constructor status tracking
  - Test interface implementation detection
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 9.1_

- [ ] 3. Implement AppDetector for annotation.App(main) detection
- [ ] 3.1 Create the AppAnnotation data structure
  - Define struct to hold call expression, containing declaration, main function reference, and position
  - Include validation state for error reporting
  - _Requirements: 1.1_

- [ ] 3.2 Implement DetectAppAnnotation method
  - Traverse AST to find var declarations with annotation.App calls
  - Validate the called function is from the annotation package
  - Handle aliased imports of the annotation package
  - Return nil when no App annotation is present in the package
  - _Requirements: 1.1, 1.3_

- [ ] 3.3 Implement ValidateAppAnnotation method
  - Check that exactly one App annotation exists per package
  - Verify the App annotation argument references the main function
  - Report diagnostic for multiple App annotations with all positions
  - Report diagnostic when App references a non-main function
  - _Requirements: 1.2, 1.4_

- [ ] 3.4 Add unit tests for AppDetector
  - Test detection of valid annotation.App(main) declaration
  - Test skipping when no App annotation is present
  - Test error reporting for multiple App annotations
  - Test error reporting when App references non-main function
  - Test handling of aliased annotation package imports
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [ ] 4. Implement InterfaceRegistry for interface-to-implementation resolution
- [ ] 4.1 Create the InterfaceRegistry with Build method
  - Build mapping from interface types to implementing injectable structs
  - Use types.Implements to detect interface implementations for both value and pointer receivers
  - Support building from the complete set of registered injectables
  - _Requirements: 3.5, 3.8_

- [ ] 4.2 Implement Resolve method with error handling
  - Return the fully qualified type name of the implementing injectable
  - Return AmbiguousImplementationError when multiple implementations exist
  - Return UnresolvedInterfaceError when no implementation is found
  - _Requirements: 3.5, 3.6, 3.7_

- [ ] 4.3 Add unit tests for InterfaceRegistry
  - Test building registry from injectables with various interface implementations
  - Test resolving interface to single implementation
  - Test error when multiple implementations exist for same interface
  - Test error when no implementation exists for required interface
  - Test cross-package interface resolution
  - _Requirements: 3.5, 3.6, 3.7, 3.8_

- [ ] 5. Implement DependencyGraph for constructor parameter analysis
- [ ] 5.1 Create the Graph and Node data structures
  - Define Graph struct with nodes map and edges map
  - Define Node struct with type metadata, dependencies, in-degree counter, and injectable reference
  - Support efficient lookup by fully qualified type name
  - _Requirements: 3.1, 3.2_

- [ ] 5.2 Implement BuildGraph method with constructor validation
  - Retrieve all injectables from the registry
  - Validate that each injectable has a constructor (existing or pending)
  - Parse constructor parameter types and create dependency edges
  - Filter non-injectable parameters and handle interface resolution via InterfaceRegistry
  - Return error for missing constructors, unresolved interfaces, or unresolvable parameters
  - _Requirements: 2.4, 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7_

- [ ] 5.3 Add unit tests for DependencyGraph
  - Test graph construction from simple dependency chains
  - Test filtering of non-injectable parameter types
  - Test error when injectable lacks constructor
  - Test error when interface parameter cannot be resolved
  - Test error when concrete parameter is not injectable
  - Test handling of constructors with multiple return values
  - _Requirements: 2.4, 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7_

- [ ] 6. Implement TopologicalSort with cycle detection
- [ ] 6.1 Implement Kahn's algorithm for topological ordering
  - Compute in-degree for each node in the graph
  - Process nodes with zero in-degree, maintaining alphabetical order for determinism
  - Decrement in-degrees of dependent nodes and add newly zero-degree nodes to queue
  - Return ordered list of type names when complete
  - _Requirements: 5.1, 5.2_

- [ ] 6.2 Implement cycle detection and path reconstruction
  - Detect cycles when nodes with non-zero in-degree remain after sort
  - Reconstruct the cycle path for error reporting using BFS or DFS
  - Return CycleError with full cycle path
  - _Requirements: 4.1, 4.2, 4.3_

- [ ] 6.3 Add unit tests for TopologicalSort
  - Test ordering of simple linear dependency chain
  - Test deterministic alphabetical ordering when multiple valid orders exist
  - Test handling of empty graph (no injectables)
  - Test cycle detection with full path reconstruction
  - Test handling of disconnected components in the graph
  - _Requirements: 4.1, 4.2, 4.3, 5.1, 5.2, 5.3_

- [ ] 7. Implement BootstrapGenerator for IIFE code generation
- [ ] 7.1 Implement GenerateBootstrap method
  - Generate IIFE pattern with anonymous struct return type
  - Generate constructor calls in topological order using field names as intermediate variables
  - Generate return statement with struct literal initializing all fields
  - Collect required import paths for external package types
  - _Requirements: 6.1, 6.2, 7.1, 7.2, 7.3, 7.4_

- [ ] 7.2 Implement field name derivation with conflict handling
  - Convert type names to lowerCamelCase for field names
  - Handle all-caps abbreviations (e.g., DB -> db)
  - Append numeric suffix when field name conflicts occur
  - _Requirements: 7.2_

- [ ] 7.3 Implement code formatting with gofmt compliance
  - Use go/format package to ensure generated code passes gofmt
  - Handle formatting errors gracefully with informative error messages
  - _Requirements: 7.5_

- [ ] 7.4 Implement idempotency detection using hash comparison
  - Compute deterministic SHA-256 hash from ordered type names and dependencies
  - Truncate hash to first 8 hex characters for comment marker
  - Generate hash comment marker above dependency variable declaration
  - Compare hash values to determine if regeneration is needed
  - _Requirements: 6.5, 6.6_

- [ ] 7.5 Implement dependency reference detection in main function
  - Walk main function body to find references to the dependency variable
  - Detect identifier references and selector expressions accessing dependency
  - Exclude blank identifier assignments from consideration
  - Return whether _ = dependency insertion should be skipped
  - _Requirements: 6.3_

- [ ] 7.6 Implement existing bootstrap detection
  - Find existing dependency variable declaration in the package
  - Extract hash comment marker for comparison
  - Support both new insertion and replacement scenarios
  - _Requirements: 6.5, 6.6_

- [ ] 7.7 Add unit tests for BootstrapGenerator
  - Test IIFE generation with single injectable
  - Test IIFE generation with multiple injectables in dependency order
  - Test field name derivation with various naming patterns
  - Test field name conflict resolution with numeric suffixes
  - Test import collection for cross-package types
  - Test gofmt compliance of generated code
  - Test idempotency hash computation and comparison
  - Test dependency reference detection in main function
  - Test detection and handling of existing bootstrap code
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 7.1, 7.2, 7.3, 7.4, 7.5_

- [ ] 8. Extend DiagnosticEmitter with bootstrap-specific error reporting
- [ ] 8.1 Implement bootstrap error emission methods
  - Add EmitMultipleAppError for reporting multiple annotation.App declarations
  - Add EmitNonMainAppError for reporting App referencing non-main function
  - Add EmitMissingConstructorError for reporting injectable without constructor
  - Add EmitCircularDependencyError for reporting cycles with full path
  - Add EmitAmbiguousImplementationError for reporting multiple interface implementations
  - Add EmitUnresolvedInterfaceError for reporting unresolved interface dependencies
  - Add EmitUnresolvableParameterError for reporting non-injectable constructor parameters
  - All methods include source position and descriptive messages
  - _Requirements: 8.1, 8.2_

- [ ] 8.2 Add unit tests for bootstrap diagnostic emission
  - Test each error emission method produces correct diagnostic format
  - Test source position inclusion in all error messages
  - Test message clarity and actionability
  - _Requirements: 8.1, 8.2, 8.3_

- [ ] 9. Extend SuggestedFixBuilder with bootstrap fix generation
- [ ] 9.1 Implement BuildBootstrapFix for new bootstrap insertion
  - Calculate insertion position after App annotation declaration
  - Create SuggestedFix with dependency variable code
  - Include descriptive message explaining the fix action
  - _Requirements: 6.1, 8.3_

- [ ] 9.2 Implement BuildBootstrapReplacementFix for updating existing bootstrap
  - Calculate replacement range covering existing dependency declaration
  - Create SuggestedFix with updated bootstrap code
  - Include descriptive message indicating bootstrap update
  - _Requirements: 6.6, 8.3_

- [ ] 9.3 Implement BuildMainReferenceFix for adding dependency reference
  - Calculate insertion position at start of main function body
  - Create SuggestedFix with _ = dependency statement
  - Skip generation when dependency is already referenced in main
  - _Requirements: 6.3, 8.3_

- [ ] 9.4 Add unit tests for bootstrap fix builders
  - Test bootstrap fix generation with proper position calculation
  - Test replacement fix generation preserving correct code range
  - Test main reference fix generation at correct position
  - Test fix message clarity and description
  - _Requirements: 6.1, 6.3, 6.6, 8.3_

- [ ] 10. Implement AppAnalyzer to orchestrate bootstrap generation
- [ ] 10.1 Create the AppAnalyzer with required dependency on InjectAnalyzer
  - Define analyzer with proper name, documentation, and required analyzers
  - Configure to use inspect.Analyzer for AST traversal
  - _Requirements: 9.1_

- [ ] 10.2 Implement the Run function for AppAnalyzer
  - Detect App annotation using AppDetector; skip if not found
  - Validate App annotation and emit diagnostics for invalid cases
  - Retrieve all injectables from GlobalRegistry
  - Build dependency graph with InterfaceRegistry for interface resolution
  - Perform topological sort and handle cycle errors
  - Generate bootstrap code using BootstrapGenerator
  - Check for existing bootstrap and determine if update is needed
  - Emit appropriate SuggestedFix for insertion or replacement
  - Handle dependency reference in main function
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 9.1_

- [ ] 10.3 Add integration tests for AppAnalyzer
  - Test bootstrap generation with valid App annotation and injectables
  - Test skipping bootstrap when no App annotation present
  - Test error diagnostics for invalid App annotations
  - Test error diagnostics for dependency graph issues
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 6.1_

- [ ] 11. Add golden file tests for end-to-end bootstrap scenarios
- [ ] 11.1 Create test fixtures for basic bootstrap scenarios
  - Create simple test case with single injectable and constructor
  - Create multitype test case with multiple injectables in dependency order
  - Create crosspackage test case with injectables from multiple packages
  - _Requirements: 6.1, 6.2, 6.4, 7.1, 7.2, 7.3, 7.4_

- [ ] 11.2 Create test fixtures for error scenarios
  - Create circular dependency test case with expected error diagnostic
  - Create multiple App annotation test case with expected error
  - Create non-main App reference test case with expected error
  - Create unresolvable parameter test case with expected error
  - _Requirements: 1.2, 1.4, 4.1, 4.3_

- [ ] 11.3 Create test fixtures for interface resolution scenarios
  - Create interface dependency test case with injectable implementation
  - Create ambiguous implementation test case with expected error
  - Create unresolved interface test case with expected error
  - Create crosspackage interface test case with implementation in different package
  - _Requirements: 3.5, 3.6, 3.7, 3.8_

- [ ] 11.4 Create test fixtures for idempotency and update scenarios
  - Create test case with existing up-to-date bootstrap (no changes expected)
  - Create test case with outdated bootstrap requiring update
  - Create test case where dependency is already referenced in main
  - _Requirements: 6.3, 6.5, 6.6_

- [ ] 11.5 Create test fixtures for single-pass and module-wide scenarios
  - Create singlepass test case with constructors being generated in same pass
  - Create modulewide test case discovering injectables without explicit imports
  - Create pending constructor test case validating registry behavior
  - _Requirements: 9.1, 9.2, 9.3_

- [ ] 11.6 Run golden file tests and verify output
  - Execute all test cases using analysistest.RunWithSuggestedFixes
  - Verify generated bootstrap code matches expected golden files
  - Verify error diagnostics match expected patterns
  - _Requirements: 6.4_

---

## Requirements Coverage

| Requirement | Tasks |
|-------------|-------|
| 1.1 | 3.1, 3.2, 3.4, 10.2, 10.3 |
| 1.2 | 3.3, 3.4, 10.2, 10.3, 11.2 |
| 1.3 | 3.2, 3.4, 10.2, 10.3 |
| 1.4 | 3.3, 3.4, 10.2, 10.3, 11.2 |
| 2.1 | 2.1, 2.2, 2.4 |
| 2.2 | 2.1, 2.4 |
| 2.3 | 1.1, 1.2, 1.3, 2.2, 2.4 |
| 2.4 | 2.4, 5.2, 5.3 |
| 3.1 | 5.1, 5.2, 5.3 |
| 3.2 | 5.1, 5.2, 5.3 |
| 3.3 | 5.2, 5.3 |
| 3.4 | 5.2, 5.3 |
| 3.5 | 4.1, 4.2, 4.3, 5.2, 5.3, 11.3 |
| 3.6 | 4.2, 4.3, 5.2, 5.3, 11.3 |
| 3.7 | 4.2, 4.3, 5.2, 5.3, 11.3 |
| 3.8 | 4.1, 4.3, 11.3 |
| 4.1 | 6.2, 6.3, 11.2 |
| 4.2 | 6.2, 6.3 |
| 4.3 | 6.2, 6.3, 11.2 |
| 5.1 | 6.1, 6.3 |
| 5.2 | 6.1, 6.3 |
| 5.3 | 6.3 |
| 6.1 | 7.1, 7.7, 9.1, 9.4, 10.2, 10.3, 11.1 |
| 6.2 | 7.1, 7.7, 10.2, 11.1 |
| 6.3 | 7.5, 7.7, 9.3, 9.4, 10.2, 11.4 |
| 6.4 | 7.7, 10.2, 11.1, 11.6 |
| 6.5 | 7.4, 7.6, 7.7, 10.2, 11.4 |
| 6.6 | 7.4, 7.6, 7.7, 9.2, 9.4, 10.2, 11.4 |
| 7.1 | 7.1, 7.7, 11.1 |
| 7.2 | 7.1, 7.2, 7.7, 11.1 |
| 7.3 | 7.1, 7.7, 11.1 |
| 7.4 | 7.1, 7.7, 11.1 |
| 7.5 | 7.3, 7.7 |
| 8.1 | 8.1, 8.2 |
| 8.2 | 8.1, 8.2 |
| 8.3 | 8.2, 9.1, 9.2, 9.3, 9.4 |
| 9.1 | 1.1, 2.2, 2.3, 2.4, 10.1, 10.2, 11.5 |
| 9.2 | 1.1, 2.2, 2.4, 11.5 |
| 9.3 | 1.1, 1.2, 1.3, 2.4, 11.5 |

All 37 acceptance criteria across 9 requirements are covered by the implementation tasks.
