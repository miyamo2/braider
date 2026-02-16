package detect

import (
	"testing"
)

func TestProvideCallDetector_DetectProviders_WithRealPackage(t *testing.T) {
	_, pass := LoadTestPackage(t, "provide_call")

	detector := NewProvideCallDetector(ResolveMarkers())
	candidates := detector.DetectProviders(pass)

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates from real package, got %d", len(candidates))
	}

	// Verify candidate details
	foundNames := make(map[string]bool)
	for _, c := range candidates {
		foundNames[c.ProviderFuncName] = true

		if c.CallExpr == nil {
			t.Errorf("candidate %q: CallExpr should not be nil", c.ProviderFuncName)
		}
		if c.ProviderFunc == nil {
			t.Errorf("candidate %q: ProviderFunc should not be nil", c.ProviderFuncName)
		}
		if c.ProviderFuncSig == nil {
			t.Errorf("candidate %q: ProviderFuncSig should not be nil", c.ProviderFuncName)
		}
		if c.ReturnType == nil {
			t.Errorf("candidate %q: ReturnType should not be nil", c.ProviderFuncName)
		}
	}

	if !foundNames["NewUserRepository"] {
		t.Error("expected candidate NewUserRepository not found")
	}
	if !foundNames["NewServiceImpl"] {
		t.Error("expected candidate NewServiceImpl not found")
	}

	// Verify return type names
	for _, c := range candidates {
		switch c.ProviderFuncName {
		case "NewUserRepository":
			if c.ReturnTypeName != "UserRepository" {
				t.Errorf("NewUserRepository ReturnTypeName = %q, want %q", c.ReturnTypeName, "UserRepository")
			}
		case "NewServiceImpl":
			if c.ReturnTypeName != "ServiceImpl" {
				t.Errorf("NewServiceImpl ReturnTypeName = %q, want %q", c.ReturnTypeName, "ServiceImpl")
			}
		}
	}
}
