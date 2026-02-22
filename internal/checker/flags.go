package checker

import (
	"flag"
	"fmt"
)

// Flags holds all parsed command-line flags for the braider checker.
type Flags struct {
	Fix       bool
	PrintDiff bool
	Verbose   bool
	Patterns  []string
}

// ParseArgs parses command-line arguments and returns the configuration.
// Unlike analysisflags.Parse, this does not register per-analyzer flags
// since braider's analyzers are fixed and not user-selectable.
func ParseArgs(args []string) (*Config, error) {
	fs := flag.NewFlagSet("braider", flag.ContinueOnError)

	var flags Flags
	fs.BoolVar(&flags.Fix, "fix", false, "apply suggested fixes")
	fs.BoolVar(&flags.PrintDiff, "diff", false, "print diffs instead of applying fixes")
	fs.BoolVar(&flags.Verbose, "v", false, "verbose output")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	flags.Patterns = fs.Args()
	if len(flags.Patterns) == 0 {
		return nil, fmt.Errorf("no packages specified")
	}

	return &Config{
		ExitPolicy: DefaultExitCodePolicy(),
		Fix:        flags.Fix,
		PrintDiff:  flags.PrintDiff,
		Verbose:    flags.Verbose,
		Patterns:   flags.Patterns,
	}, nil
}
