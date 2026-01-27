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
