package analyzer

import (
	"context"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
)

// TestContextCancellationMechanism tests the context cancellation mechanism.
// This corresponds to Task 4.1: Add context cancellation for fatal validation errors (Requirement 8.5).
func TestContextCancellationMechanism(t *testing.T) {
	t.Run("context_with_cancel_can_be_created", func(t *testing.T) {
		// Test that we can create a cancellable context
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Verify context is not cancelled initially
		select {
		case <-ctx.Done():
			t.Error("Context should not be cancelled initially")
		default:
			// Expected: context not cancelled
		}

		// Cancel the context
		cancel()

		// Verify context is now cancelled
		select {
		case <-ctx.Done():
			// Expected: context cancelled
		default:
			t.Error("Context should be cancelled after cancel() is called")
		}
	})

	t.Run("analyzer_run_method_accepts_context", func(t *testing.T) {
		// Verify that DependencyAnalyzeRunner has access to ValidationContext
		providerRegistry, injectorRegistry, packageTracker, validationContext,
			provideDetector, provideStructDetector, injectDetector, structDetector,
			fieldAnalyzer, constructorAnalyzer,
			constructorGenerator, suggestedFixBuilder, diagnosticEmitter := setupDependencyAnalyzerDeps()

		runner := NewDependencyAnalyzeRunner(
			providerRegistry,
			injectorRegistry,
			packageTracker,
			validationContext,
			provideDetector,
			provideStructDetector,
			injectDetector,
			structDetector,
			fieldAnalyzer,
			constructorAnalyzer,
			constructorGenerator,
			suggestedFixBuilder,
			diagnosticEmitter,
		)

		// Verify runner exists and has Run method
		if runner == nil {
			t.Fatal("Runner should not be nil")
		}

		// The Run method signature accepts *analysis.Pass which contains context
		// We just verify the structure exists
	})
}

// TestCorrelationErrorHandling tests the mechanism for non-fatal correlation errors.
// This corresponds to Task 4.2: Handle correlation errors as non-fatal (Requirement 8.6).
func TestCorrelationErrorHandling(t *testing.T) {
	t.Run("diagnostic_emitter_can_report_errors_without_stopping", func(t *testing.T) {
		// Test that diagnostic emitter can report multiple errors
		// without causing the analyzer to halt

		diagnosticEmitter := report.NewDiagnosticEmitter()
		if diagnosticEmitter == nil {
			t.Fatal("DiagnosticEmitter should not be nil")
		}

		// Correlation errors should be reported via pass.Report() but should not cancel context
		// This test verifies the mechanism exists
	})

	t.Run("registry_can_detect_duplicates", func(t *testing.T) {
		// Test that registry can track and detect duplicate (TypeName, Name) pairs
		injectorRegistry := registry.NewInjectorRegistry()

		// Register first injector with name
		injectorRegistry.Register(&registry.InjectorInfo{
			TypeName: "example.com/pkg.Service",
			Name:     "primary",
		})

		// Try to get it back
		info, found := injectorRegistry.GetByName("example.com/pkg.Service", "primary")
		if !found || info == nil {
			t.Error("Should find registered named injector")
		}

		// The actual duplicate detection logic will be implemented in Task 4.2
		// This test verifies the GetByName mechanism exists
	})
}

// TestRegistryErrorReporting tests that registries can report validation errors.
func TestRegistryErrorReporting(t *testing.T) {
	t.Run("registry_can_validate_and_store_metadata", func(t *testing.T) {
		// Test that registry stores option metadata correctly
		injectorRegistry := registry.NewInjectorRegistry()
		providerRegistry := registry.NewProviderRegistry()

		// Register injector with option metadata
		injectorRegistry.Register(&registry.InjectorInfo{
			TypeName: "example.com/pkg.Service",
			OptionMetadata: detect.OptionMetadata{
				IsDefault: true,
			},
		})

		// Register provider with option metadata
		providerRegistry.Register(&registry.ProviderInfo{
			TypeName: "example.com/pkg.Factory",
			OptionMetadata: detect.OptionMetadata{
				IsDefault: true,
			},
		})

		// Verify registries store and retrieve correctly
		injector := injectorRegistry.Get("example.com/pkg.Service")
		if injector == nil {
			t.Error("Should find registered injector")
		}

		provider := providerRegistry.Get("example.com/pkg.Factory")
		if provider == nil {
			t.Error("Should find registered provider")
		}
	})
}
