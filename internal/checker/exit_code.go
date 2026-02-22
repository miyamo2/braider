package checker

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

// ExitAction determines what effect a diagnostic category has on the exit code.
type ExitAction int

const (
	// ExitIgnore means this category does not affect the exit code.
	ExitIgnore ExitAction = iota
	// ExitNonZero means this category causes exit code 1.
	ExitNonZero
	// ExitWithCode means this category causes a specific exit code.
	ExitWithCode
)

// CategoryRule defines how a single diagnostic category maps to exit behavior.
type CategoryRule struct {
	// Category is the diagnostic category string to match.
	Category string
	// Action determines the exit behavior for this category.
	Action ExitAction
	// Code is used when Action is ExitWithCode.
	Code int
}

// ExitCodePolicy defines the complete mapping from diagnostic categories to exit codes.
type ExitCodePolicy struct {
	// Rules is an ordered list of category-to-exit-code mappings.
	// The first matching rule wins.
	Rules []CategoryRule
	// DefaultAction is applied when no rule matches a diagnostic's category.
	DefaultAction ExitAction
	// DefaultCode is used when DefaultAction is ExitWithCode.
	DefaultCode int
}

// CategorizedDiagnostic pairs a diagnostic with its source analyzer and package.
type CategorizedDiagnostic struct {
	analysis.Diagnostic
	Analyzer *analysis.Analyzer
	Package  *packages.Package
}

// DefaultExitCodePolicy returns the standard braider exit code policy.
// Diagnostics with SuggestedFixes (constructor/bootstrap) are ignored.
// Error diagnostics cause exit 1. Warnings are ignored.
func DefaultExitCodePolicy() ExitCodePolicy {
	return ExitCodePolicy{
		Rules: []CategoryRule{
			{Category: "constructor_fix", Action: ExitIgnore},
			{Category: "bootstrap_fix", Action: ExitIgnore},
			{Category: "warning", Action: ExitIgnore},
			{Category: "error", Action: ExitNonZero},
		},
		DefaultAction: ExitNonZero,
		DefaultCode:   1,
	}
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
			switch rule.Action {
			case ExitIgnore:
				return 0
			case ExitNonZero:
				return 1
			case ExitWithCode:
				return rule.Code
			}
		}
	}
	// Default
	switch p.DefaultAction {
	case ExitIgnore:
		return 0
	case ExitNonZero:
		return 1
	case ExitWithCode:
		return p.DefaultCode
	}
	return 1
}
