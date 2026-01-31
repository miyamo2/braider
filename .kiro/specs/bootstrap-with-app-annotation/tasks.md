# Implementation Plan

## Task Overview

This implementation plan covers the bootstrap-with-app-annotation feature for braider. The feature introduces a multichecker architecture with two analyzers (DependencyAnalyzer and AppAnalyzer) that work together to detect annotation markers, discover dependencies across packages, and generate IIFE-based bootstrap code.

---

## Tasks

- [x] 1. Establish Global Registry Infrastructure
- [x] 1.1 (P) Implement the Provider Registry for storing discovered Provide-annotated structs
  - Create thread-safe registry with mutex protection for parallel analyzer execution
  - Support registration of provider information including type name, package path, constructor name, and dependencies
  - Implement retrieval methods for getting all providers or specific provider by type name
  - Include clear method for test isolation
  - Populate implements field with interface types the struct satisfies
  - _Requirements: 2.3_

- [x] 1.2 (P) Implement the Injector Registry for storing discovered Inject-annotated structs
  - Create thread-safe registry with mutex protection for parallel analyzer execution
  - Support registration of injector information including type name, package path, constructor name, and dependencies
  - Implement retrieval methods for getting all injectors or specific injector by type name
  - Include clear method for test isolation
  - Populate implements field with interface types the struct satisfies
  - _Requirements: 9.1_

- [x] 1.3 Implement the Package Tracker for cross-package synchronization
  - Create channel-based synchronization mechanism for package completion tracking
  - Support marking individual packages as scanned when DependencyAnalyzer completes
  - Implement waiting mechanism that blocks until all expected packages are scanned
  - Handle timeout scenarios to prevent deadlocks
  - Include clear method for test isolation
  - _Requirements: 9.2, 9.3_

- [x] 2. Implement Detection Components
- [x] 2.1 (P) Implement Provide annotation detection capability
  - Detect structs embedding annotation.Provide marker via AST traversal
  - Extract struct type information and package path
  - Identify constructor function following New<TypeName> naming convention
  - Detect interface implementations using go/types.Implements
  - Exclude structs without Provide marker from dependency graph consideration
  - _Requirements: 2.1, 2.2_

- [x] 2.2 Implement App annotation detection and validation
  - Detect var _ = annotation.App(main) declarations in package AST
  - Validate exactly one App annotation exists per package
  - Verify App annotation references the main function specifically
  - Extract position information for diagnostic reporting
  - Handle aliased imports of annotation package
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 2.3 Implement Package Loader for module-wide package enumeration
  - Use golang.org/x/tools/go/packages to load all packages in module
  - Return list of package paths for synchronization with PackageTracker
  - Handle module root detection and recursive package discovery
  - _Requirements: 9.2_

- [x] 3. Implement Dependency Analyzer
- [x] 3.1 Create DependencyAnalyzer as primary analyzer for struct detection
  - Define analyzer with proper name, documentation, and dependencies on inspect.Analyzer
  - Implement run function that processes each package
  - Detect both Inject-annotated structs for constructor generation and Provide-annotated structs for registry
  - Register discovered providers to GlobalProviderRegistry
  - Register discovered injectors to GlobalInjectorRegistry
  - Mark package as scanned via GlobalPackageTracker upon completion
  - _Requirements: 2.1, 2.2, 9.1, 9.2_

- [x] 3.2 Implement constructor validation for Provide-annotated structs
  - Verify each Provide struct has corresponding New<TypeName> constructor function
  - Report error diagnostic when constructor is missing
  - Extract constructor parameter types as dependency information
  - Use first return value as the provided type
  - _Requirements: 2.4, 3.4_

- [x] 4. Implement Graph Construction Components
- [x] 4.1 (P) Implement Interface Registry for interface-to-implementation mapping
  - Build mapping from interface types to implementing provider and injector structs
  - Use go/types.Implements for both value and pointer receivers
  - Detect and report ambiguous implementations when multiple structs implement same interface
  - Return appropriate error when no implementation found for required interface
  - Support cross-package interface resolution
  - _Requirements: 3.5, 3.6, 3.7, 3.8_

