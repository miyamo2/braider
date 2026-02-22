package checker

import (
	"flag"
	"fmt"
)

// ParseArgs parses command-line arguments and returns a partial Config.
// The caller MUST set Config.Pipeline and Config.DiagnosticPolicy before calling Run,
// since ParseArgs cannot know which analyzers or diagnostic policies to use.
func ParseArgs(programName string, args []string) (*Config, error) {
	fs := flag.NewFlagSet(programName, flag.ContinueOnError)

	var (
		fix       bool
		printDiff bool
		verbose   bool
	)
	fs.BoolVar(&fix, "fix", false, "apply suggested fixes")
	fs.BoolVar(&printDiff, "diff", false, "print diffs instead of applying fixes")
	fs.BoolVar(&verbose, "v", false, "verbose output")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	patterns := fs.Args()
	if len(patterns) == 0 {
		return nil, fmt.Errorf("no packages specified")
	}

	return &Config{
		Fix:       fix,
		PrintDiff: printDiff,
		Verbose:   verbose,
		Patterns:  patterns,
	}, nil
}
