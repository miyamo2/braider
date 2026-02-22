package checker

import "testing"

func TestSeverityConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  Severity
		want int
	}{
		{"SeverityInfo", SeverityInfo, 0},
		{"SeverityWarn", SeverityWarn, 1},
		{"SeverityError", SeverityError, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.got) != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestDiagnosticPolicy_resolveSeverity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		policy   DiagnosticPolicy
		category string
		want     Severity
	}{
		{
			name:     "no rules returns DefaultSeverity",
			policy:   DiagnosticPolicy{DefaultSeverity: SeverityInfo},
			category: "anything",
			want:     SeverityInfo,
		},
		{
			name: "unmatched category returns DefaultSeverity",
			policy: DiagnosticPolicy{
				Rules:           []CategoryRule{{Category: "other", Severity: SeverityError}},
				DefaultSeverity: SeverityWarn,
			},
			category: "nomatch",
			want:     SeverityWarn,
		},
		{
			name: "exact match returns Error",
			policy: DiagnosticPolicy{
				Rules: []CategoryRule{{Category: "err", Severity: SeverityError}},
			},
			category: "err",
			want:     SeverityError,
		},
		{
			name: "exact match returns Warn",
			policy: DiagnosticPolicy{
				Rules: []CategoryRule{{Category: "warn", Severity: SeverityWarn}},
			},
			category: "warn",
			want:     SeverityWarn,
		},
		{
			name: "exact match returns Info",
			policy: DiagnosticPolicy{
				Rules:           []CategoryRule{{Category: "info", Severity: SeverityInfo}},
				DefaultSeverity: SeverityError,
			},
			category: "info",
			want:     SeverityInfo,
		},
		{
			name: "first matching rule wins",
			policy: DiagnosticPolicy{
				Rules: []CategoryRule{
					{Category: "cat", Severity: SeverityWarn},
					{Category: "cat", Severity: SeverityError},
				},
			},
			category: "cat",
			want:     SeverityWarn,
		},
		{
			name: "empty category matches",
			policy: DiagnosticPolicy{
				Rules:           []CategoryRule{{Category: "", Severity: SeverityError}},
				DefaultSeverity: SeverityInfo,
			},
			category: "",
			want:     SeverityError,
		},
		{
			name:     "zero value Policy returns SeverityInfo",
			policy:   DiagnosticPolicy{},
			category: "anything",
			want:     SeverityInfo,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.policy.resolveSeverity(tt.category)
			if got != tt.want {
				t.Errorf("resolveSeverity(%q) = %d, want %d", tt.category, got, tt.want)
			}
		})
	}
}
