package report_test

import (
	"go/token"
	"strings"
	"testing"

	"github.com/miyamo2/braider/internal/report"
	"golang.org/x/tools/go/analysis"
)

// mockReporter collects diagnostics for testing.
type mockReporter struct {
	diagnostics []analysis.Diagnostic
}

func (m *mockReporter) Report(d analysis.Diagnostic) {
	m.diagnostics = append(m.diagnostics, d)
}

func TestDiagnosticEmitter_EmitConstructorFix(t *testing.T) {
	emitter := report.NewDiagnosticEmitter()
	reporter := &mockReporter{}

	pos := token.Pos(100)
	fix := analysis.SuggestedFix{
		Message: "generate constructor for MyService",
		TextEdits: []analysis.TextEdit{
			{Pos: pos, End: pos, NewText: []byte("// code")},
		},
	}

	emitter.EmitConstructorFix(reporter, pos, "MyService", fix)

	if len(reporter.diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(reporter.diagnostics))
	}

	d := reporter.diagnostics[0]

	// Verify position
	if d.Pos != pos {
		t.Errorf("diagnostic.Pos = %d, want %d", d.Pos, pos)
	}

	// Verify message
	expectedMsg := "missing constructor for MyService"
	if d.Message != expectedMsg {
		t.Errorf("diagnostic.Message = %q, want %q", d.Message, expectedMsg)
	}

	// Verify suggested fix is included
	if len(d.SuggestedFixes) != 1 {
		t.Fatalf("expected 1 SuggestedFix, got %d", len(d.SuggestedFixes))
	}

	if d.SuggestedFixes[0].Message != fix.Message {
		t.Errorf("SuggestedFix.Message = %q, want %q", d.SuggestedFixes[0].Message, fix.Message)
	}
}

func TestDiagnosticEmitter_EmitCircularDependency(t *testing.T) {
	emitter := report.NewDiagnosticEmitter()
	reporter := &mockReporter{}

	pos := token.Pos(200)
	cycle := []string{"A", "B", "C", "A"}

	emitter.EmitCircularDependency(reporter, pos, cycle)

	if len(reporter.diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(reporter.diagnostics))
	}

	d := reporter.diagnostics[0]

	// Verify position
	if d.Pos != pos {
		t.Errorf("diagnostic.Pos = %d, want %d", d.Pos, pos)
	}

	// Verify message contains cycle
	if !strings.Contains(d.Message, "circular dependency") {
		t.Errorf("diagnostic.Message should contain 'circular dependency', got %q", d.Message)
	}

	if !strings.Contains(d.Message, "A -> B -> C -> A") {
		t.Errorf("diagnostic.Message should contain cycle path, got %q", d.Message)
	}

	// No suggested fix for errors
	if len(d.SuggestedFixes) != 0 {
		t.Errorf("circular dependency should have no suggested fixes")
	}
}

func TestDiagnosticEmitter_EmitGenerationError(t *testing.T) {
	emitter := report.NewDiagnosticEmitter()
	reporter := &mockReporter{}

	pos := token.Pos(300)

	emitter.EmitGenerationError(reporter, pos, "ConfigService", "invalid field type")

	if len(reporter.diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(reporter.diagnostics))
	}

	d := reporter.diagnostics[0]

	// Verify position
	if d.Pos != pos {
		t.Errorf("diagnostic.Pos = %d, want %d", d.Pos, pos)
	}

	// Verify message
	if !strings.Contains(d.Message, "failed to generate constructor") {
		t.Errorf("diagnostic.Message should contain 'failed to generate constructor', got %q", d.Message)
	}

	if !strings.Contains(d.Message, "ConfigService") {
		t.Errorf("diagnostic.Message should contain struct name, got %q", d.Message)
	}

	if !strings.Contains(d.Message, "invalid field type") {
		t.Errorf("diagnostic.Message should contain error reason, got %q", d.Message)
	}

	// No suggested fix for errors
	if len(d.SuggestedFixes) != 0 {
		t.Errorf("generation error should have no suggested fixes")
	}
}

func TestDiagnosticEmitter_EmitExistingConstructorFix(t *testing.T) {
	emitter := report.NewDiagnosticEmitter()
	reporter := &mockReporter{}

	pos := token.Pos(100)
	fix := analysis.SuggestedFix{
		Message: "regenerate constructor for MyService",
		TextEdits: []analysis.TextEdit{
			{Pos: pos, End: token.Pos(200), NewText: []byte("// code")},
		},
	}

	emitter.EmitExistingConstructorFix(reporter, pos, "MyService", fix)

	if len(reporter.diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(reporter.diagnostics))
	}

	d := reporter.diagnostics[0]

	// Verify message for existing constructor
	expectedMsg := "outdated constructor for MyService"
	if d.Message != expectedMsg {
		t.Errorf("diagnostic.Message = %q, want %q", d.Message, expectedMsg)
	}
}

