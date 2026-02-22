package report

// Diagnostic category constants used for exit code policy decisions.
const (
	// CategoryConstructorFix is emitted when a constructor needs generation/update.
	CategoryConstructorFix = "constructor_fix"
	// CategoryBootstrapFix is emitted when bootstrap code needs generation/update.
	CategoryBootstrapFix = "bootstrap_fix"
	// CategoryError is emitted for fatal errors (circular deps, unresolvable types).
	CategoryError = "error"
	// CategoryWarning is emitted for non-fatal warnings (duplicate app, timeout).
	CategoryWarning = "warning"
)
