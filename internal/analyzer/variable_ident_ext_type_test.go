package analyzer

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestIntegration_VariableIdentExternalType tests Variable[variable.Default] with a local
// variable whose type comes from an external package (e.g., var Output = os.Stdout).
// PackagePath must reflect the declaring package (config), not the type's package (os).
// Bootstrap should emit "config.Output", not "os.Output".
func TestIntegration_VariableIdentExternalType(t *testing.T) {
	depAnalyzer, appAnalyzer := setupIntegrationDeps()
	testdir := "testdata/bootstrapgen/variable_ident_ext_type"
	analysistest.Run(t, testdir, depAnalyzer, "variable_ident_ext_type/config")
	analysistest.RunWithSuggestedFixes(t, testdir, appAnalyzer, ".")
}
