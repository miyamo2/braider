// Package checker implements a phase-based analysis driver.
//
// Unlike multichecker, which runs all analyzers per-package with no phase ordering,
// this checker supports executing analyzers in sequential phases. Each phase completes
// for ALL packages before the next phase starts.
//
// The checker uses the public golang.org/x/tools/go/analysis/checker.Analyze() API
// to run analyzers within each phase.
package checker

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/tools/go/analysis"
	gochecker "golang.org/x/tools/go/analysis/checker"
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
	// DiagnosticPolicy determines how diagnostic categories map to severity levels and exit codes.
	DiagnosticPolicy DiagnosticPolicy
}

// Run executes the full pipeline: load packages, run phases, apply fixes, and returns the exit code.
func Run(cfg Config) (int, error) {
	args, err := parseArgs(os.Args[0], os.Args[1:])
	if err != nil {
		return 1, fmt.Errorf("parsing arguments: %w", err)
	}
	return run(cfg, args)
}

// Run executes the full pipeline: load packages, run phases, apply fixes, and returns the exit code.
func run(cfg Config, args *Args) (int, error) {
	if args == nil {
		return 1, fmt.Errorf("args cannot be nil")
	}
	if len(cfg.Pipeline.Phases) == 0 {
		return 1, fmt.Errorf("pipeline has no phases")
	}

	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax}, args.Patterns...)
	if err != nil {
		return 1, fmt.Errorf("loading packages: %w", err)
	}

	var loadErrors []error
	packages.Visit(
		pkgs, nil, func(pkg *packages.Package) {
			for _, err := range pkg.Errors {
				loadErrors = append(loadErrors, err)
			}
		},
	)
	if len(loadErrors) > 0 {
		return 1, fmt.Errorf("package loading errors: %w", errors.Join(loadErrors...))
	}

	hasError := false
	hasDiagnostics := false
	for _, phase := range cfg.Pipeline.Phases {
		graph, err := gochecker.Analyze(
			phase.Analyzers, pkgs, &gochecker.Options{
				Sequential: args.Sequential,
			},
		)
		if err != nil {
			return 1, fmt.Errorf("phase %q: %w", phase.Name, err)
		}

		for act := range graph.All() {
			if act.Err != nil {
				hasError = true
				continue
			}
			if !act.IsRoot {
				continue
			}
			for _, d := range act.Diagnostics {
				severity := cfg.DiagnosticPolicy.ResolveSeverity(d.Category)
				switch severity {
				case SeverityError:
					hasError = true
				case SeverityWarn:
					hasDiagnostics = true
				}
			}
		}

		if !args.Fix {
			graph.PrintText(os.Stderr, -1)
		}

		if args.Fix {
			if err := applyFixes(graph, args.PrintDiff, args.Verbose); err != nil {
				return 1, fmt.Errorf("applying fixes for phase %q: %w", phase.Name, err)
			}
		}

		if phase.AfterPhase != nil {
			if err := phase.AfterPhase(graph); err != nil {
				return 1, fmt.Errorf("phase %q after-phase callback: %w", phase.Name, err)
			}
		}
	}

	if hasError {
		return 1, nil
	}
	if !args.Fix && hasDiagnostics {
		return 3, nil
	}
	return 0, nil
}