- [x] 4.2 Implement Dependency Graph builder
  - Construct graph nodes from registered providers and injectors
  - Parse constructor signatures to extract parameter types as dependencies
  - Create dependency edges from parameter types to struct being constructed
  - Distinguish Inject structs (dependency struct fields) from Provide structs (local variables)
  - Resolve interface-typed parameters to implementing provider or injector
  - Exclude primitive and non-providable external types from graph edges
  - Validate all constructor parameters are resolvable and halt with error if not
  - _Requirements: 2.4, 3.1, 3.2, 3.3_

- [x] 4.3 Implement Topological Sort with cycle detection
  - Implement Kahn's algorithm for ordering dependency initialization
  - Apply alphabetical tie-breaking for deterministic output when multiple valid orderings exist
  - Detect circular dependencies via non-zero in-degree nodes after sort completion
  - Reconstruct and report full cycle path when cycle detected
  - Handle empty graph case gracefully
  - _Requirements: 4.1, 4.2, 4.3, 5.1, 5.2, 5.3_

- [x] 5. Implement App Analyzer
- [x] 5.1 Create AppAnalyzer as secondary analyzer for bootstrap generation
  - Define analyzer with proper name, documentation, and dependencies on inspect.Analyzer
  - Implement run function that detects App annotation and triggers bootstrap generation
  - Wait for all packages to be scanned via GlobalPackageTracker before proceeding
  - Retrieve all providers and injectors from global registries
  - Skip bootstrap generation when no App annotation present
  - _Requirements: 1.1, 1.3, 9.2, 9.3_

- [x] 5.2 Implement bootstrap generation orchestration in AppAnalyzer
  - Build dependency graph from retrieved providers and injectors
  - Execute topological sort and handle cycle errors
  - Detect existing bootstrap code and check if current via hash comparison
  - Generate new bootstrap when missing or update when outdated
  - Emit appropriate diagnostics with SuggestedFix
  - _Requirements: 6.1, 6.5, 6.6_

- [x] 6. Implement Bootstrap Code Generation
- [x] 6.1 Implement Bootstrap Generator for IIFE code synthesis
  - Generate immediately-invoked function expression returning anonymous struct
  - Place Inject structs as fields in returned dependency struct
  - Place Provide structs as local variables only within IIFE
  - Order constructor calls according to topological sort result
  - Use field names as intermediate variable names
  - _Requirements: 6.2, 7.1, 7.3_

- [x] 6.2 Implement field name derivation and naming rules
  - Convert type names to lowerCamelCase for field and variable names
  - Handle all-caps abbreviations appropriately
  - Resolve naming conflicts by appending numeric suffix
  - Ensure consistent naming across generated code
  - _Requirements: 7.2_

- [x] 6.3 Implement import collection for cross-package dependencies
  - Track external package paths required by dependency types
  - Generate appropriate import statements for bootstrap code
  - Handle package aliases when necessary
  - _Requirements: 7.4_

- [x] 6.4 Implement bootstrap code formatting and validation
  - Format generated code according to gofmt standards
  - Ensure generated code is valid and compilable Go
  - Apply consistent indentation and spacing
  - _Requirements: 6.4, 7.5_

- [x] 6.5 Implement idempotency detection via hash comparison
  - Compute deterministic hash from ordered type names and dependencies
  - Store hash as comment marker above dependency variable declaration
  - Compare hashes to determine if regeneration needed
  - Skip diagnostic emission when bootstrap is current
  - _Requirements: 6.5_

- [x] 6.6 Implement dependency reference detection in main function
  - Walk main function body to detect existing references to dependency variable
  - Distinguish between blank identifier assignments and actual usage
  - Determine whether to add _ = dependency statement to main
  - _Requirements: 6.3_

- [x] 7. Implement Diagnostic and Reporting Components
- [x] 7.1 (P) Extend DiagnosticEmitter with bootstrap-specific error methods
  - Add method for reporting multiple App annotation errors
  - Add method for reporting non-main App reference errors
  - Add method for reporting missing constructor errors
  - Add method for reporting circular dependency errors with full path
  - Add method for reporting ambiguous interface implementation errors
  - Add method for reporting unresolved interface dependency errors
  - Add method for reporting unresolvable constructor parameter errors
  - Include source position in all error messages
  - Provide descriptive messages explaining issue and resolution
  - _Requirements: 8.1, 8.2_