func TestNewDiagnosticEmitter(t *testing.T) {
	emitter := report.NewDiagnosticEmitter()

	if emitter == nil {
		t.Fatal("NewDiagnosticEmitter returned nil")
	}

	// Verify it implements DiagnosticEmitter interface
	var _ report.DiagnosticEmitter = emitter
}

func TestDiagnosticEmitter_EmitMissingConstructorError(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		wantMsg  string
	}{
		{
			name:     "standard struct name",
			typeName: "UserRepository",
			wantMsg:  "Provide struct UserRepository requires a constructor (NewUserRepository)",
		},
		{
			name:     "empty type name",
			typeName: "",
			wantMsg:  "Provide struct  requires a constructor (New)",
		},
		{
			name:     "type with special characters",
			typeName: "User$Repository",
			wantMsg:  "Provide struct User$Repository requires a constructor (NewUser$Repository)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emitter := report.NewDiagnosticEmitter()
			reporter := &mockReporter{}
			pos := token.Pos(100)

			emitter.EmitMissingConstructorError(reporter, pos, tt.typeName)

			if len(reporter.diagnostics) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(reporter.diagnostics))
			}

			d := reporter.diagnostics[0]

			// Verify position
			if d.Pos != pos {
				t.Errorf("diagnostic.Pos = %d, want %d", d.Pos, pos)
			}

			// Verify message
			if d.Message != tt.wantMsg {
				t.Errorf("diagnostic.Message = %q, want %q", d.Message, tt.wantMsg)
			}

			// Verify category field
			if d.Category != "constructor" {
				t.Errorf("diagnostic.Category = %q, want %q", d.Category, "constructor")
			}

			// No suggested fixes for this error
			if len(d.SuggestedFixes) != 0 {
				t.Errorf("expected no SuggestedFixes, got %d", len(d.SuggestedFixes))
			}
		})
	}
}

func TestDiagnosticEmitter_EmitNonMainAppError(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		wantMsg  string
	}{
		{
			name:     "non-main function",
			funcName: "initialize",
			wantMsg:  "annotation.App must reference main function, got initialize",
		},
		{
			name:     "empty function name",
			funcName: "",
			wantMsg:  "annotation.App must reference main function, got ",
		},
		{
			name:     "capital Main",
			funcName: "Main",
			wantMsg:  "annotation.App must reference main function, got Main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emitter := report.NewDiagnosticEmitter()
			reporter := &mockReporter{}
			pos := token.Pos(150)

			emitter.EmitNonMainAppError(reporter, pos, tt.funcName)

			if len(reporter.diagnostics) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(reporter.diagnostics))
			}

			d := reporter.diagnostics[0]

			// Verify position
			if d.Pos != pos {
				t.Errorf("diagnostic.Pos = %d, want %d", d.Pos, pos)
			}

			// Verify message
			if d.Message != tt.wantMsg {
				t.Errorf("diagnostic.Message = %q, want %q", d.Message, tt.wantMsg)
			}

			// No suggested fixes for this error
			if len(d.SuggestedFixes) != 0 {
				t.Errorf("expected no SuggestedFixes, got %d", len(d.SuggestedFixes))
			}
		})
	}
}

func TestDiagnosticEmitter_EmitBootstrapFix(t *testing.T) {
	tests := []struct {
		name string
		fix  analysis.SuggestedFix
	}{
		{
			name: "single TextEdit",
			fix: analysis.SuggestedFix{
				Message: "add bootstrap code",
				TextEdits: []analysis.TextEdit{
					{Pos: token.Pos(100), End: token.Pos(100), NewText: []byte("// bootstrap")},
				},
			},
		},
		{
			name: "multiple TextEdits",
			fix: analysis.SuggestedFix{
				Message: "add bootstrap code",
				TextEdits: []analysis.TextEdit{
					{Pos: token.Pos(100), End: token.Pos(100), NewText: []byte("// part 1")},
					{Pos: token.Pos(200), End: token.Pos(200), NewText: []byte("// part 2")},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emitter := report.NewDiagnosticEmitter()
			reporter := &mockReporter{}
			pos := token.Pos(50)

			emitter.EmitBootstrapFix(reporter, pos, tt.fix)

			if len(reporter.diagnostics) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(reporter.diagnostics))
			}

			d := reporter.diagnostics[0]

			// Verify position
			if d.Pos != pos {
				t.Errorf("diagnostic.Pos = %d, want %d", d.Pos, pos)
			}

			// Verify static message
			expectedMsg := "bootstrap code is missing"
			if d.Message != expectedMsg {
				t.Errorf("diagnostic.Message = %q, want %q", d.Message, expectedMsg)
			}

			// Verify exactly 1 SuggestedFix
			if len(d.SuggestedFixes) != 1 {
				t.Fatalf("expected 1 SuggestedFix, got %d", len(d.SuggestedFixes))
			}

			// Verify fix is passed through unchanged
			if d.SuggestedFixes[0].Message != tt.fix.Message {
				t.Errorf("SuggestedFix.Message = %q, want %q", d.SuggestedFixes[0].Message, tt.fix.Message)
			}

			if len(d.SuggestedFixes[0].TextEdits) != len(tt.fix.TextEdits) {
				t.Errorf("SuggestedFix.TextEdits length = %d, want %d",
					len(d.SuggestedFixes[0].TextEdits), len(tt.fix.TextEdits))
			}
		})
	}
}

