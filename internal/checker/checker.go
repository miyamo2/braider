// Package checker implements a phase-based analysis driver for braider.
//
// Unlike multichecker, which runs all analyzers per-package with no phase ordering,
// this checker supports executing analyzers in sequential phases. Each phase completes
// for ALL packages before the next phase starts. This eliminates the need for
// polling-based coordination (PackageTracker) between analyzers.
//
// The checker uses the public golang.org/x/tools/go/analysis/checker.Analyze() API
// to run analyzers within each phase.
package checker

import (
	"errors"
	"fmt"
	"os"

	gochecker "golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

// Phase represents a group of analyzers that run together.
// All analyzers in a phase complete on ALL packages before the next phase starts.
type Phase struct {
	// Name is a human-readable label for logging/debugging.
	Name string
	// Analyzers are the analysis passes to run in this phase.
	// Analyzers within a phase may run concurrently across packages.
	Analyzers []*analysis.Analyzer
	// AfterPhase is an optional callback invoked after all analyzers in this phase
	// have completed on all packages and fixes have been applied.
	// It receives the resulting Graph, enabling callers to extract per-package
	// Action.Result values and aggregate them for consumption by subsequent phases.
	// This is the extension point for Result-based cross-phase data passing.
	AfterPhase func(graph *gochecker.Graph) error
}

// Pipeline defines an ordered sequence of phases.
type Pipeline struct {
	// Phases are executed in order. Each phase completes fully before the next begins.
	Phases []Phase
}

// Config controls the behavior of the checker.
type Config struct {
	// Pipeline defines the phase-ordered analyzer execution plan.
	Pipeline Pipeline
	// ExitPolicy determines how diagnostics map to exit codes.
	ExitPolicy ExitCodePolicy
	// Fix enables automatic application of SuggestedFixes.
	Fix bool
	// PrintDiff prints unified diffs instead of applying fixes (used with Fix).
	PrintDiff bool
	// Verbose enables verbose output during fix application.
	Verbose bool
	// Sequential forces sequential (non-parallel) execution within each phase.
	Sequential bool
	// Patterns are the package patterns to analyze (e.g., "./...").
	Patterns []string
}

// Result holds the outcome of a complete pipeline execution.
type Result struct {
	// PhaseResults contains the checker.Graph for each phase, indexed by phase name.
	PhaseResults map[string]*gochecker.Graph
	// AllDiagnostics is a flattened list of all diagnostics across all phases.
	AllDiagnostics []CategorizedDiagnostic
	// ExitCode is the computed exit code based on ExitPolicy.
	ExitCode int
}

// DefaultPipeline creates a two-phase pipeline for braider's standard analyzers.
// Phase 1 runs the DependencyAnalyzer, Phase 2 runs the AppAnalyzer.
func DefaultPipeline(depAnalyzer, appAnalyzer *analysis.Analyzer) Pipeline {
	return Pipeline{
		Phases: []Phase{
			{
				Name:      "dependency",
				Analyzers: []*analysis.Analyzer{depAnalyzer},
			},
			{
				Name:      "app",
				Analyzers: []*analysis.Analyzer{appAnalyzer},
			},
		},
	}
}

// Run executes the full pipeline: load packages, run phases, apply fixes, compute exit code.
func Run(cfg Config) (*Result, error) {
	if len(cfg.Pipeline.Phases) == 0 {
		return nil, fmt.Errorf("pipeline has no phases")
	}
	// Step 1: Load packages
	loadCfg := &packages.Config{
		Mode: packages.LoadAllSyntax,
	}
	pkgs, err := packages.Load(loadCfg, cfg.Patterns...)
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	// Check for package loading errors
	var loadErrors []error
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, err := range pkg.Errors {
			loadErrors = append(loadErrors, err)
		}
	})
	if len(loadErrors) > 0 {
		return nil, fmt.Errorf("package loading errors: %w", errors.Join(loadErrors...))
	}

	result := &Result{
		PhaseResults: make(map[string]*gochecker.Graph),
	}

	// Step 2: Execute phases sequentially
	for _, phase := range cfg.Pipeline.Phases {
		opts := &gochecker.Options{
			Sequential: cfg.Sequential,
		}

		graph, err := gochecker.Analyze(phase.Analyzers, pkgs, opts)
		if err != nil {
			return nil, fmt.Errorf("phase %q: %w", phase.Name, err)
		}

		result.PhaseResults[phase.Name] = graph

		// Collect diagnostics from this phase (root actions only)
		for act := range graph.All() {
			if !act.IsRoot {
				continue
			}
			for _, d := range act.Diagnostics {
				result.AllDiagnostics = append(result.AllDiagnostics, CategorizedDiagnostic{
					Diagnostic: d,
					Analyzer:   act.Analyzer,
					Package:    act.Package,
				})
			}
		}

		// Print diagnostics for this phase
		graph.PrintText(os.Stderr, -1)

		// Step 3: Apply fixes after each phase (if enabled)
		if cfg.Fix || cfg.PrintDiff {
			if err := ApplyFixes(graph, cfg.PrintDiff, cfg.Verbose); err != nil {
				return nil, fmt.Errorf("applying fixes for phase %q: %w", phase.Name, err)
			}
		}

		// Step 4: Invoke AfterPhase callback (e.g., aggregate Results for next phase)
		if phase.AfterPhase != nil {
			if err := phase.AfterPhase(graph); err != nil {
				return nil, fmt.Errorf("phase %q after-phase callback: %w", phase.Name, err)
			}
		}
	}

	// Step 5: Compute exit code
	result.ExitCode = cfg.ExitPolicy.ComputeExitCode(result.AllDiagnostics)
	return result, nil
}