- [x] 7.2 (P) Extend SuggestedFixBuilder with bootstrap-specific fix methods
  - Add method for building bootstrap insertion fix
  - Add method for building bootstrap replacement fix for outdated code
  - Add method for building main function reference fix
  - Provide clear descriptions of what each fix accomplishes
  - _Requirements: 8.3_

- [x] 7.3 Implement bootstrap diagnostic emission in AppAnalyzer
  - Emit diagnostic with SuggestedFix for new bootstrap generation
  - Emit diagnostic with SuggestedFix for bootstrap updates
  - Skip emission when bootstrap is current (idempotent behavior)
  - _Requirements: 6.1, 6.6_

- [x] 8. Implement Multichecker CLI Integration
- [x] 8.1 Update CLI entry point to use multichecker with both analyzers
  - Replace singlechecker.Main with multichecker.Main
  - Register both DependencyAnalyzer and AppAnalyzer
  - Ensure proper analyzer ordering and execution
  - _Requirements: 9.1, 9.2_

- [x] 9. Implement Unit Tests
- [x] 9.1 (P) Add unit tests for ProviderRegistry
  - Test registration of provider information
  - Test retrieval of all providers
  - Test retrieval by type name including not-found case
  - Test thread safety under concurrent access
  - Test clear method for isolation
  - _Requirements: 2.3_

- [x] 9.2 (P) Add unit tests for InjectorRegistry
  - Test registration of injector information
  - Test retrieval of all injectors
  - Test retrieval by type name including not-found case
  - Test thread safety under concurrent access
  - Test clear method for isolation
  - _Requirements: 9.1_

- [x] 9.3 (P) Add unit tests for PackageTracker
  - Test marking packages as scanned
  - Test checking package scan status
  - Test waiting for all packages via channel synchronization
  - Test timeout handling to prevent deadlocks
  - Test clear method for isolation
  - _Requirements: 9.2, 9.3_

- [x] 9.4 (P) Add unit tests for AppDetector
  - Test detection of valid App annotation
  - Test validation with multiple App annotations
  - Test validation with non-main function reference
  - Test handling of aliased annotation imports
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 9.5 (P) Add unit tests for InterfaceRegistry
  - Test building interface-to-implementation mapping
  - Test single implementation resolution
  - Test ambiguous implementation detection
  - Test unresolved interface error
  - _Requirements: 3.5, 3.6, 3.7, 3.8_

- [x] 9.6 (P) Add unit tests for DependencyGraph
  - Test graph construction with providers and injectors
  - Test edge creation from constructor parameters
  - Test IsInject flag handling for Inject vs Provide distinction
  - Test exclusion of non-providable types
  - Test missing constructor validation
  - _Requirements: 2.4, 3.1, 3.2, 3.3, 3.4_

- [x] 9.7 (P) Add unit tests for TopologicalSort
  - Test ordering with various graph topologies
  - Test alphabetical tie-breaking for determinism
  - Test cycle detection and path reconstruction
  - Test empty graph handling
  - _Requirements: 4.1, 4.2, 4.3, 5.1, 5.2, 5.3_

- [x] 9.8 (P) Add unit tests for BootstrapGenerator
  - Test IIFE generation with anonymous struct
  - Test field name derivation including conflict handling
  - Test idempotency check via hash comparison
  - Test dependency reference detection in main function
  - Test Inject vs Provide struct placement in generated code
  - _Requirements: 6.2, 6.3, 6.5, 7.1, 7.2, 7.3_

- [ ] 10. Implement Integration Tests
- [x] 10.1 Add integration test for basic bootstrap generation
  - Test package with valid App annotation triggers bootstrap
  - Test Inject structs appear as dependency struct fields
  - Test Provide structs appear as local variables only
  - Verify generated code compiles and follows expected structure
  - _Requirements: 1.1, 6.1, 6.2, 7.1_

- [x] 10.2 Add integration test for no App annotation scenario
  - Test package without App annotation skips bootstrap generation
  - Verify no diagnostics emitted for packages without App marker
  - _Requirements: 1.3_

- [x] 10.3 Add integration test for App annotation validation errors
  - Test multiple App annotations produce error diagnostic
  - Test non-main function reference produces error diagnostic
  - Verify error messages include position and description
  - _Requirements: 1.2, 1.4, 8.1, 8.2_

