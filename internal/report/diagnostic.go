package report

import (
	"fmt"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
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

	// EmitMissingConstructorError reports Provide struct without constructor.
	EmitMissingConstructorError(reporter Reporter, pos token.Pos, typeName string)
}

// diagnosticEmitter is the default implementation of DiagnosticEmitter.
type diagnosticEmitter struct{}

// NewDiagnosticEmitter creates a new DiagnosticEmitter instance.
func NewDiagnosticEmitter() DiagnosticEmitter {
	return &diagnosticEmitter{}
}

// EmitConstructorFix reports a diagnostic with constructor SuggestedFix.
func (e *diagnosticEmitter) EmitConstructorFix(
	reporter Reporter,
	pos token.Pos,
	structName string,
	fix analysis.SuggestedFix,
) {
	reporter.Report(analysis.Diagnostic{
		Pos:            pos,
		Message:        fmt.Sprintf("missing constructor for %s", structName),
		SuggestedFixes: []analysis.SuggestedFix{fix},
	})
}

// EmitExistingConstructorFix reports a diagnostic for replacing an existing constructor.
func (e *diagnosticEmitter) EmitExistingConstructorFix(
	reporter Reporter,
	pos token.Pos,
	structName string,
	fix analysis.SuggestedFix,
) {
	reporter.Report(analysis.Diagnostic{
		Pos:            pos,
		Message:        fmt.Sprintf("outdated constructor for %s", structName),
		SuggestedFixes: []analysis.SuggestedFix{fix},
	})
}

// EmitCircularDependency reports a circular dependency error.
func (e *diagnosticEmitter) EmitCircularDependency(reporter Reporter, pos token.Pos, cycle []string) {
	cyclePath := strings.Join(cycle, " -> ")
	reporter.Report(analysis.Diagnostic{
		Pos:     pos,
		Message: fmt.Sprintf("circular dependency detected: %s", cyclePath),
	})
}

// EmitGenerationError reports a constructor generation failure.
func (e *diagnosticEmitter) EmitGenerationError(reporter Reporter, pos token.Pos, structName string, reason string) {
	reporter.Report(analysis.Diagnostic{
		Pos:     pos,
		Message: fmt.Sprintf("failed to generate constructor for %s: %s", structName, reason),
	})
}

// EmitMissingConstructorError reports Provide struct without constructor.
func (e *diagnosticEmitter) EmitMissingConstructorError(reporter Reporter, pos token.Pos, typeName string) {
	reporter.Report(analysis.Diagnostic{
		Pos:      pos,
		Category: "constructor",
		Message:  fmt.Sprintf("Provide struct %s requires a constructor (New%s)", typeName, typeName),
	})
}
