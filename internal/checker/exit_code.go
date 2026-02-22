package checker

// Severity represents the severity level of a diagnostic.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarn
	SeverityError
)

// CategoryRule maps a diagnostic category string to a severity level.
// The severity determines both the output destination and exit code contribution.
type CategoryRule struct {
	// Category is the diagnostic category string to match.
	Category string
	// Severity is the severity level for this category.
	Severity Severity
}

// DiagnosticPolicy defines the complete mapping from diagnostic categories to severity levels.
type DiagnosticPolicy struct {
	// Rules is an ordered list of category-to-severity mappings.
	// The first matching rule wins.
	Rules []CategoryRule
	// DefaultSeverity is applied when no rule matches a diagnostic's category.
	DefaultSeverity Severity
}

// ResolveSeverity finds the severity for a given category.
func (p DiagnosticPolicy) ResolveSeverity(category string) Severity {
	for _, rule := range p.Rules {
		if rule.Category == category {
			return rule.Severity
		}
	}
	return p.DefaultSeverity
}
