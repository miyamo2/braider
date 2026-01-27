# Implementation Plan

## Task Overview

This plan implements the Constructor Auto-Generation feature for braider, enabling automatic generation of constructor functions for Go structs with embedded `annotation.Inject`. The implementation follows a pipeline architecture: Detection -> Generation -> Reporting.

---

## Tasks

- [x] 1. Implement Inject Annotation Detection
- [x] 1.1 (P) Create the inject annotation detector capability
  - Detect embedded `annotation.Inject` fields in struct types
  - Verify the embedded type comes from the correct braider annotation package path
  - Handle aliased imports of the annotation package
  - Return the embedded Inject field when found for downstream filtering
  - _Requirements: 1.1, 6.1, 6.2, 6.4_

- [x] 1.2 Unit tests for inject annotation detection
  - Test detection of standard `annotation.Inject` embedding
  - Test detection with aliased import names
  - Test rejection of named (non-embedded) Inject fields
  - Test rejection of Inject types from different packages
  - _Requirements: 1.1, 6.1, 6.2, 6.4_

- [x] 2. Implement Struct Detection and Candidate Collection
- [x] 2.1 Create the struct detector capability
  - Traverse AST to find type declarations containing struct types
  - Filter structs that embed `annotation.Inject` using the inject detector
  - Collect constructor candidates with their AST context (TypeSpec, GenDecl, StructType)
  - Support detection across all Go files in the analyzed package
  - Depends on: Task 1.1 (inject detector)
  - _Requirements: 1.1, 1.5_

- [x] 2.2 Implement existing constructor detection
  - Find existing `New<StructName>` functions in the package
  - Verify the function returns a pointer to the struct type
  - Track existing constructors for replacement rather than duplicate insertion
  - _Requirements: 3.5_

- [x] 2.3 Unit tests for struct detection
  - Test detection of structs with Inject embedding
  - Test skipping of structs without Inject embedding
  - Test multi-file package analysis
  - Test existing constructor detection and tracking
  - _Requirements: 1.1, 1.5, 3.5_

- [x] 3. Implement Field Analysis
- [x] 3.1 (P) Create the field analyzer capability
  - Extract field list from struct types
  - Exclude the embedded `annotation.Inject` field from constructor parameters
  - Resolve field types using the type checker information
  - Classify types as interface vs concrete for parameter type selection
  - Preserve field declaration order for parameter ordering
  - _Requirements: 1.2, 1.3, 1.4, 2.5, 2.6_

- [x] 3.2 Unit tests for field analysis
  - Test extraction of exported and unexported fields
  - Test exclusion of Inject embedding from results
  - Test interface type classification
  - Test concrete type classification
  - Test structs with no injectable fields (Inject-only)
  - _Requirements: 1.2, 1.3, 1.4, 2.5, 2.6_

- [x] 4. Implement Constructor Code Generation
- [x] 4.1 Create the constructor generator capability
  - Generate `New<StructName>` function name from struct name
  - Build parameter list matching field order and resolved types
  - Generate function body with struct literal and field assignments
  - Return a pointer to the initialized struct
  - Depends on: Task 3.1 (field analyzer)
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [x] 4.2 (P) Implement parameter naming with keyword avoidance
  - Derive parameter names from field names (lowercase first letter)
  - Detect conflicts with Go keywords and append suffix
  - Detect conflicts with Go builtins and use alternative naming
  - _Requirements: 4.3, 4.4_

- [x] 4.3 Unit tests for constructor generation
  - Test function name generation from struct name
  - Test parameter ordering matches field declaration order
  - Test parameter types match field types (interface and concrete)
  - Test keyword conflict resolution in parameter names
  - Test builtin conflict resolution in parameter names
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 4.3, 4.4_

- [x] 5. Implement Code Formatting
- [x] 5.1 (P) Create the code formatter capability
  - Apply `go/format.Source` to generated constructor code
  - Ensure output passes gofmt validation without modifications
  - Handle formatting errors gracefully
  - _Requirements: 4.1, 4.2_

- [x] 5.2 Unit tests for code formatting
  - Test formatting of valid constructor code
  - Test proper indentation and spacing in output
  - Test error handling for malformed input
  - _Requirements: 4.1, 4.2_

- [x] 6. Implement SuggestedFix Building
- [x] 6.1 Create the suggested fix builder capability
  - Calculate insertion position after struct definition for new constructors
  - Calculate replacement range for existing constructors
  - Include blank line separator between struct and constructor
  - Build TextEdit with correct byte positions for insertion or replacement
  - Handle doc comments in replacement range calculation
  - Depends on: Task 5.1 (code formatter)
  - _Requirements: 3.2, 3.4, 3.5, 4.5_

- [x] 6.2 Unit tests for suggested fix building
  - Test insertion position calculation after struct
  - Test replacement range calculation for existing constructors
  - Test blank line inclusion in TextEdit
  - Test handling of constructors with doc comments
  - _Requirements: 3.2, 3.4, 3.5, 4.5_

