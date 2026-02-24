package report

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
	"golang.org/x/tools/go/analysis"
)

// Diagnostic category constants used in analysis.Diagnostic.Category.
const (
	// CategoryConstructorGeneration is for constructor generation and update diagnostics.
	CategoryConstructorGeneration = "braider:constructor-generation"
	// CategoryDependencyResolution is for dependency graph construction and cycle detection diagnostics.
	CategoryDependencyResolution = "braider:dependency-resolution"
	// CategoryDependencyRegistration is for duplicate dependency registration diagnostics.
	CategoryDependencyRegistration = "braider:dependency-registration"
	// CategoryAppValidation is for annotation.App validation diagnostics.
	CategoryAppValidation = "braider:app-validation"
	// CategoryBootstrapGeneration is for bootstrap code generation diagnostics.
	CategoryBootstrapGeneration = "braider:bootstrap-generation"
	// CategoryPackageLoading is for package loading and wait timeout diagnostics.
	CategoryPackageLoading = "braider:package-loading"
	// CategoryOptionValidation is for annotation option constraint violation diagnostics.
	// Maps to SeverityCritical in phasedchecker, aborting the pipeline.
	CategoryOptionValidation = "braider:option-validation"
	// CategoryExpressionValidation is for unsupported expression diagnostics.
	// Maps to SeverityCritical in phasedchecker, aborting the pipeline.
	CategoryExpressionValidation = "braider:expression-validation"
	// CategoryStructTagValidation is for braider struct tag validation diagnostics.
	CategoryStructTagValidation = "braider:struct-tag-validation"
	// CategoryContainerValidation is for container type parameter and field diagnostics.
	CategoryContainerValidation = "braider:container-validation"
)

// Reporter is an interface for reporting diagnostics.
// This matches the analysis.Pass.Report method signature.
type Reporter interface {
	Report(analysis.Diagnostic)
}

// DiagnosticEmitter emits diagnostics to the analysis pass.
type DiagnosticEmitter interface {
	// EmitConstructorFix reports a diagnostic with constructor SuggestedFix.
	EmitConstructorFix(
		reporter Reporter,
		pos token.Pos,
		structName string,
		fix analysis.SuggestedFix,
	)

	// EmitExistingConstructorFix reports a diagnostic for replacing an existing constructor.
	EmitExistingConstructorFix(
		reporter Reporter,
		pos token.Pos,
		structName string,
		fix analysis.SuggestedFix,
	)

	// EmitCircularDependency reports a circular dependency error.
	EmitCircularDependency(reporter Reporter, pos token.Pos, cycle []string)

	// EmitGenerationError reports a constructor generation failure.
	EmitGenerationError(reporter Reporter, pos token.Pos, structName string, reason string)

	// EmitNonMainAppError reports App referencing non-main function.
	EmitNonMainAppError(reporter Reporter, pos token.Pos, funcName string)

	// EmitBootstrapFix reports a diagnostic for missing bootstrap code.
	EmitBootstrapFix(reporter Reporter, pos token.Pos, fix analysis.SuggestedFix)

	// EmitBootstrapUpdateFix reports a diagnostic for outdated bootstrap code.
	EmitBootstrapUpdateFix(reporter Reporter, pos token.Pos, fix analysis.SuggestedFix)

	// EmitDuplicateAppWarning reports duplicate annotation.App.
	EmitDuplicateAppWarning(reporter Reporter, pos token.Pos)

	// EmitPackageLoadWarning reports a warning when package loading fails.
	EmitPackageLoadWarning(reporter Reporter, pos token.Pos, reason string)

	// EmitPackageWaitWarning reports a warning when waiting for packages times out.
	EmitPackageWaitWarning(reporter Reporter, pos token.Pos, reason string)

	// EmitGraphBuildError reports a dependency graph construction error.
	EmitGraphBuildError(reporter Reporter, pos token.Pos, reason string)

	// EmitDuplicateNamedDependencyWarning reports duplicate (TypeName, Name) pairs (non-fatal).
	EmitDuplicateNamedDependencyWarning(
		reporter Reporter, pos token.Pos, typeName string, name string, location1 string, location2 string,
	)

	// EmitOptionValidationError reports a fatal option validation error (constraint violation, interface mismatch, non-literal Namer).
	EmitOptionValidationError(reporter Reporter, pos token.Pos, reason string)

	// EmitUnsupportedVariableExpression reports an unsupported Variable argument expression error.
	EmitUnsupportedVariableExpression(reporter Reporter, pos token.Pos, reason string)

	// EmitInvalidStructTagError reports an invalid braider struct tag value (braider:"").
	EmitInvalidStructTagError(reporter Reporter, pos token.Pos, fieldName string)

	// EmitStructTagConflictError reports a braider struct tag conflict with WithoutConstructor.
	EmitStructTagConflictError(reporter Reporter, pos token.Pos, fieldName string, reason string)

	// EmitContainerTypeError reports a non-struct container type parameter.
	EmitContainerTypeError(reporter Reporter, pos token.Pos, typeName string)

	// EmitContainerFieldError reports an unresolvable container field.
	EmitContainerFieldError(reporter Reporter, pos token.Pos, fieldName string, fieldType string, reason string)
}