- [x] 10.4 Add integration test for cross-package dependency discovery
  - Test providers and injectors registered from all packages via global registries
  - Test bootstrap includes dependencies from multiple packages
  - Verify import statements generated for external package types
  - _Requirements: 2.3, 7.4, 9.1_

- [ ] 10.5 Add integration test for circular dependency detection
  - Test cycle detected and reported with full path
  - Verify error message includes all types in cycle
  - _Requirements: 4.1, 4.3_

- [ ] 10.6 Add integration test for empty dependency graph
  - Test empty bootstrap generated when no providers or injectors
  - Verify no errors for valid but empty configuration
  - _Requirements: 5.3_

- [ ] 10.7 Add integration test for idempotent behavior
  - Test no diagnostic when bootstrap is current
  - Test hash comparison correctly identifies unchanged state
  - _Requirements: 6.5_

- [ ] 10.8 Add integration test for dependency already referenced in main
  - Test _ = dependency not added when dependency is already used
  - Verify detection works for field access patterns
  - _Requirements: 6.3_

- [ ] 10.9 Add integration test for interface dependency resolution
  - Test interface parameter resolved to implementing provider or injector
  - Verify correct type passed to constructor in generated code
  - _Requirements: 3.5_

- [ ] 10.10 Add integration test for ambiguous interface implementation
  - Test error reported when multiple structs implement same interface
  - Verify error message lists all conflicting implementations
  - _Requirements: 3.6_

- [ ] 10.11 Add integration test for cross-package interface resolution
  - Test interface defined in one package with implementation in another
  - Verify resolution works via global registry
  - _Requirements: 3.8_

- [ ] 10.12 Add integration test for unresolved interface dependency
  - Test error reported when interface parameter has no implementation
  - Verify error message suggests adding annotation.Provide
  - _Requirements: 3.7_

- [x] 10.13 Add integration test for single-pass constructor and bootstrap generation
  - Test both constructor and bootstrap generated in single go vet -fix invocation
  - Verify channel synchronization ensures all packages scanned
  - _Requirements: 9.2, 9.3_

- [ ] 10.14 Add integration test for module-wide discovery
  - Test all providers and injectors discovered without explicit imports in main
  - Verify bootstrap includes all annotated structs from module
  - _Requirements: 2.3, 9.1_

- [ ] 10.15 Add integration test for unresolvable constructor parameter
  - Test error reported when constructor parameter cannot be resolved
  - Verify error message identifies the unresolvable type
  - _Requirements: 3.3_

- [ ] 10.16 Add integration test for bootstrap update when outdated
  - Test diagnostic emitted with SuggestedFix for outdated bootstrap
  - Verify updated bootstrap reflects current dependency graph
  - _Requirements: 6.6_

- [ ] 10.17 Add integration test for missing constructor error
  - Test error reported when Provide struct lacks constructor
  - Verify error message identifies the struct requiring constructor
  - _Requirements: 2.4_

- [ ] 11. Create Golden File Test Fixtures
- [ ] 11.1 Create test fixtures for basic single-package bootstrap
  - Create main.go with App annotation
  - Create service package with Inject struct and constructor
  - Create expected golden file with generated bootstrap code
  - _Requirements: 6.1, 7.1_

- [ ] 11.2 Create test fixtures for multi-type cross-package bootstrap
  - Create main package with App annotation
  - Create repository package with Provide structs
  - Create service package with Inject structs depending on repositories
  - Create expected golden file showing proper field vs local variable placement
  - _Requirements: 6.2, 7.3_

- [ ] 11.3 Create test fixtures for interface dependency scenario
  - Create domain package with interface definition
  - Create repository package with Provide struct implementing interface
  - Create service package with Inject struct depending on interface
  - Create expected golden file showing interface resolution
  - _Requirements: 3.5_

- [ ] 11.4 Create test fixtures for dependency already used scenario
  - Create main.go with existing dependency.field usage
  - Verify golden file does not add _ = dependency statement
  - _Requirements: 6.3_

- [ ] 11.5 Create test fixtures for error cases
  - Create circular dependency scenario
  - Create multiple App annotation scenario
  - Create ambiguous interface implementation scenario
  - Create unresolved interface scenario
  - Create unresolvable parameter scenario
  - _Requirements: 1.2, 1.4, 3.6, 3.7, 4.1_