- [x] 7. Implement Diagnostic Emission
- [x] 7.1 (P) Create the diagnostic emitter capability
  - Emit diagnostics for constructor generation with SuggestedFix
  - Include source position information in all diagnostic messages
  - Format messages to be clear and actionable
  - _Requirements: 3.1, 5.2, 5.4_

- [x] 7.2 (P) Implement error diagnostics
  - Report circular dependency errors with cycle path
  - Report constructor generation failures with reason
  - Provide actionable guidance in error messages
  - _Requirements: 5.1, 5.3, 5.4_

- [x] 7.3 Unit tests for diagnostic emission
  - Test diagnostic message format for constructor availability
  - Test position information inclusion
  - Test circular dependency error reporting
  - Test generation failure error reporting
  - _Requirements: 3.1, 5.1, 5.2, 5.3, 5.4_

- [x] 8. Integrate Analysis Pipeline
- [x] 8.1 Wire all components into the analyzer run function
  - Integrate inject detector with struct detector
  - Connect field analyzer to constructor generator
  - Wire code formatter to suggested fix builder
  - Connect diagnostic emitter to the analysis pass
  - Orchestrate the detection-generation-reporting pipeline
  - Depends on: Tasks 1.1, 2.1, 2.2, 3.1, 4.1, 4.2, 5.1, 6.1, 7.1, 7.2
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 3.1, 3.2, 3.3, 3.4, 3.5, 4.1, 4.2, 4.3, 4.4, 4.5, 5.1, 5.2, 5.3, 5.4, 6.1, 6.2, 6.4_

- [x] 8.2 Verify go vet -fix integration
  - Ensure SuggestedFix applies correctly via `go vet -fix`
  - Verify generated code is written to source files
  - _Requirements: 3.3_

- [x] 9. Integration and Golden File Tests
- [x] 9.1 Create golden file test fixtures
  - Create simple struct test case with single dependency
  - Create multi-field struct test case
  - Create interface-typed field test case
  - Create mixed pointer and value type test case
  - Create aliased import test case
  - _Requirements: 1.1, 1.2, 1.3, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 6.1, 6.2_

- [x] 9.2 Create existing constructor replacement test fixtures
  - Create test case for constructor replacement with additional fields
  - Create test case for constructor replacement with removed fields
  - Verify TextEdit correctly handles line count differences
  - _Requirements: 3.5_

- [x] 9.3 Create negative test cases
  - Create struct without Inject embedding (should be skipped)
  - Create struct with named Inject field (should be skipped)
  - Create struct with Inject-only fields (should be skipped)
  - Create struct with wrong package Inject (should be skipped)
  - _Requirements: 1.4, 6.4_

- [x] 9.4 Run integration tests with analysistest
  - Execute all golden file tests with RunWithSuggestedFixes
  - Verify diagnostic messages match expected format
  - Verify generated constructor code matches golden files
  - _Requirements: 3.1, 3.2, 3.3, 4.1, 4.2, 4.5_

---

## Requirements Coverage

| Requirement | Tasks |
|-------------|-------|
| 1.1 | 1.1, 1.2, 2.1, 2.3, 8.1, 9.1 |
| 1.2 | 3.1, 3.2, 8.1, 9.1 |
| 1.3 | 3.1, 3.2, 8.1, 9.1 |
| 1.4 | 3.1, 3.2, 8.1, 9.3 |
| 1.5 | 2.1, 2.3, 8.1 |
| 2.1 | 4.1, 4.3, 8.1, 9.1 |
| 2.2 | 4.1, 4.3, 8.1, 9.1 |
| 2.3 | 4.1, 4.3, 8.1, 9.1 |
| 2.4 | 4.1, 4.3, 8.1, 9.1 |
| 2.5 | 3.1, 3.2, 8.1, 9.1 |
| 2.6 | 3.1, 3.2, 8.1, 9.1 |
| 3.1 | 7.1, 7.3, 8.1, 9.4 |
| 3.2 | 6.1, 6.2, 8.1, 9.4 |
| 3.3 | 8.1, 8.2, 9.4 |
| 3.4 | 6.1, 6.2, 8.1 |
| 3.5 | 2.2, 2.3, 6.1, 6.2, 8.1, 9.2 |
| 4.1 | 5.1, 5.2, 8.1, 9.4 |
| 4.2 | 5.1, 5.2, 8.1, 9.4 |
| 4.3 | 4.2, 4.3, 8.1 |
| 4.4 | 4.2, 4.3, 8.1 |
| 4.5 | 6.1, 6.2, 8.1, 9.4 |
| 5.1 | 7.2, 7.3, 8.1 |
| 5.2 | 7.1, 7.3, 8.1 |
| 5.3 | 7.2, 7.3, 8.1 |
| 5.4 | 7.1, 7.2, 7.3, 8.1 |
| 6.1 | 1.1, 1.2, 8.1, 9.1 |
| 6.2 | 1.1, 1.2, 8.1, 9.1 |
| 6.4 | 1.1, 1.2, 8.1, 9.3 |
