package checker

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

// CategoryRule maps a diagnostic category string directly to an exit code.
// Exit code 0 means the category is ignored (does not affect the exit code).
type CategoryRule struct {
	// Category is the diagnostic category string to match.
	Category string
	// Code is the exit code for this category. Use 0 to ignore.
	Code int
}

// ExitCodePolicy defines the complete mapping from diagnostic categories to exit codes.
type ExitCodePolicy struct {
	// Rules is an ordered list of category-to-exit-code mappings.
	// The first matching rule wins.
	Rules []CategoryRule
	// DefaultCode is applied when no rule matches a diagnostic's category.
	DefaultCode int
}

// CategorizedDiagnostic pairs a diagnostic with its source analyzer and package.
type CategorizedDiagnostic struct {
	analysis.Diagnostic
	Analyzer *analysis.Analyzer
	Package  *packages.Package
}

// ComputeExitCode evaluates all diagnostics against the policy and returns the exit code.
func (p ExitCodePolicy) ComputeExitCode(diagnostics []CategorizedDiagnostic) int {
	exitCode := 0
	for _, d := range diagnostics {
		code := p.resolveExitCode(d.Category)
		if code > exitCode {
			exitCode = code
		}
	}
	return exitCode
}

// resolveExitCode finds the exit code for a given category.
func (p ExitCodePolicy) resolveExitCode(category string) int {
	for _, rule := range p.Rules {
		if rule.Category == category {
			return rule.Code
		}
	}
	return p.DefaultCode
}
