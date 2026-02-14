# Implementation Plan

- [x] 1. Extend FieldAnalyzer with struct tag parsing
- [x] 1.1 Add struct tag metadata fields to the FieldInfo value object
  - Extend the field information structure with a named dependency string and an excluded boolean to carry struct tag state through the analysis pipeline
  - When a field has no `braider` tag, both new fields must retain zero-value defaults to preserve existing behavior
  - Ensure the named dependency and excluded states are mutually exclusive
  - _Requirements: 3.3, 3.4_

- [x] 1.2 Implement the struct tag parsing helper on the field analyzer
  - Extract the raw tag string from the AST field tag literal, strip surrounding backticks via unquoting, and parse using Go's standard struct tag lookup for the `braider` key
  - When the lookup returns an empty string with presence confirmed, record an invalid-tag state for the caller to report
  - When the tag value is `"-"`, set the excluded flag; otherwise populate the named dependency with the parsed value
  - Handle edge cases: missing tag literal, unquoting failure (treat as untagged), and other struct tag keys on the same field (ignore them)
  - _Requirements: 3.1, 3.2, 3.4_

- [x] 1.3 Integrate struct tag parsing into the field analysis loop
  - Call the tag parsing helper for each field during field analysis, populating the new metadata on each returned field info
  - Propagate any invalid-tag diagnostic information so the caller (DependencyAnalyzeRunner) can emit the appropriate error
  - Verify that the inject annotation embedding field continues to be skipped before tag parsing
  - _Requirements: 1.1, 1.5, 1.6, 2.1, 3.1, 3.3_

- [x] 1.4 Add unit tests for struct tag parsing and field analysis
  - Test parsing of `braider:"someName"`, `braider:"-"`, `braider:""`, absent tag, and multi-tag fields (e.g., `json:"x" braider:"name"`)
  - Verify FieldInfo values: named dependency populated for named tags, excluded flag set for exclusion tags, defaults for untagged fields
  - Test that the invalid-tag state is correctly signaled for empty tag values
  - _Requirements: 1.1, 1.5, 2.1, 3.1, 3.2, 3.3, 3.4_

- [ ] 2. Add diagnostic emission methods for struct tag errors
- [ ] 2.1 (P) Add an invalid struct tag diagnostic method to the diagnostic emitter
  - Extend the diagnostic emitter interface and implementation with a method that reports an invalid braider struct tag value, identifying the field name and source position
  - Follow the existing diagnostic emission pattern (format string with field name)
  - _Requirements: 3.2_

- [ ] 2.2 (P) Add a struct tag conflict diagnostic method to the diagnostic emitter
  - Extend the diagnostic emitter interface and implementation with a method that reports a braider struct tag conflict with an existing constructor under the WithoutConstructor option, including the field name and a human-readable reason
  - Follow the existing diagnostic emission pattern
  - _Requirements: 4.3_

- [ ] 2.3 Add unit tests for the new diagnostic methods
  - Verify that both new methods produce correctly formatted diagnostic messages with the expected positions and content
  - Cannot run in parallel with 2.1/2.2 because the methods must exist first
  - _Requirements: 3.2, 4.3_

- [ ] 3. Update DependencyAnalyzeRunner Phase 1 for tag-aware constructor generation
- [ ] 3.1 Filter excluded fields and build the named dependency map before constructor generation
  - After calling the field analyzer, remove fields marked as excluded from the list passed to the constructor generator
  - Build a mapping from field name to named dependency value for fields with a non-empty named dependency
  - When the field analyzer signals an invalid tag, emit the invalid struct tag diagnostic and treat the field as a standard dependency
  - _Requirements: 1.1, 1.3, 1.6, 2.1, 2.2, 3.2_

- [ ] 3.2 Switch constructor generation to the tag-aware path
  - Replace the call to the basic constructor generation method with the named-dependency-aware generation method, passing the filtered field list and the dependency names map
  - When the dependency names map is empty, the behavior must be identical to the current basic generation path
  - Update the existing-constructor staleness check to compare against filtered fields rather than all fields
  - _Requirements: 1.3, 2.2, 2.4_

- [ ] 3.3 Add unit tests for Phase 1 tag-aware constructor generation
  - Test that excluded fields are removed before constructor generation
  - Test that the dependency names map is correctly built from struct tag metadata
  - Test that constructor generation with no tags produces identical output to the current behavior
  - Test that all fields excluded results in a zero-parameter constructor
  - _Requirements: 1.3, 1.6, 2.2, 2.4_

- [ ] 4. Update DependencyAnalyzeRunner Phase 3 for tag-aware registry registration
- [ ] 4.1 Construct composite dependency keys for named fields during registration
  - When building the dependency list for registry registration, use the composite key format (fully qualified type plus name separator and tag value) for fields with a named dependency
  - Skip excluded fields entirely when constructing the dependency list
  - Preserve the existing plain type name format for fields without a braider tag
  - _Requirements: 1.1, 1.4, 2.3_

