package checker

import (
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		args     []string
		wantFix  bool
		wantDiff bool
		wantV    bool
		wantPats []string
		wantErr  string
	}{
		{
			name:     "basic pattern",
			args:     []string{"./..."},
			wantPats: []string{"./..."},
		},
		{
			name:     "fix flag",
			args:     []string{"-fix", "./..."},
			wantFix:  true,
			wantPats: []string{"./..."},
		},
		{
			name:     "diff flag",
			args:     []string{"-diff", "./..."},
			wantDiff: true,
			wantPats: []string{"./..."},
		},
		{
			name:     "verbose flag",
			args:     []string{"-v", "./..."},
			wantV:    true,
			wantPats: []string{"./..."},
		},
		{
			name:     "all flags combined",
			args:     []string{"-fix", "-diff", "-v", "./..."},
			wantFix:  true,
			wantDiff: true,
			wantV:    true,
			wantPats: []string{"./..."},
		},
		{
			name:     "multiple patterns",
			args:     []string{"pkg1", "pkg2"},
			wantPats: []string{"pkg1", "pkg2"},
		},
		{
			name:    "no packages",
			args:    []string{},
			wantErr: "no packages specified",
		},
		{
			name:    "unknown flag",
			args:    []string{"-unknown", "./..."},
			wantErr: "flag provided but not defined",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := ParseArgs("test", tt.args)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.Fix != tt.wantFix {
				t.Errorf("Fix = %v, want %v", cfg.Fix, tt.wantFix)
			}
			if cfg.PrintDiff != tt.wantDiff {
				t.Errorf("PrintDiff = %v, want %v", cfg.PrintDiff, tt.wantDiff)
			}
			if cfg.Verbose != tt.wantV {
				t.Errorf("Verbose = %v, want %v", cfg.Verbose, tt.wantV)
			}
			if len(cfg.Patterns) != len(tt.wantPats) {
				t.Fatalf("Patterns = %v, want %v", cfg.Patterns, tt.wantPats)
			}
			for i, p := range cfg.Patterns {
				if p != tt.wantPats[i] {
					t.Errorf("Patterns[%d] = %q, want %q", i, p, tt.wantPats[i])
				}
			}
			if len(cfg.Pipeline.Phases) != 0 {
				t.Errorf("Pipeline.Phases should be empty, got %d phases", len(cfg.Pipeline.Phases))
			}
			if len(cfg.DiagnosticPolicy.Rules) != 0 {
				t.Errorf("DiagnosticPolicy.Rules should be empty, got %d rules", len(cfg.DiagnosticPolicy.Rules))
			}
		})
	}
}