func TestDiagnosticEmitter_EmitBootstrapUpdateFix(t *testing.T) {
	tests := []struct {
		name string
		fix  analysis.SuggestedFix
	}{
		{
			name: "single replacement",
			fix: analysis.SuggestedFix{
				Message: "update bootstrap code",
				TextEdits: []analysis.TextEdit{
					{Pos: token.Pos(100), End: token.Pos(200), NewText: []byte("// updated")},
				},
			},
		},
		{
			name: "multiple replacements",
			fix: analysis.SuggestedFix{
				Message: "update bootstrap code",
				TextEdits: []analysis.TextEdit{
					{Pos: token.Pos(100), End: token.Pos(150), NewText: []byte("// update 1")},
					{Pos: token.Pos(200), End: token.Pos(250), NewText: []byte("// update 2")},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emitter := report.NewDiagnosticEmitter()
			reporter := &mockReporter{}
			pos := token.Pos(75)

			emitter.EmitBootstrapUpdateFix(reporter, pos, tt.fix)

			if len(reporter.diagnostics) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(reporter.diagnostics))
			}

			d := reporter.diagnostics[0]

			// Verify position
			if d.Pos != pos {
				t.Errorf("diagnostic.Pos = %d, want %d", d.Pos, pos)
			}

			// Verify static message (different from EmitBootstrapFix)
			expectedMsg := "bootstrap code is outdated"
			if d.Message != expectedMsg {
				t.Errorf("diagnostic.Message = %q, want %q", d.Message, expectedMsg)
			}

			// Verify exactly 1 SuggestedFix
			if len(d.SuggestedFixes) != 1 {
				t.Fatalf("expected 1 SuggestedFix, got %d", len(d.SuggestedFixes))
			}

			// Verify fix is passed through unchanged
			if d.SuggestedFixes[0].Message != tt.fix.Message {
				t.Errorf("SuggestedFix.Message = %q, want %q", d.SuggestedFixes[0].Message, tt.fix.Message)
			}

			if len(d.SuggestedFixes[0].TextEdits) != len(tt.fix.TextEdits) {
				t.Errorf("SuggestedFix.TextEdits length = %d, want %d",
					len(d.SuggestedFixes[0].TextEdits), len(tt.fix.TextEdits))
			}
		})
	}
}

func TestDiagnosticEmitter_EmitDuplicateAppWarning(t *testing.T) {
	emitter := report.NewDiagnosticEmitter()
	reporter := &mockReporter{}
	pos := token.Pos(250)

	emitter.EmitDuplicateAppWarning(reporter, pos)

	if len(reporter.diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(reporter.diagnostics))
	}

	d := reporter.diagnostics[0]

	// Verify position
	if d.Pos != pos {
		t.Errorf("diagnostic.Pos = %d, want %d", d.Pos, pos)
	}

	// Verify static message
	expectedMsg := "another annotation.App in the same package is being applied"
	if d.Message != expectedMsg {
		t.Errorf("diagnostic.Message = %q, want %q", d.Message, expectedMsg)
	}

	// No suggested fixes for warning
	if len(d.SuggestedFixes) != 0 {
		t.Errorf("expected no SuggestedFixes, got %d", len(d.SuggestedFixes))
	}
}

