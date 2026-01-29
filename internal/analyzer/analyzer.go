package analyzer

import (
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/report"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

// Analyzer is the braider analyzer that detects annotation.Inject-embedded structs
// and generates constructor functions via SuggestedFix.
var Analyzer = &analysis.Analyzer{
	Name:     "braider",
	Doc:      "resolves DI bindings and generates wiring code",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

// Components used by the analyzer
var (
	constructorGenerator = generate.NewConstructorGenerator()
	suggestedFixBuilder  = report.NewSuggestedFixBuilder()
)

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := &passReporter{pass: pass}

	// Phase 1: Detect constructor candidates
	candidates := structDetector.DetectCandidates(pass)

	// Phase 2: Process each candidate
	for _, candidate := range candidates {
		// Analyze fields (excluding annotation.Inject)
		fields := fieldAnalyzer.AnalyzeFields(pass, candidate.StructType, candidate.InjectField)

		// Skip structs with no injectable fields
		if !fieldAnalyzer.HasInjectableFields(fields) {
			continue
		}

		// Generate constructor code
		constructor, err := constructorGenerator.GenerateConstructor(candidate, fields)
		if err != nil {
			diagnosticEmitter.EmitGenerationError(
				reporter,
				candidate.TypeSpec.Pos(),
				candidate.TypeSpec.Name.Name,
				err.Error(),
			)
			continue
		}

		// Build suggested fix
		fix := suggestedFixBuilder.BuildConstructorFix(pass, candidate, constructor)

		// Emit diagnostic with suggested fix
		if candidate.ExistingConstructor != nil {
			diagnosticEmitter.EmitExistingConstructorFix(
				reporter,
				candidate.ExistingConstructor.Pos(),
				constructor.StructName,
				fix,
			)
		} else {
			diagnosticEmitter.EmitConstructorFix(
				reporter,
				candidate.TypeSpec.Pos(),
				constructor.StructName,
				fix,
			)
		}
	}

	return nil, nil
}