- [ ] 4.2 Implement WithoutConstructor conflict validation
  - When the injectable struct uses the WithoutConstructor option, cross-check struct tag metadata against the existing constructor's parameter types
  - Emit a conflict diagnostic when a field with the exclusion tag matches a constructor parameter type
  - Emit a conflict diagnostic when a field with a named dependency tag does not match any constructor parameter type
  - _Requirements: 4.1, 4.2, 4.3_

- [ ] 4.3 Add unit tests for Phase 3 tag-aware registration and validation
  - Test that composite dependency keys appear in registered injector info for named fields
  - Test that excluded fields are absent from the dependency list
  - Test that WithoutConstructor conflict detection emits the correct diagnostics
  - Test that type-level Named[N] and field-level braider tag coexist independently
  - _Requirements: 1.1, 1.4, 2.3, 4.1, 4.2, 4.3_

- [ ] 5. Add integration tests for constructor generation with struct tags
- [ ] 5.1 (P) Add a constructorgen test case for named dependency via struct tag
  - Create a test fixture with an Injectable struct that has a field annotated with a braider named tag
  - Verify the generated constructor uses the named dependency as the parameter name via a golden file
  - _Requirements: 1.3_

- [ ] 5.2 (P) Add a constructorgen test case for field exclusion via struct tag
  - Create a test fixture with an Injectable struct that has a field annotated with the exclusion tag
  - Verify the generated constructor omits the excluded field's parameter via a golden file
  - _Requirements: 2.2_

- [ ] 6. Add integration tests for bootstrap generation with struct tags
- [ ] 6.1 Add a bootstrapgen test case for named dependency wiring
  - Create a test fixture with an Injectable struct using a braider named tag, a provider registered with a matching Named option, and an App annotation in the main package
  - Verify the generated bootstrap code passes the named dependency variable to the constructor
  - Verify topological sort includes the named dependency in the initialization order
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 5.1, 5.2_

- [ ] 6.2 Add a bootstrapgen test case for field exclusion in bootstrap
  - Create a test fixture with an Injectable struct using the exclusion tag and an App annotation
  - Verify the generated bootstrap code omits the excluded dependency from the wiring
  - Verify the dependency graph does not contain an edge for the excluded field
  - _Requirements: 2.1, 2.2, 2.3, 5.1_

- [ ] 6.3 Add a bootstrapgen test case for mixed struct tags
  - Create a test fixture with an Injectable struct containing one named-tagged field, one exclusion-tagged field, and one untagged field
  - Verify the generated constructor and bootstrap code handle each field according to its tag state
  - _Requirements: 1.1, 1.6, 2.1, 2.3_

- [ ] 6.4 Add a bootstrapgen test case for all fields excluded
  - Create a test fixture where every non-annotation field has the exclusion tag
  - Verify the generated constructor has zero parameters and the bootstrap code uses the zero-arg constructor
  - _Requirements: 2.4_

- [ ] 6.5 Add a bootstrapgen error test case for empty struct tag
  - Create a test fixture with a field annotated with an empty braider tag value
  - Verify the analyzer emits the expected invalid-tag diagnostic
  - _Requirements: 3.2_

- [ ] 6.6 Add a bootstrapgen error test case for WithoutConstructor conflict
  - Create a test fixture with an Injectable struct using WithoutConstructor and a braider exclusion tag on a field accepted by the existing constructor
  - Verify the analyzer emits the expected conflict diagnostic
  - _Requirements: 4.3_

- [ ] 7. Add integration tests for idempotent behavior with struct tags
- [ ] 7.1 (P) Add an idempotent test case for struct tag stability
  - Create a test fixture where a struct with a braider named tag has already been analyzed (bootstrap hash in place)
  - Verify that re-running the analyzer produces no diagnostic (hash matches, no regeneration)
  - _Requirements: 6.4_

- [ ] 7.2 (P) Add an outdated test case for struct tag changes
  - Create a test fixture where the existing bootstrap hash no longer matches after a struct tag modification
  - Verify the analyzer detects the hash mismatch and triggers bootstrap regeneration
  - _Requirements: 6.1, 6.2, 6.3_

- [ ] 8. Validate type compatibility and cross-cutting requirements
- [ ] 8.1 (P) Add a bootstrapgen test case exercising all supported field types with named tags
  - Create a test fixture containing concrete, pointer, and interface fields each with braider named tags and matching named providers
  - Verify the generated constructor and bootstrap correctly wire all three field types
  - _Requirements: 1.5_

- [ ] 8.2 Verify hash computation captures struct tag changes through dependency list differences
  - Confirm that adding a braider named tag changes the dependency list (plain type name becomes composite key), causing a hash change
  - Confirm that adding an exclusion tag removes a dependency from the list, causing a hash change
  - Confirm that removing a tag reverts the dependency to the plain type name, causing a hash change
  - This verification is covered by the outdated test case (7.2) and the idempotent test case (7.1); add explicit assertions if not already covered
  - _Requirements: 5.3, 6.1, 6.2, 6.3, 6.4_
