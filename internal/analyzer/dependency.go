package analyzer

import (
	"context"
	"go/types"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/registry"
	"github.com/miyamo2/braider/internal/report"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

// DependencyAnalyzer detects annotation.Provide and annotation.Injectable structs
// across all packages and registers them to global registries.
func DependencyAnalyzer(
	provideRegistry *registry.ProviderRegistry,
	injectRegistry *registry.InjectorRegistry,
	packageTracker *registry.PackageTracker,
	bootstrapCancel context.CancelCauseFunc,
	provideCallDetector detect.ProvideCallDetector,
	injectDetector detect.InjectDetector,
	structDetector detect.StructDetector,
	fieldAnalyzer detect.FieldAnalyzer,
	constructorAnalyzer detect.ConstructorAnalyzer,
	optionExtractor detect.OptionExtractor,
	constructorGenerator generate.ConstructorGenerator,
	suggestedFixBuilder report.SuggestedFixBuilder,
	diagnosticEmitter report.DiagnosticEmitter,
) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "braider_dependency",
		Doc:  "detects Provide and Inject annotated structs and registers to global registry",
		Run: NewDependencyAnalyzeRunner(
			provideRegistry,
			injectRegistry,
			packageTracker,
			bootstrapCancel,
			provideCallDetector,
			injectDetector,
			structDetector,
			fieldAnalyzer,
			constructorAnalyzer,
			optionExtractor,
			constructorGenerator,
			suggestedFixBuilder,
			diagnosticEmitter,
		).Run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

type DependencyAnalyzeRunner struct {
	provideRegistry      *registry.ProviderRegistry
	injectRegistry       *registry.InjectorRegistry
	packageTracker       *registry.PackageTracker
	bootstrapCancel      context.CancelCauseFunc
	provideCallDetector  detect.ProvideCallDetector
	injectDetector       detect.InjectDetector
	structDetector       detect.StructDetector
	fieldAnalyzer        detect.FieldAnalyzer
	constructorAnalyzer  detect.ConstructorAnalyzer
	optionExtractor      detect.OptionExtractor
	constructorGenerator generate.ConstructorGenerator
	suggestedFixBuilder  report.SuggestedFixBuilder
	diagnosticEmitter    report.DiagnosticEmitter
}

func NewDependencyAnalyzeRunner(
	provideRegistry *registry.ProviderRegistry,
	injectRegistry *registry.InjectorRegistry,
	packageTracker *registry.PackageTracker,
	bootstrapCancel context.CancelCauseFunc,
	provideCallDetector detect.ProvideCallDetector,
	injectDetector detect.InjectDetector,
	structDetector detect.StructDetector,
	fieldAnalyzer detect.FieldAnalyzer,
	constructorAnalyzer detect.ConstructorAnalyzer,
	optionExtractor detect.OptionExtractor,
	constructorGenerator generate.ConstructorGenerator,
	suggestedFixBuilder report.SuggestedFixBuilder,
	diagnosticEmitter report.DiagnosticEmitter,
) *DependencyAnalyzeRunner {
	return &DependencyAnalyzeRunner{
		provideRegistry:      provideRegistry,
		injectRegistry:       injectRegistry,
		packageTracker:       packageTracker,
		bootstrapCancel:      bootstrapCancel,
		provideCallDetector:  provideCallDetector,
		injectDetector:       injectDetector,
		structDetector:       structDetector,
		fieldAnalyzer:        fieldAnalyzer,
		constructorAnalyzer:  constructorAnalyzer,
		optionExtractor:      optionExtractor,
		constructorGenerator: constructorGenerator,
		suggestedFixBuilder:  suggestedFixBuilder,
		diagnosticEmitter:    diagnosticEmitter,
	}
}

func (r *DependencyAnalyzeRunner) Run(pass *analysis.Pass) (interface{}, error) {
	reporter := &passReporter{pass: pass}

	// Phase 1: Constructor Generation for Inject structs
	// Detect Inject structs that need constructors and generate them via SuggestedFix
	injectCandidates := r.structDetector.DetectCandidates(pass)
	for _, candidate := range injectCandidates {
		// Analyze fields (excluding annotation.Injectable)
		fields := r.fieldAnalyzer.AnalyzeFields(pass, candidate.StructType, candidate.InjectField)

		// Skip if no injectable fields
		if !r.fieldAnalyzer.HasInjectableFields(fields) {
			continue
		}

		// Check if existing constructor is up-to-date
		if candidate.ExistingConstructor != nil {
			// Extract expected dependencies from struct fields
			var expectedDeps []string
			for _, field := range fields {
				if field.Type != nil {
					expectedDeps = append(expectedDeps, field.Type.String())
				}
			}

			// Extract actual dependencies from existing constructor
			actualDeps := r.constructorAnalyzer.ExtractDependencies(pass, candidate.ExistingConstructor)

			// If dependencies match, skip (constructor is up-to-date)
			if dependenciesMatch(expectedDeps, actualDeps) {
				continue
			}
		}

		// Generate constructor code
		constructor, err := r.constructorGenerator.GenerateConstructor(candidate, fields)
		if err != nil {
			r.diagnosticEmitter.EmitGenerationError(
				reporter,
				candidate.TypeSpec.Pos(),
				candidate.TypeSpec.Name.Name,
				err.Error(),
			)
			continue
		}

		// Build suggested fix
		fix := r.suggestedFixBuilder.BuildConstructorFix(pass, candidate, constructor)

		// Emit diagnostic with suggested fix
		if candidate.ExistingConstructor != nil {
			r.diagnosticEmitter.EmitExistingConstructorFix(
				reporter,
				candidate.ExistingConstructor.Pos(),
				constructor.StructName,
				fix,
			)
		} else {
			r.diagnosticEmitter.EmitConstructorFix(
				reporter,
				candidate.TypeSpec.Pos(),
				constructor.StructName,
				fix,
			)
		}
	}

	// Phase 2: Detect and register Provide calls (var _ = annotation.Provide[T](fn))
	providers := r.provideCallDetector.DetectProviders(pass)
	for _, provider := range providers {
		// Extract dependencies from provider function parameters
		var dependencies []string
		if provider.ProviderFuncSig != nil {
			params := provider.ProviderFuncSig.Params()
			for i := 0; i < params.Len(); i++ {
				dependencies = append(dependencies, params.At(i).Type().String())
			}
		}

		// Extract option metadata
		var metadata detect.OptionMetadata
		if provider.CallExpr != nil && r.optionExtractor != nil {
			var providerFuncType types.Type
			if provider.ProviderFuncSig != nil {
				providerFuncType = provider.ProviderFuncSig
			}
			var err error
			metadata, err = r.optionExtractor.ExtractProvideOptions(pass, provider.CallExpr, providerFuncType)
			if err != nil {
				r.diagnosticEmitter.EmitOptionValidationError(reporter, provider.CallExpr.Pos(), err.Error())
				r.bootstrapCancel(err)
				continue
			}
		}

		// Determine the type name from return type's actual package
		var typePkgPath, typePkgName string
		if provider.ReturnType != nil {
			rt := provider.ReturnType
			if ptr, ok := rt.(*types.Pointer); ok {
				rt = ptr.Elem()
			}
			if named, ok := rt.(*types.Named); ok {
				if pkg := named.Obj().Pkg(); pkg != nil {
					typePkgPath = pkg.Path()
					typePkgName = pkg.Name()
				}
			}
		}
		if typePkgPath == "" {
			typePkgPath = pass.Pkg.Path()
			typePkgName = pass.Pkg.Name()
		}
		typeName := typePkgPath + "." + provider.ReturnTypeName

		// Determine registered type
		var registeredType types.Type
		if metadata.TypedInterface != nil {
			registeredType = metadata.TypedInterface
		} else if provider.ReturnType != nil {
			registeredType = provider.ReturnType
		}

		// Register to GlobalProviderRegistry
		if err := r.provideRegistry.Register(
			&registry.ProviderInfo{
				TypeName:        typeName,
				PackagePath:     typePkgPath,
				PackageName:     typePkgName,
				LocalName:       provider.ReturnTypeName,
				ConstructorName: provider.ProviderFuncName,
				Dependencies:    dependencies,
				Implements:      provider.Implements,
				IsPending:       false,
				RegisteredType:  registeredType,
				Name:            metadata.Name,
				OptionMetadata:  metadata,
			},
		); err != nil {
			existingLocation := pass.Pkg.Path()
			if existing, ok := r.provideRegistry.GetByName(typeName, metadata.Name); ok {
				existingLocation = existing.PackagePath
			}
			r.diagnosticEmitter.EmitDuplicateNamedDependencyWarning(
				reporter,
				provider.CallExpr.Pos(),
				typeName,
				metadata.Name,
				existingLocation,
				pass.Pkg.Path(),
			)
		}
	}

	// Phase 3: Detect and register Inject structs with IsPending flag
	// Re-detect injectors to include state after constructor generation
	injectors := r.structDetector.DetectCandidates(pass)
	for _, injector := range injectors {
		var dependencies []string
		var isPending bool

		// Determine IsPending flag and extract dependencies
		if injector.ExistingConstructor != nil {
			// Constructor exists on disk
			dependencies = r.constructorAnalyzer.ExtractDependencies(pass, injector.ExistingConstructor)
			isPending = false
		} else {
			// Constructor generated in this pass (pending)
			fields := r.fieldAnalyzer.AnalyzeFields(pass, injector.StructType, injector.InjectField)
			for _, field := range fields {
				if field.Type != nil {
					dependencies = append(dependencies, field.Type.String())
				}
			}
			isPending = true
		}

		// Extract option metadata for inject
		var metadata detect.OptionMetadata
		if injector.InjectField != nil && r.optionExtractor != nil {
			concreteType := types.NewPointer(pass.TypesInfo.ObjectOf(injector.TypeSpec.Name).Type())
			var err error
			metadata, err = r.optionExtractor.ExtractInjectOptions(pass, injector.InjectField.Type, concreteType)
			if err != nil {
				r.diagnosticEmitter.EmitOptionValidationError(reporter, injector.TypeSpec.Pos(), err.Error())
				r.bootstrapCancel(err)
				continue
			}
		}

		// Determine registered type
		var registeredType types.Type
		if metadata.TypedInterface != nil {
			registeredType = metadata.TypedInterface
		} else {
			// Constructors always return pointer types (*StructName),
			// so wrap the concrete struct type with a pointer.
			registeredType = types.NewPointer(pass.TypesInfo.ObjectOf(injector.TypeSpec.Name).Type())
		}

		// Detect implemented interfaces from the type
		var implements []string
		if injector.TypeSpec != nil {
			obj := pass.TypesInfo.Defs[injector.TypeSpec.Name]
			if obj != nil {
				if namedType, ok := obj.Type().(*types.Named); ok {
					implements = r.provideCallDetector.DetectImplementedInterfaces(pass, namedType)
				}
			}
		}

		// Register to GlobalInjectorRegistry with IsPending flag
		if err := r.injectRegistry.Register(
			&registry.InjectorInfo{
				TypeName:        pass.Pkg.Path() + "." + injector.TypeSpec.Name.Name,
				PackagePath:     pass.Pkg.Path(),
				PackageName:     pass.Pkg.Name(),
				LocalName:       injector.TypeSpec.Name.Name,
				ConstructorName: getConstructorName(injector),
				Dependencies:    dependencies,
				Implements:      implements,
				IsPending:       isPending,
				RegisteredType:  registeredType,
				Name:            metadata.Name,
				OptionMetadata:  metadata,
			},
		); err != nil {
			injectorTypeName := pass.Pkg.Path() + "." + injector.TypeSpec.Name.Name
			existingLocation := pass.Pkg.Path()
			if existing, ok := r.injectRegistry.GetByName(injectorTypeName, metadata.Name); ok {
				existingLocation = existing.PackagePath
			}
			r.diagnosticEmitter.EmitDuplicateNamedDependencyWarning(
				reporter,
				injector.TypeSpec.Pos(),
				injectorTypeName,
				metadata.Name,
				existingLocation,
				pass.Pkg.Path(),
			)
		}
	}

	// Phase 4: Mark package as scanned
	r.packageTracker.MarkPackageScanned(pass.Pkg.Path())

	return nil, nil
}

// getConstructorName returns the constructor name for an injector candidate.
// If ExistingConstructor exists, returns its name; otherwise returns expected name.
func getConstructorName(injector detect.ConstructorCandidate) string {
	if injector.ExistingConstructor != nil {
		return injector.ExistingConstructor.Name.Name
	}
	return "New" + injector.TypeSpec.Name.Name
}

// dependenciesMatch checks if two dependency lists are equivalent.
// Returns true if both lists contain the same dependencies (order-independent).
func dependenciesMatch(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}

	// Create a map for O(n) lookup
	depMap := make(map[string]bool)
	for _, dep := range expected {
		depMap[dep] = true
	}

	// Check if all actual dependencies are in expected
	for _, dep := range actual {
		if !depMap[dep] {
			return false
		}
	}

	return true
}

// passReporter adapts analysis.Pass to report.Reporter interface.
type passReporter struct {
	pass *analysis.Pass
}

func (r *passReporter) Report(d analysis.Diagnostic) {
	r.pass.Report(d)
}