func TestDiagnosticEmitter_EmitPackageLoadWarning(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{
			name:   "standard error",
			reason: "module not found",
		},
		{
			name:   "empty reason",
			reason: "",
		},
		{
			name:   "long error message",
			reason: strings.Repeat("error: ", 50), // 350 chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emitter := report.NewDiagnosticEmitter()
			reporter := &mockReporter{}
			pos := token.Pos(300)

			emitter.EmitPackageLoadWarning(reporter, pos, tt.reason)

			if len(reporter.diagnostics) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(reporter.diagnostics))
			}

			d := reporter.diagnostics[0]

			// Verify position
			if d.Pos != pos {
				t.Errorf("diagnostic.Pos = %d, want %d", d.Pos, pos)
			}

			// Verify message contains reason
			if !strings.Contains(d.Message, tt.reason) {
				t.Errorf("diagnostic.Message should contain reason %q, got %q", tt.reason, d.Message)
			}

			// Verify message format
			expectedPrefix := "warning: failed to load module packages:"
			if !strings.Contains(d.Message, expectedPrefix) {
				t.Errorf("diagnostic.Message should contain %q, got %q", expectedPrefix, d.Message)
			}

			expectedSuffix := "(bootstrap may be incomplete)"
			if !strings.Contains(d.Message, expectedSuffix) {
				t.Errorf("diagnostic.Message should contain %q, got %q", expectedSuffix, d.Message)
			}

			// No suggested fixes for warning
			if len(d.SuggestedFixes) != 0 {
				t.Errorf("expected no SuggestedFixes, got %d", len(d.SuggestedFixes))
			}
		})
	}
}

func TestDiagnosticEmitter_EmitPackageWaitWarning(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{
			name:   "timeout error",
			reason: "context deadline exceeded",
		},
		{
			name:   "empty reason",
			reason: "",
		},
		{
			name:   "multiline error message",
			reason: "timeout error:\npackage A not ready\npackage B not ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emitter := report.NewDiagnosticEmitter()
			reporter := &mockReporter{}
			pos := token.Pos(350)

			emitter.EmitPackageWaitWarning(reporter, pos, tt.reason)

			if len(reporter.diagnostics) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(reporter.diagnostics))
			}

			d := reporter.diagnostics[0]

			// Verify position
			if d.Pos != pos {
				t.Errorf("diagnostic.Pos = %d, want %d", d.Pos, pos)
			}

			// Verify message contains reason
			if !strings.Contains(d.Message, tt.reason) {
				t.Errorf("diagnostic.Message should contain reason %q, got %q", tt.reason, d.Message)
			}

			// Verify message contains "warning:" and "timeout"
			if !strings.Contains(d.Message, "warning:") {
				t.Errorf("diagnostic.Message should contain 'warning:', got %q", d.Message)
			}

			if !strings.Contains(d.Message, "timeout") {
				t.Errorf("diagnostic.Message should contain 'timeout', got %q", d.Message)
			}

			expectedSuffix := "(bootstrap may be incomplete)"
			if !strings.Contains(d.Message, expectedSuffix) {
				t.Errorf("diagnostic.Message should contain %q, got %q", expectedSuffix, d.Message)
			}

			// No suggested fixes for warning
			if len(d.SuggestedFixes) != 0 {
				t.Errorf("expected no SuggestedFixes, got %d", len(d.SuggestedFixes))
			}
		})
	}
}

func TestDiagnosticEmitter_EmitGraphBuildError(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{
			name:   "circular dependency",
			reason: "circular dependency detected",
		},
		{
			name:   "empty reason",
			reason: "",
		},
		{
			name:   "unknown type",
			reason: "unknown type: interface{User}",
		},
		{
			name:   "missing dependency",
			reason: "dependency 'Logger' not found for 'UserService'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emitter := report.NewDiagnosticEmitter()
			reporter := &mockReporter{}
			pos := token.Pos(400)

			emitter.EmitGraphBuildError(reporter, pos, tt.reason)

			if len(reporter.diagnostics) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(reporter.diagnostics))
			}

			d := reporter.diagnostics[0]

			// Verify position
			if d.Pos != pos {
				t.Errorf("diagnostic.Pos = %d, want %d", d.Pos, pos)
			}

			// Verify message contains reason
			if !strings.Contains(d.Message, tt.reason) {
				t.Errorf("diagnostic.Message should contain reason %q, got %q", tt.reason, d.Message)
			}

			// Verify message format
			expectedPrefix := "failed to build dependency graph:"
			if !strings.Contains(d.Message, expectedPrefix) {
				t.Errorf("diagnostic.Message should contain %q, got %q", expectedPrefix, d.Message)
			}

			// No suggested fixes for error
			if len(d.SuggestedFixes) != 0 {
				t.Errorf("expected no SuggestedFixes, got %d", len(d.SuggestedFixes))
			}
		})
	}
}