// diagnosticEmitter is the default implementation of DiagnosticEmitter.
type diagnosticEmitter struct {
	annotation.Injectable[inject.Typed[DiagnosticEmitter]]
}

// NewDiagnosticEmitter is a constructor for diagnosticEmitter.
//
// Generated by braider. DO NOT EDIT.
func NewDiagnosticEmitter() *diagnosticEmitter {
	return &diagnosticEmitter{}
}

// EmitConstructorFix reports a diagnostic with constructor SuggestedFix.
func (e *diagnosticEmitter) EmitConstructorFix(
	reporter Reporter,
	pos token.Pos,
	structName string,
	fix analysis.SuggestedFix,
) {
	reporter.Report(
		analysis.Diagnostic{
			Pos:            pos,
			Category:       CategoryConstructorGeneration,
			Message:        fmt.Sprintf("missing constructor for %s", structName),
			SuggestedFixes: []analysis.SuggestedFix{fix},
		},
	)
}

// EmitExistingConstructorFix reports a diagnostic for replacing an existing constructor.
func (e *diagnosticEmitter) EmitExistingConstructorFix(
	reporter Reporter,
	pos token.Pos,
	structName string,
	fix analysis.SuggestedFix,
) {
	reporter.Report(
		analysis.Diagnostic{
			Pos:            pos,
			Category:       CategoryConstructorGeneration,
			Message:        fmt.Sprintf("outdated constructor for %s", structName),
			SuggestedFixes: []analysis.SuggestedFix{fix},
		},
	)
}

// EmitCircularDependency reports a circular dependency error.
func (e *diagnosticEmitter) EmitCircularDependency(reporter Reporter, pos token.Pos, cycle []string) {
	cyclePath := strings.Join(cycle, " -> ")
	reporter.Report(
		analysis.Diagnostic{
			Pos:      pos,
			Category: CategoryDependencyResolution,
			Message:  fmt.Sprintf("circular dependency detected: %s", cyclePath),
		},
	)
}

// EmitGenerationError reports a constructor generation failure.
func (e *diagnosticEmitter) EmitGenerationError(reporter Reporter, pos token.Pos, structName string, reason string) {
	reporter.Report(
		analysis.Diagnostic{
			Pos:      pos,
			Category: CategoryConstructorGeneration,
			Message:  fmt.Sprintf("failed to generate constructor for %s: %s", structName, reason),
		},
	)
}

// EmitNonMainAppError reports App referencing non-main function.
func (e *diagnosticEmitter) EmitNonMainAppError(reporter Reporter, pos token.Pos, funcName string) {
	reporter.Report(
		analysis.Diagnostic{
			Pos:      pos,
			Category: CategoryAppValidation,
			Message:  fmt.Sprintf("annotation.App must reference main function, got %s", funcName),
		},
	)
}

// EmitBootstrapFix reports a diagnostic for missing bootstrap code.
func (e *diagnosticEmitter) EmitBootstrapFix(reporter Reporter, pos token.Pos, fix analysis.SuggestedFix) {
	reporter.Report(
		analysis.Diagnostic{
			Pos:            pos,
			Category:       CategoryBootstrapGeneration,
			Message:        "bootstrap code is missing",
			SuggestedFixes: []analysis.SuggestedFix{fix},
		},
	)
}

// EmitBootstrapUpdateFix reports a diagnostic for outdated bootstrap code.
func (e *diagnosticEmitter) EmitBootstrapUpdateFix(reporter Reporter, pos token.Pos, fix analysis.SuggestedFix) {
	reporter.Report(
		analysis.Diagnostic{
			Pos:            pos,
			Category:       CategoryBootstrapGeneration,
			Message:        "bootstrap code is outdated",
			SuggestedFixes: []analysis.SuggestedFix{fix},
		},
	)
}

// EmitDuplicateAppWarning reports duplicate annotation.App in the same package.
func (e *diagnosticEmitter) EmitDuplicateAppWarning(reporter Reporter, pos token.Pos) {
	reporter.Report(
		analysis.Diagnostic{
			Pos:      pos,
			Category: CategoryAppValidation,
			Message:  "another annotation.App in the same package is being applied",
		},
	)
}

