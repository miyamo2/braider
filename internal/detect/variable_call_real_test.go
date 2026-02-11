package detect

import (
	"testing"
)

func TestVariableCallDetector_DetectVariables_WithRealPackage(t *testing.T) {
	_, pass := LoadTestPackage(t, "variable_call")

	detector := NewVariableCallDetector()
	candidates := detector.DetectVariables(pass)

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates from real package, got %d", len(candidates))
	}

	// Verify candidate details
	for _, c := range candidates {
		if c.CallExpr == nil {
			t.Errorf("candidate: CallExpr should not be nil")
		}
		if c.ArgumentExpr == nil {
			t.Errorf("candidate: ArgumentExpr should not be nil")
		}
		if c.ArgumentType == nil {
			t.Errorf("candidate: ArgumentType should not be nil")
		}
		if c.ExpressionText == "" {
			t.Errorf("candidate: ExpressionText should not be empty")
		}
		if !c.IsQualified {
			t.Errorf("candidate: IsQualified should be true for os.Stdout/os.Stderr")
		}
		if len(c.ExpressionPkgs) == 0 {
			t.Errorf("candidate: ExpressionPkgs should not be empty")
		}
		if _, ok := c.ExpressionPkgs["os"]; !ok {
			t.Errorf("candidate: ExpressionPkgs should contain 'os', got %v", c.ExpressionPkgs)
		}
	}

	// Verify expression texts
	foundTexts := make(map[string]bool)
	for _, c := range candidates {
		foundTexts[c.ExpressionText] = true
	}

	if !foundTexts["os.Stdout"] {
		t.Error("expected expression text 'os.Stdout' not found")
	}
	if !foundTexts["os.Stderr"] {
		t.Error("expected expression text 'os.Stderr' not found")
	}

	// Verify TypeName (os.File for *os.File after pointer dereference)
	for _, c := range candidates {
		if c.TypeName != "os.File" {
			t.Errorf("TypeName = %q, want %q", c.TypeName, "os.File")
		}
		if c.PackagePath != "os" {
			t.Errorf("PackagePath = %q, want %q", c.PackagePath, "os")
		}
	}
}
