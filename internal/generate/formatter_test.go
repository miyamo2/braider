package generate_test

import (
	"strings"
	"testing"

	"github.com/miyamo2/braider/internal/generate"
)

func TestCodeFormatter_FormatCode(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		wantErr       bool
		expectedParts []string
	}{
		{
			name: "formats valid constructor",
			code: `func NewService(repo Repository) *Service {
return &Service{repo: repo}
}`,
			wantErr: false,
			expectedParts: []string{
				"func NewService(repo Repository) *Service",
				"return &Service{repo: repo}",
			},
		},
		{
			name: "formats with proper indentation",
			code: `func NewService(repo Repository,logger Logger) *Service {
return &Service{
repo: repo,
logger: logger,
}
}`,
			wantErr: false,
			expectedParts: []string{
				"repo Repository",
				"logger Logger",
			},
		},
		{
			name:    "returns error for invalid Go code",
			code:    "func invalid syntax {",
			wantErr: true,
		},
		{
			name:    "returns error for empty code",
			code:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := generate.NewCodeFormatter()
			result, err := formatter.FormatCode(tt.code)

			if tt.wantErr {
				if err == nil {
					t.Error("FormatCode() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("FormatCode() unexpected error: %v", err)
			}

			for _, part := range tt.expectedParts {
				if !strings.Contains(result, part) {
					t.Errorf("FormatCode() missing expected content: %q\nGot:\n%s", part, result)
				}
			}
		})
	}
}

func TestCodeFormatter_OutputPassesGofmt(t *testing.T) {
	formatter := generate.NewCodeFormatter()

	// Various valid Go code snippets
	validCodes := []string{
		`func NewService(repo Repository) *Service {
return &Service{repo: repo}
}`,
		`// NewConfig is a constructor for Config.
func NewConfig() *Config {
return &Config{}
}`,
		`func NewMulti(a A, b B, c C) *Multi {
return &Multi{a: a, b: b, c: c}
}`,
	}

	for _, code := range validCodes {
		result, err := formatter.FormatCode(code)
		if err != nil {
			t.Fatalf("FormatCode() failed for valid code: %v", err)
		}

		// Formatted code should be idempotent
		result2, err := formatter.FormatCode(result)
		if err != nil {
			t.Fatalf("FormatCode() failed on already formatted code: %v", err)
		}

		if result != result2 {
			t.Errorf("FormatCode() output is not idempotent:\nFirst:\n%s\nSecond:\n%s", result, result2)
		}
	}
}