// EmitPackageLoadWarning reports a warning when package loading fails.
func (e *diagnosticEmitter) EmitPackageLoadWarning(reporter Reporter, pos token.Pos, reason string) {
	reporter.Report(
		analysis.Diagnostic{
			Pos:      pos,
			Category: CategoryPackageLoading,
			Message:  fmt.Sprintf("warning: failed to load module packages: %s (bootstrap may be incomplete)", reason),
		},
	)
}

// EmitPackageWaitWarning reports a warning when waiting for packages times out.
func (e *diagnosticEmitter) EmitPackageWaitWarning(reporter Reporter, pos token.Pos, reason string) {
	reporter.Report(
		analysis.Diagnostic{
			Pos:      pos,
			Category: CategoryPackageLoading,
			Message: fmt.Sprintf(
				"warning: timeout waiting for package analysis: %s (bootstrap may be incomplete)",
				reason,
			),
		},
	)
}

// EmitGraphBuildError reports a dependency graph construction error.
func (e *diagnosticEmitter) EmitGraphBuildError(reporter Reporter, pos token.Pos, reason string) {
	reporter.Report(
		analysis.Diagnostic{
			Pos:      pos,
			Category: CategoryDependencyResolution,
			Message:  fmt.Sprintf("failed to build dependency graph: %s", reason),
		},
	)
}

// EmitDuplicateNamedDependencyWarning reports duplicate (TypeName, Name) pairs (non-fatal correlation error).
func (e *diagnosticEmitter) EmitDuplicateNamedDependencyWarning(
	reporter Reporter, pos token.Pos, typeName string, name string, location1 string, location2 string,
) {
	reporter.Report(
		analysis.Diagnostic{
			Pos:      pos,
			Category: CategoryDependencyRegistration,
			Message: fmt.Sprintf(
				"duplicate dependency name %q for type %s (first: %s, duplicate: %s)",
				name,
				typeName,
				location1,
				location2,
			),
		},
	)
}

// EmitOptionValidationError reports a fatal option validation error.
// Category "braider:option-validation" maps to SeverityCritical in phasedchecker, aborting the pipeline.
func (e *diagnosticEmitter) EmitOptionValidationError(reporter Reporter, pos token.Pos, reason string) {
	reporter.Report(
		analysis.Diagnostic{
			Pos:      pos,
			Category: CategoryOptionValidation,
			Message:  fmt.Sprintf("option validation error: %s", reason),
		},
	)
}

// EmitUnsupportedVariableExpression reports an unsupported Variable argument expression error.
// Category "braider:expression-validation" maps to SeverityCritical in phasedchecker, aborting the pipeline.
func (e *diagnosticEmitter) EmitUnsupportedVariableExpression(reporter Reporter, pos token.Pos, reason string) {
	reporter.Report(
		analysis.Diagnostic{
			Pos:      pos,
			Category: CategoryExpressionValidation,
			Message:  reason,
		},
	)
}

// EmitInvalidStructTagError reports an invalid braider struct tag value (braider:"").
func (e *diagnosticEmitter) EmitInvalidStructTagError(reporter Reporter, pos token.Pos, fieldName string) {
	reporter.Report(analysis.Diagnostic{
		Pos:      pos,
		Category: CategoryStructTagValidation,
		Message:  fmt.Sprintf("invalid braider struct tag on field %s: tag value must not be empty", fieldName),
	})
}

// EmitStructTagConflictError reports a braider struct tag conflict with WithoutConstructor.
func (e *diagnosticEmitter) EmitStructTagConflictError(reporter Reporter, pos token.Pos, fieldName string, reason string) {
	reporter.Report(analysis.Diagnostic{
		Pos:      pos,
		Category: CategoryStructTagValidation,
		Message:  fmt.Sprintf("braider struct tag conflict on field %s: %s", fieldName, reason),
	})
}

// EmitContainerTypeError reports a non-struct container type parameter.
func (e *diagnosticEmitter) EmitContainerTypeError(reporter Reporter, pos token.Pos, typeName string) {
	reporter.Report(analysis.Diagnostic{
		Pos:      pos,
		Category: CategoryContainerValidation,
		Message:  fmt.Sprintf("container type parameter must be a struct type, got %s", typeName),
	})
}

// EmitContainerFieldError reports an unresolvable container field.
func (e *diagnosticEmitter) EmitContainerFieldError(reporter Reporter, pos token.Pos, fieldName string, fieldType string, reason string) {
	reporter.Report(analysis.Diagnostic{
		Pos:      pos,
		Category: CategoryContainerValidation,
		Message:  fmt.Sprintf("container field %q (type %s): %s", fieldName, fieldType, reason),
	})
}
