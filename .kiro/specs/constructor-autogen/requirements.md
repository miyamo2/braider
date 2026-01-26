# Requirements Document

## Introduction

This document specifies the requirements for the **Constructor Auto-Generation** feature of braider. This feature enables automatic generation of constructor functions for Go structs with injectable dependencies, leveraging the `analysis.SuggestedFix` mechanism to propose code fixes that can be applied via `go vet -fix`.

Constructor auto-generation is a foundational capability that allows developers to define struct types with dependencies and have braider automatically generate idiomatic Go constructor functions (`New*` functions) that initialize all dependencies.

## Requirements

### Requirement 1: Struct Detection for Constructor Generation

**Objective:** As a Go developer, I want braider to detect structs that need constructor generation, so that I can automatically generate constructors for my dependency-bearing types.

#### Acceptance Criteria

1. When a struct type is annotated with a braider marker comment, the Analyzer shall identify the struct as a candidate for constructor generation.
2. When a struct has exported fields with dependency types, the Analyzer shall recognize those fields as constructor parameters.
3. When a struct has unexported fields with dependency types, the Analyzer shall recognize those fields as constructor parameters.
4. If a struct has no injectable dependencies, the Analyzer shall not propose constructor generation for that struct.
5. The Analyzer shall support detection of structs across all Go files in the analyzed package.

### Requirement 2: Constructor Code Generation

**Objective:** As a Go developer, I want braider to generate idiomatic constructor functions, so that I can initialize my structs with all required dependencies.

#### Acceptance Criteria

1. When a constructor-candidate struct is detected, the Analyzer shall generate a `New<StructName>` function signature.
2. The Analyzer shall generate constructor parameters matching the struct's dependency fields in declaration order.
3. The Analyzer shall generate constructor body that assigns each parameter to the corresponding struct field.
4. The Analyzer shall generate a constructor that returns a pointer to the initialized struct.
5. If the struct has interface-typed dependencies, the Analyzer shall use the interface types as constructor parameter types.
6. If the struct has concrete-typed dependencies, the Analyzer shall use the concrete types as constructor parameter types.

### Requirement 3: SuggestedFix Integration

**Objective:** As a Go developer, I want constructor generation to be proposed as a suggested fix, so that I can review and apply the changes using standard Go tooling.

#### Acceptance Criteria

1. When a struct requires a constructor, the Analyzer shall emit a diagnostic with a SuggestedFix containing the generated constructor code.
2. The Analyzer shall include the generated constructor code as a TextEdit in the SuggestedFix.
3. When `go vet -fix` is executed, the Analyzer shall enable automatic application of the suggested constructor code.
4. The Analyzer shall position the generated constructor immediately after the struct type definition.
5. If the constructor already exists for a struct, the Analyzer shall not propose a duplicate constructor.

### Requirement 4: Code Formatting and Style

**Objective:** As a Go developer, I want generated constructors to follow Go formatting conventions, so that the generated code integrates seamlessly with my codebase.

#### Acceptance Criteria

1. The Analyzer shall generate constructor code that passes `gofmt` without modifications.
2. The Analyzer shall use proper indentation and spacing in generated code.
3. The Analyzer shall generate parameter names derived from field names using Go naming conventions.
4. If a parameter name conflicts with a Go keyword or builtin, the Analyzer shall use an alternative valid identifier.
5. The Analyzer shall include a blank line between the struct definition and the generated constructor.

### Requirement 5: Error Reporting and Diagnostics

**Objective:** As a Go developer, I want clear error messages when constructor generation encounters issues, so that I can understand and resolve problems.

#### Acceptance Criteria

1. If a struct has circular dependencies, the Analyzer shall report a diagnostic with a clear error message.
2. The Analyzer shall include source position information in all diagnostic messages.
3. If constructor generation fails, the Analyzer shall report the reason for failure in the diagnostic message.
4. The Analyzer shall provide actionable guidance in error messages when possible.

### Requirement 6: Marker Comment Syntax

**Objective:** As a Go developer, I want a clear and simple syntax for marking structs for constructor generation, so that I can easily opt-in specific types.

#### Acceptance Criteria

1. The Analyzer shall recognize a specific comment directive format to mark structs for constructor generation.
2. When a struct has the marker comment immediately preceding its type declaration, the Analyzer shall process that struct for constructor generation.
3. If the marker comment is malformed, the Analyzer shall report a diagnostic explaining the expected format.
4. The Analyzer shall ignore structs without the marker comment.
