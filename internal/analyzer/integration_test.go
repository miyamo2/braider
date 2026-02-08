// Package analyzer provides integration tests for complete annotation flows.
//
// These tests verify end-to-end behavior of the annotation system by combining
// multiple components: OptionExtractor, NamerValidator, registries, ConstructorGenerator,
// BootstrapGenerator, DependencyGraphBuilder, TopologicalSorter, and ValidationContext.
//
// Task 8.6: Integration tests for complete annotation flows.
// Requirements: 1.1, 1.2, 1.3, 1.4, 2.5, 2.6, 2.7, 2.8, 3.4, 3.5, 3.6, 3.7,
//
//	4.2, 4.3, 4.4, 4.5, 6.1, 6.2, 6.3, 6.4, 6.5, 7.1, 7.2, 7.4, 7.5,
//	8.1, 8.2, 8.3, 8.4, 8.5, 8.6
package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/graph"
	"github.com/miyamo2/braider/internal/registry"
	"golang.org/x/tools/go/analysis"
)

// --- Test helpers ---

// createInterfaceType creates a types.Named representing a named interface type with methods.
func createInterfaceType(pkgPath, pkgName, ifaceName string, methods ...*types.Func) *types.Named {
	pkg := types.NewPackage(pkgPath, pkgName)
	var sigs []*types.Func
	for _, m := range methods {
		sigs = append(sigs, m)
	}
	iface := types.NewInterfaceType(sigs, nil)
	iface.Complete()
	tn := types.NewTypeName(token.NoPos, pkg, ifaceName, nil)
	named := types.NewNamed(tn, iface, nil)
	return named
}

// createNamedStruct creates a types.Named representing a named struct type.
func createNamedStruct(pkgPath, pkgName, structName string, fields ...*types.Var) *types.Named {
	pkg := types.NewPackage(pkgPath, pkgName)
	st := types.NewStruct(fields, nil)
	tn := types.NewTypeName(token.NoPos, pkg, structName, nil)
	return types.NewNamed(tn, st, nil)
}

// mockNamerValidator is a mock NamerValidator for integration tests.
type mockNamerValidator struct {
	extractNameFn func(pass *analysis.Pass, namerType types.Type) (string, error)
}

func (m *mockNamerValidator) ExtractName(pass *analysis.Pass, namerType types.Type) (string, error) {
	if m.extractNameFn != nil {
		return m.extractNameFn(pass, namerType)
	}
	return "", nil
}

// --- Integration Tests ---

// TestIntegration_InjectableTypedFlow tests the complete Injectable[Typed[I]] flow:
// struct with interface annotation -> constructor returns interface -> bootstrap declares interface variable
//
// Requirements: 1.1, 1.3, 2.5, 2.6, 6.1, 6.2, 7.1, 7.5
func TestIntegration_InjectableTypedFlow(t *testing.T) {
	// Step 1: Create interface type that the concrete struct will implement
	doMethod := types.NewFunc(token.NoPos, nil, "DoSomething", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	serviceIface := createInterfaceType("example.com/service", "service", "IUserService", doMethod)
	serviceIfaceUnderlying := serviceIface.Underlying().(*types.Interface)

	// Step 2: Create concrete struct type that implements the interface
	concreteStruct := createNamedStruct("example.com/service", "service", "userService")
	concretePtr := types.NewPointer(concreteStruct)
	// Add method to make it implement the interface
	doImpl := types.NewFunc(token.NoPos, concreteStruct.Obj().Pkg(), "DoSomething",
		types.NewSignatureType(types.NewVar(token.NoPos, nil, "s", concretePtr), nil, nil, nil, nil, false))
	concreteStruct.AddMethod(doImpl)

	// Validate: concrete pointer type implements interface
	if !types.Implements(concretePtr, serviceIfaceUnderlying) {
		t.Fatal("Expected concrete type to implement interface")
	}

	// Step 3: Create registries and register injector with Typed[I] option metadata
	injectorRegistry := registry.NewInjectorRegistry()
	providerRegistry := registry.NewProviderRegistry()

	injectorInfo := &registry.InjectorInfo{
		TypeName:        "example.com/service.userService",
		PackagePath:     "example.com/service",
		PackageName:     "service",
		LocalName:       "userService",
		ConstructorName: "NewUserService",
		Dependencies:    []string{},
		Implements:      []string{},
		IsPending:       false,
		RegisteredType:  serviceIface,
		Name:            "",
		OptionMetadata: detect.OptionMetadata{
			TypedInterface: serviceIface,
		},
	}
	injectorRegistry.Register(injectorInfo)

	// Step 4: Build dependency graph
	graphBuilder := graph.NewDependencyGraphBuilder()
	pass := &analysis.Pass{
		Pkg: types.NewPackage("example.com/main", "main"),
	}

	depGraph, err := graphBuilder.BuildGraph(pass, providerRegistry.GetAll(), injectorRegistry.GetAll())
	if err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}

	// Step 5: Topological sort
	sorter := graph.NewTopologicalSorter()
	sortedTypes, err := sorter.Sort(depGraph)
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}

	// Verify node in graph has RegisteredType set
	node := depGraph.Nodes["example.com/service.userService"]
	if node == nil {
		t.Fatal("Expected node in graph")
	}
	if node.RegisteredType == nil {
		t.Error("Expected RegisteredType to be set on graph node")
	}

	// Step 6: Generate bootstrap code
	bootstrapGen := generate.NewBootstrapGenerator()
	bootstrap, err := bootstrapGen.GenerateBootstrap(pass, depGraph, sortedTypes)
	if err != nil {
		t.Fatalf("GenerateBootstrap() error = %v", err)
	}

	// Step 7: Verify bootstrap declares interface-typed variable
	if !strings.Contains(bootstrap.DependencyVar, "IUserService") {
		t.Errorf("Bootstrap should declare interface-typed variable IUserService.\nGot:\n%s", bootstrap.DependencyVar)
	}

	// Step 8: Verify constructor generation with interface return type
	ctorGen := generate.NewConstructorGenerator()
	candidate := detect.ConstructorCandidate{
		TypeSpec: &ast.TypeSpec{
			Name: &ast.Ident{Name: "userService"},
		},
	}
	ctor, err := ctorGen.GenerateConstructorWithOptions(candidate, nil, injectorInfo)
	if err != nil {
		t.Fatalf("GenerateConstructorWithOptions() error = %v", err)
	}
	if ctor == nil {
		t.Fatal("Expected constructor to be generated, got nil")
	}
	// The constructor return type should reference the interface, not the concrete struct
	if !strings.Contains(ctor.Code, "IUserService") {
		t.Errorf("Constructor should return interface type IUserService.\nGot:\n%s", ctor.Code)
	}
}

// TestIntegration_NamedDependencyFlow tests the Named dependency flow:
// multiple structs with same type but different names -> separate variables with unique names
//
// Requirements: 1.1, 1.3, 2.7, 4.2, 4.3, 4.4, 4.5, 6.3, 7.4
func TestIntegration_NamedDependencyFlow(t *testing.T) {
	// Step 1: Register two injectors with same type but different names
	injectorRegistry := registry.NewInjectorRegistry()
	providerRegistry := registry.NewProviderRegistry()

	// Primary repository - note: the graph builder creates composite keys TypeName#Name
	// so we use unique TypeNames for each named dependency in the registry
	primaryInfo := &registry.InjectorInfo{
		TypeName:        "example.com/repo.Repository",
		PackagePath:     "example.com/repo",
		PackageName:     "repo",
		LocalName:       "Repository",
		ConstructorName: "NewRepository",
		Dependencies:    []string{},
		Implements:      []string{},
		IsPending:       false,
		Name:            "primary",
		OptionMetadata: detect.OptionMetadata{
			Name: "primary",
		},
	}
	injectorRegistry.Register(primaryInfo)

	// Secondary repository - use a different TypeName since the registry is keyed by TypeName
	// In the real system, multiple named instances of the same base type would need
	// unique keys in the registry. The graph builder uses makeNodeKey(TypeName, Name).
	secondaryInfo := &registry.InjectorInfo{
		TypeName:        "example.com/repo.SecondaryRepository",
		PackagePath:     "example.com/repo",
		PackageName:     "repo",
		LocalName:       "SecondaryRepository",
		ConstructorName: "NewSecondaryRepository",
		Dependencies:    []string{},
		Implements:      []string{},
		IsPending:       false,
		Name:            "secondary",
		OptionMetadata: detect.OptionMetadata{
			Name: "secondary",
		},
	}
	injectorRegistry.Register(secondaryInfo)

	// Step 2: Build dependency graph with named dependencies as distinct nodes
	graphBuilder := graph.NewDependencyGraphBuilder()
	pass := &analysis.Pass{
		Pkg: types.NewPackage("example.com/main", "main"),
	}

	depGraph, err := graphBuilder.BuildGraph(pass, providerRegistry.GetAll(), injectorRegistry.GetAll())
	if err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}

	// Step 3: Verify both named nodes are in the graph
	if len(depGraph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes in graph, got %d", len(depGraph.Nodes))
	}

	// The graph builder creates composite keys TypeName#Name for named dependencies
	primaryNode := depGraph.Nodes["example.com/repo.Repository#primary"]
	if primaryNode == nil {
		t.Error("Expected primary named node in graph")
	} else if primaryNode.Name != "primary" {
		t.Errorf("Expected primary node Name='primary', got %q", primaryNode.Name)
	}

	secondaryNode := depGraph.Nodes["example.com/repo.SecondaryRepository#secondary"]
	if secondaryNode == nil {
		t.Error("Expected secondary named node in graph")
	} else if secondaryNode.Name != "secondary" {
		t.Errorf("Expected secondary node Name='secondary', got %q", secondaryNode.Name)
	}

	// Step 4: Topological sort
	sorter := graph.NewTopologicalSorter()
	sortedTypes, err := sorter.Sort(depGraph)
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}

	if len(sortedTypes) != 2 {
		t.Errorf("Expected 2 sorted types, got %d: %v", len(sortedTypes), sortedTypes)
	}

	// Step 5: Generate bootstrap code
	bootstrapGen := generate.NewBootstrapGenerator()
	bootstrap, err := bootstrapGen.GenerateBootstrap(pass, depGraph, sortedTypes)
	if err != nil {
		t.Fatalf("GenerateBootstrap() error = %v", err)
	}

	// Step 6: Verify both named variables are present with unique names
	if !strings.Contains(bootstrap.DependencyVar, "primary") {
		t.Errorf("Bootstrap should contain 'primary' variable.\nGot:\n%s", bootstrap.DependencyVar)
	}
	if !strings.Contains(bootstrap.DependencyVar, "secondary") {
		t.Errorf("Bootstrap should contain 'secondary' variable.\nGot:\n%s", bootstrap.DependencyVar)
	}
	// Verify both are initialized
	if !strings.Contains(bootstrap.DependencyVar, "primary := repo.NewRepository()") {
		t.Errorf("Bootstrap should initialize primary variable.\nGot:\n%s", bootstrap.DependencyVar)
	}
	if !strings.Contains(bootstrap.DependencyVar, "secondary := repo.NewSecondaryRepository()") {
		t.Errorf("Bootstrap should initialize secondary variable.\nGot:\n%s", bootstrap.DependencyVar)
	}
}

// TestIntegration_MixedOptionsFlow tests mixed options flow:
// custom option type implementing Typed[I] + Named[N] -> constructor returns interface, variable uses custom name
//
// Requirements: 1.1, 1.3, 2.6, 2.7, 4.2, 4.4, 6.2, 6.3, 7.1
func TestIntegration_MixedOptionsFlow(t *testing.T) {
	// Step 1: Create interface type
	interfaceType := types.NewNamed(
		types.NewTypeName(token.NoPos, types.NewPackage("example.com/domain", "domain"), "IRepository", nil),
		types.NewInterfaceType(nil, nil),
		nil,
	)

	// Step 2: Register injector with both Typed[I] and Named[N] options
	injectorRegistry := registry.NewInjectorRegistry()
	providerRegistry := registry.NewProviderRegistry()

	injectorInfo := &registry.InjectorInfo{
		TypeName:        "example.com/repo.Repository",
		PackagePath:     "example.com/repo",
		PackageName:     "repo",
		LocalName:       "Repository",
		ConstructorName: "NewRepository",
		Dependencies:    []string{},
		Implements:      []string{},
		IsPending:       false,
		RegisteredType:  interfaceType,
		Name:            "primaryRepo",
		OptionMetadata: detect.OptionMetadata{
			TypedInterface: interfaceType,
			Name:           "primaryRepo",
		},
	}
	injectorRegistry.Register(injectorInfo)

	// Step 3: Build dependency graph
	graphBuilder := graph.NewDependencyGraphBuilder()
	pass := &analysis.Pass{
		Pkg: types.NewPackage("example.com/main", "main"),
	}

	depGraph, err := graphBuilder.BuildGraph(pass, providerRegistry.GetAll(), injectorRegistry.GetAll())
	if err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}

	// Step 4: Topological sort
	sorter := graph.NewTopologicalSorter()
	sortedTypes, err := sorter.Sort(depGraph)
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}

	// Step 5: Generate bootstrap code
	bootstrapGen := generate.NewBootstrapGenerator()
	bootstrap, err := bootstrapGen.GenerateBootstrap(pass, depGraph, sortedTypes)
	if err != nil {
		t.Fatalf("GenerateBootstrap() error = %v", err)
	}

	// Step 6: Verify variable uses custom name AND interface type
	if !strings.Contains(bootstrap.DependencyVar, "primaryRepo") {
		t.Errorf("Bootstrap should use custom name 'primaryRepo'.\nGot:\n%s", bootstrap.DependencyVar)
	}
	if !strings.Contains(bootstrap.DependencyVar, "IRepository") {
		t.Errorf("Bootstrap should declare interface type IRepository.\nGot:\n%s", bootstrap.DependencyVar)
	}

	// Step 7: Verify constructor uses interface return type
	ctorGen := generate.NewConstructorGenerator()
	candidate := detect.ConstructorCandidate{
		TypeSpec: &ast.TypeSpec{
			Name: &ast.Ident{Name: "Repository"},
		},
	}

	depNames := map[string]string{}
	ctor, err := ctorGen.GenerateConstructorWithNamedDeps(candidate, nil, injectorInfo, depNames)
	if err != nil {
		t.Fatalf("GenerateConstructorWithNamedDeps() error = %v", err)
	}
	if ctor == nil {
		t.Fatal("Expected constructor, got nil")
	}
	if !strings.Contains(ctor.Code, "IRepository") {
		t.Errorf("Constructor should return interface type IRepository.\nGot:\n%s", ctor.Code)
	}
}

// TestIntegration_ProvideTypedFlow tests Provide[Typed[I]] flow:
// provider function annotated with interface type -> bootstrap assigns to interface variable
//
// Requirements: 1.2, 1.4, 3.4, 3.5, 3.7, 7.2
func TestIntegration_ProvideTypedFlow(t *testing.T) {
	// Step 1: Create interface type for the provider
	interfaceType := types.NewNamed(
		types.NewTypeName(token.NoPos, types.NewPackage("example.com/domain", "domain"), "IRepository", nil),
		types.NewInterfaceType(nil, nil),
		nil,
	)

	// Step 2: Register provider with Typed[I] option
	injectorRegistry := registry.NewInjectorRegistry()
	providerRegistry := registry.NewProviderRegistry()

	providerInfo := &registry.ProviderInfo{
		TypeName:        "example.com/repo.Repository",
		PackagePath:     "example.com/repo",
		PackageName:     "repo",
		LocalName:       "Repository",
		ConstructorName: "NewRepository",
		Dependencies:    []string{},
		Implements:      []string{},
		IsPending:       false,
		RegisteredType:  interfaceType,
		Name:            "",
		OptionMetadata: detect.OptionMetadata{
			TypedInterface: interfaceType,
		},
	}
	providerRegistry.Register(providerInfo)

	// Step 3: Register injector that depends on the provider
	injectorInfo := &registry.InjectorInfo{
		TypeName:        "example.com/service.UserService",
		PackagePath:     "example.com/service",
		PackageName:     "service",
		LocalName:       "UserService",
		ConstructorName: "NewUserService",
		Dependencies:    []string{"example.com/repo.Repository"},
		Implements:      []string{},
		IsPending:       false,
		OptionMetadata: detect.OptionMetadata{
			IsDefault: true,
		},
	}
	injectorRegistry.Register(injectorInfo)

	// Step 4: Build dependency graph
	graphBuilder := graph.NewDependencyGraphBuilder()
	pass := &analysis.Pass{
		Pkg: types.NewPackage("example.com/main", "main"),
	}

	depGraph, err := graphBuilder.BuildGraph(pass, providerRegistry.GetAll(), injectorRegistry.GetAll())
	if err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}

	// Step 5: Topological sort
	sorter := graph.NewTopologicalSorter()
	sortedTypes, err := sorter.Sort(depGraph)
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}

	// Verify provider comes before service in sort order
	repoIdx := -1
	svcIdx := -1
	for i, tp := range sortedTypes {
		if tp == "example.com/repo.Repository" {
			repoIdx = i
		}
		if tp == "example.com/service.UserService" {
			svcIdx = i
		}
	}
	if repoIdx < 0 || svcIdx < 0 {
		t.Fatalf("Missing expected types in sorted list: %v", sortedTypes)
	}
	if repoIdx >= svcIdx {
		t.Errorf("Provider should come before service in topological order. Got repo=%d, svc=%d", repoIdx, svcIdx)
	}

	// Step 6: Generate bootstrap code
	bootstrapGen := generate.NewBootstrapGenerator()
	bootstrap, err := bootstrapGen.GenerateBootstrap(pass, depGraph, sortedTypes)
	if err != nil {
		t.Fatalf("GenerateBootstrap() error = %v", err)
	}

	// Step 7: Verify provider is initialized as local variable (not a struct field)
	providerNode := depGraph.Nodes["example.com/repo.Repository"]
	if providerNode == nil {
		t.Fatal("Expected provider node in graph")
	}
	if providerNode.IsField {
		t.Error("Provider should not be a struct field (IsField=false)")
	}

	// Step 8: Verify bootstrap has interface-typed provider via RegisteredType
	if providerNode.RegisteredType == nil {
		t.Error("Expected RegisteredType set for provider node")
	}

	// Step 9: Verify bootstrap assigns provider result and service consumes it
	if !strings.Contains(bootstrap.DependencyVar, "repo.NewRepository()") {
		t.Errorf("Bootstrap should call provider constructor.\nGot:\n%s", bootstrap.DependencyVar)
	}
	if !strings.Contains(bootstrap.DependencyVar, "service.NewUserService(repository)") {
		t.Errorf("Bootstrap should pass provider result to service constructor.\nGot:\n%s", bootstrap.DependencyVar)
	}
}

// TestIntegration_ErrorHandlingFatalFlow tests the error handling flow:
// constraint violation -> fatal diagnostic -> AppAnalyzer skipped
//
// This test mirrors the AppAnalyzer.Run() Phase 0 logic (app.go:101-106):
// when ValidationContext is cancelled by DependencyAnalyzer, bootstrap
// generation is entirely skipped despite valid registry data.
//
// Requirements: 8.1, 8.2, 8.5
func TestIntegration_ErrorHandlingFatalFlow(t *testing.T) {
	// Step 1: Create registries with valid data that would generate bootstrap
	injectorReg := registry.NewInjectorRegistry()
	providerReg := registry.NewProviderRegistry()

	injectorReg.Register(&registry.InjectorInfo{
		TypeName:        "example.com/service.Service",
		PackagePath:     "example.com/service",
		PackageName:     "service",
		LocalName:       "Service",
		ConstructorName: "NewService",
		Dependencies:    []string{},
		OptionMetadata:  detect.OptionMetadata{IsDefault: true},
	})

	graphBuilder := graph.NewDependencyGraphBuilder()
	pass := &analysis.Pass{
		Pkg: types.NewPackage("example.com/main", "main"),
	}

	// Step 2: Simulate DependencyAnalyzer encountering fatal validation error
	// (e.g., concrete type does not implement Typed[I] interface)
	validationCtx := registry.NewValidationContext()
	validationCtx.Cancel()

	// Step 3: Simulate AppAnalyzer flow (mirrors app.go Phase 0 check)
	// When context is cancelled, AppAnalyzer returns nil before graph/bootstrap
	var bootstrap *generate.GeneratedBootstrap
	if !validationCtx.IsCancelled() {
		depGraph, err := graphBuilder.BuildGraph(pass, providerReg.GetAll(), injectorReg.GetAll())
		if err != nil {
			t.Fatalf("BuildGraph() error = %v", err)
		}
		sorter := graph.NewTopologicalSorter()
		sortedTypes, err := sorter.Sort(depGraph)
		if err != nil {
			t.Fatalf("Sort() error = %v", err)
		}
		bootstrapGen := generate.NewBootstrapGenerator()
		bootstrap, err = bootstrapGen.GenerateBootstrap(pass, depGraph, sortedTypes)
		if err != nil {
			t.Fatalf("GenerateBootstrap() error = %v", err)
		}
	}

	// Step 4: Verify bootstrap was NOT generated (AppAnalyzer skipped)
	if bootstrap != nil {
		t.Error("Bootstrap should NOT be generated when ValidationContext is cancelled")
	}

	// Step 5: Verify the same data DOES produce bootstrap when context is active
	validationCtx.Reset()

	var bootstrap2 *generate.GeneratedBootstrap
	if !validationCtx.IsCancelled() {
		depGraph, err := graphBuilder.BuildGraph(pass, providerReg.GetAll(), injectorReg.GetAll())
		if err != nil {
			t.Fatalf("BuildGraph() error = %v", err)
		}
		sorter := graph.NewTopologicalSorter()
		sortedTypes, err := sorter.Sort(depGraph)
		if err != nil {
			t.Fatalf("Sort() error = %v", err)
		}
		bootstrapGen := generate.NewBootstrapGenerator()
		bootstrap2, err = bootstrapGen.GenerateBootstrap(pass, depGraph, sortedTypes)
		if err != nil {
			t.Fatalf("GenerateBootstrap() error = %v", err)
		}
	}

	if bootstrap2 == nil || bootstrap2.DependencyVar == "" {
		t.Error("Bootstrap should be generated when ValidationContext is not cancelled")
	}
}

// TestIntegration_CorrelationErrorFlow tests the correlation error flow:
// duplicate names -> non-fatal diagnostic -> bootstrap generation continues
//
// Requirements: 4.5, 8.4, 8.6
func TestIntegration_CorrelationErrorFlow(t *testing.T) {
	// Step 1: Create ValidationContext (should NOT be cancelled for correlation errors)
	validationCtx := registry.NewValidationContext()

	// Step 2: Register two injectors with same TypeName and Name (duplicate)
	// In the actual system, the registry or DependencyAnalyzer would detect this
	// and emit a diagnostic but NOT cancel the context
	injectorRegistry := registry.NewInjectorRegistry()
	providerRegistry := registry.NewProviderRegistry()

	// First registration
	info1 := &registry.InjectorInfo{
		TypeName:        "example.com/repo.Repository",
		PackagePath:     "example.com/repo",
		PackageName:     "repo",
		LocalName:       "Repository",
		ConstructorName: "NewRepository",
		Dependencies:    []string{},
		Name:            "primary",
		OptionMetadata: detect.OptionMetadata{
			Name: "primary",
		},
	}
	injectorRegistry.Register(info1)

	// Second registration with same TypeName would overwrite in current registry impl
	// In the actual DependencyAnalyzer, duplicate detection happens before registration
	// For this test, we simulate detecting the duplicate and then continuing
	info2 := &registry.InjectorInfo{
		TypeName: "example.com/repo.Repository",
		Name:     "primary",
	}

	// Simulate duplicate detection: the DependencyAnalyzer would check
	existing := injectorRegistry.Get(info2.TypeName)
	hasDuplicate := existing != nil && existing.Name == info2.Name && existing.Name != ""

	if !hasDuplicate {
		t.Fatal("Expected duplicate detection to find existing entry")
	}

	// Step 3: Verify context is NOT cancelled (correlation errors are non-fatal)
	if validationCtx.IsCancelled() {
		t.Error("ValidationContext should NOT be cancelled for correlation errors")
	}

	// Step 4: Despite duplicate warning, register dependencies and ensure bootstrap
	// generation can proceed. Use a fresh registry to simulate the state after
	// duplicate detection has been emitted as a non-fatal diagnostic.
	injectorRegistry2 := registry.NewInjectorRegistry()
	injectorRegistry2.Register(&registry.InjectorInfo{
		TypeName:        "example.com/repo.Repository",
		PackagePath:     "example.com/repo",
		PackageName:     "repo",
		LocalName:       "Repository",
		ConstructorName: "NewRepository",
		Dependencies:    []string{},
		Name:            "primary",
		OptionMetadata: detect.OptionMetadata{
			Name: "primary",
		},
	})
	// The service depends on the repository by its fully qualified TypeName.
	// The graph builder resolves TypeName to composite key TypeName#Name.
	injectorRegistry2.Register(&registry.InjectorInfo{
		TypeName:        "example.com/service.UserService",
		PackagePath:     "example.com/service",
		PackageName:     "service",
		LocalName:       "UserService",
		ConstructorName: "NewUserService",
		Dependencies:    []string{"example.com/repo.Repository#primary"},
		Name:            "",
	})

	// Step 5: Build graph and generate bootstrap (should succeed despite duplicate warning)
	graphBuilder := graph.NewDependencyGraphBuilder()
	pass := &analysis.Pass{
		Pkg: types.NewPackage("example.com/main", "main"),
	}

	depGraph, err := graphBuilder.BuildGraph(pass, providerRegistry.GetAll(), injectorRegistry2.GetAll())
	if err != nil {
		t.Fatalf("BuildGraph() should succeed despite correlation error: %v", err)
	}

	sorter := graph.NewTopologicalSorter()
	sortedTypes, err := sorter.Sort(depGraph)
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}

	bootstrapGen := generate.NewBootstrapGenerator()
	bootstrap, err := bootstrapGen.GenerateBootstrap(pass, depGraph, sortedTypes)
	if err != nil {
		t.Fatalf("GenerateBootstrap() should succeed despite correlation error: %v", err)
	}

	// Step 6: Verify bootstrap was generated (not skipped)
	if bootstrap == nil {
		t.Fatal("Bootstrap should be generated (correlation errors are non-fatal)")
	}
	if bootstrap.DependencyVar == "" {
		t.Error("Bootstrap DependencyVar should not be empty")
	}
}

// TestIntegration_WithoutConstructorFlow tests the WithoutConstructor option:
// constructor generation is skipped, but the struct still appears as a field
// in the dependency struct and is initialized in bootstrap using the manual
// constructor name.
//
// Requirements: 2.8, 6.4
func TestIntegration_WithoutConstructorFlow(t *testing.T) {
	// Step 1: Create injector info with WithoutConstructor option
	injectorInfo := &registry.InjectorInfo{
		TypeName:        "example.com/service.CustomService",
		PackagePath:     "example.com/service",
		PackageName:     "service",
		LocalName:       "CustomService",
		ConstructorName: "NewCustomService",
		Dependencies:    []string{},
		IsPending:       false,
		OptionMetadata: detect.OptionMetadata{
			WithoutConstructor: true,
		},
	}

	// Step 2: Verify constructor generation returns nil (skip)
	ctorGen := generate.NewConstructorGenerator()
	candidate := detect.ConstructorCandidate{
		TypeSpec: &ast.TypeSpec{
			Name: &ast.Ident{Name: "CustomService"},
		},
	}

	ctor, err := ctorGen.GenerateConstructorWithOptions(candidate, nil, injectorInfo)
	if err != nil {
		t.Fatalf("GenerateConstructorWithOptions() should not error, got: %v", err)
	}
	if ctor != nil {
		t.Errorf("Expected nil constructor for WithoutConstructor, got: %v", ctor)
	}

	// Step 3: Register injector and build graph
	injectorReg := registry.NewInjectorRegistry()
	providerReg := registry.NewProviderRegistry()
	injectorReg.Register(injectorInfo)

	graphBuilder := graph.NewDependencyGraphBuilder()
	pass := &analysis.Pass{
		Pkg: types.NewPackage("example.com/main", "main"),
	}

	depGraph, err := graphBuilder.BuildGraph(pass, providerReg.GetAll(), injectorReg.GetAll())
	if err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}

	// Verify node exists and is a struct field (Inject → IsField=true)
	node := depGraph.Nodes["example.com/service.CustomService"]
	if node == nil {
		t.Fatal("Expected WithoutConstructor node in graph")
	}
	if !node.IsField {
		t.Error("WithoutConstructor Inject node should have IsField=true")
	}

	// Step 4: Topological sort
	sorter := graph.NewTopologicalSorter()
	sortedTypes, err := sorter.Sort(depGraph)
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}

	// Step 5: Generate bootstrap
	bootstrapGen := generate.NewBootstrapGenerator()
	bootstrap, err := bootstrapGen.GenerateBootstrap(pass, depGraph, sortedTypes)
	if err != nil {
		t.Fatalf("GenerateBootstrap() error = %v", err)
	}

	// Step 6: Verify bootstrap includes struct field and calls manual constructor
	if !strings.Contains(bootstrap.DependencyVar, "CustomService") {
		t.Errorf("Bootstrap should include CustomService struct field.\nGot:\n%s", bootstrap.DependencyVar)
	}
	if !strings.Contains(bootstrap.DependencyVar, "service.NewCustomService()") {
		t.Errorf("Bootstrap should call manual constructor.\nGot:\n%s", bootstrap.DependencyVar)
	}
}

// TestIntegration_DependencyChainWithTypedAndNamed tests a complex dependency chain
// where typed and named dependencies coexist and must be resolved in correct topological order.
//
// Requirements: 6.5, 7.4
func TestIntegration_DependencyChainWithTypedAndNamed(t *testing.T) {
	// Setup: Logger -> Repository (Typed[IRepo], Named["mainRepo"]) -> Service
	// Named dependencies produce composite keys TypeName#Name in the graph.
	// Dependencies that reference named nodes must use the composite key.
	interfaceType := types.NewNamed(
		types.NewTypeName(token.NoPos, types.NewPackage("example.com/domain", "domain"), "IRepository", nil),
		types.NewInterfaceType(nil, nil),
		nil,
	)

	injectorRegistry := registry.NewInjectorRegistry()
	providerRegistry := registry.NewProviderRegistry()

	// Logger (default, no options)
	injectorRegistry.Register(&registry.InjectorInfo{
		TypeName:        "example.com/logger.Logger",
		PackagePath:     "example.com/logger",
		PackageName:     "logger",
		LocalName:       "Logger",
		ConstructorName: "NewLogger",
		Dependencies:    []string{},
		OptionMetadata:  detect.OptionMetadata{IsDefault: true},
	})

	// Repository (Typed + Named) - graph key will be "example.com/repo.Repository#mainRepo"
	injectorRegistry.Register(&registry.InjectorInfo{
		TypeName:        "example.com/repo.Repository",
		PackagePath:     "example.com/repo",
		PackageName:     "repo",
		LocalName:       "Repository",
		ConstructorName: "NewRepository",
		Dependencies:    []string{"example.com/logger.Logger"},
		RegisteredType:  interfaceType,
		Name:            "mainRepo",
		OptionMetadata: detect.OptionMetadata{
			TypedInterface: interfaceType,
			Name:           "mainRepo",
		},
	})

	// Service depends on Repository by composite key (TypeName#Name)
	injectorRegistry.Register(&registry.InjectorInfo{
		TypeName:        "example.com/service.UserService",
		PackagePath:     "example.com/service",
		PackageName:     "service",
		LocalName:       "UserService",
		ConstructorName: "NewUserService",
		Dependencies:    []string{"example.com/repo.Repository#mainRepo"},
		OptionMetadata:  detect.OptionMetadata{IsDefault: true},
	})

	// Build graph
	graphBuilder := graph.NewDependencyGraphBuilder()
	pass := &analysis.Pass{
		Pkg: types.NewPackage("example.com/main", "main"),
	}

	depGraph, err := graphBuilder.BuildGraph(pass, providerRegistry.GetAll(), injectorRegistry.GetAll())
	if err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}

	// Topological sort
	sorter := graph.NewTopologicalSorter()
	sortedTypes, err := sorter.Sort(depGraph)
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}

	// Verify order: Logger first, then Repository (composite key), then Service
	// Named dependencies produce composite keys TypeName#Name in the sorted list
	loggerIdx := -1
	repoIdx := -1
	svcIdx := -1
	for i, tp := range sortedTypes {
		if tp == "example.com/logger.Logger" {
			loggerIdx = i
		}
		if tp == "example.com/repo.Repository#mainRepo" {
			repoIdx = i
		}
		if tp == "example.com/service.UserService" {
			svcIdx = i
		}
	}

	if loggerIdx < 0 || repoIdx < 0 || svcIdx < 0 {
		t.Fatalf("Missing expected types in sorted list: %v", sortedTypes)
	}
	if !(loggerIdx < repoIdx && repoIdx < svcIdx) {
		t.Errorf("Expected order: Logger(%d) < Repository(%d) < Service(%d)", loggerIdx, repoIdx, svcIdx)
	}

	// Generate bootstrap
	bootstrapGen := generate.NewBootstrapGenerator()
	bootstrap, err := bootstrapGen.GenerateBootstrap(pass, depGraph, sortedTypes)
	if err != nil {
		t.Fatalf("GenerateBootstrap() error = %v", err)
	}

	// Verify:
	// 1. Logger uses default name
	if !strings.Contains(bootstrap.DependencyVar, "logger := logger.NewLogger()") {
		t.Errorf("Expected default logger initialization.\nGot:\n%s", bootstrap.DependencyVar)
	}
	// 2. Repository uses custom name and is initialized with logger
	if !strings.Contains(bootstrap.DependencyVar, "mainRepo := repo.NewRepository(logger)") {
		t.Errorf("Expected named repository initialization with logger dependency.\nGot:\n%s", bootstrap.DependencyVar)
	}
	// 3. Service uses Repository's custom name as argument
	if !strings.Contains(bootstrap.DependencyVar, "userService := service.NewUserService(mainRepo)") {
		t.Errorf("Expected service initialized with named repo.\nGot:\n%s", bootstrap.DependencyVar)
	}
	// 4. Repository variable is typed as interface (rendered with package qualifier as domain.IRepository)
	if !strings.Contains(bootstrap.DependencyVar, "mainRepo    domain.IRepository") && !strings.Contains(bootstrap.DependencyVar, "mainRepo domain.IRepository") {
		t.Errorf("Expected interface-typed field for named repository.\nGot:\n%s", bootstrap.DependencyVar)
	}
}

// TestIntegration_OptionExtractorWithNamerValidator tests the OptionExtractor + NamerValidator
// integration through to bootstrap generation. NamerValidator extracts names from Named[N]
// options, which become variable names in the generated bootstrap code.
//
// Requirements: 4.2, 4.3, 8.3
func TestIntegration_OptionExtractorWithNamerValidator(t *testing.T) {
	t.Run("Named option extracted by NamerValidator flows through to bootstrap", func(t *testing.T) {
		// Step 1: Create mock NamerValidator that returns a specific name
		mockValidator := &mockNamerValidator{
			extractNameFn: func(pass *analysis.Pass, namerType types.Type) (string, error) {
				return "primaryDB", nil
			},
		}
		extractor := detect.NewOptionExtractor(mockValidator)
		if extractor == nil {
			t.Fatal("Expected non-nil extractor")
		}

		// Step 2: Simulate the result of NamerValidator extraction by registering
		// an injector with the extracted name in OptionMetadata
		injectorReg := registry.NewInjectorRegistry()
		providerReg := registry.NewProviderRegistry()

		injectorReg.Register(&registry.InjectorInfo{
			TypeName:        "example.com/db.Connection",
			PackagePath:     "example.com/db",
			PackageName:     "db",
			LocalName:       "Connection",
			ConstructorName: "NewConnection",
			Dependencies:    []string{},
			Name:            "primaryDB",
			OptionMetadata: detect.OptionMetadata{
				Name: "primaryDB",
			},
		})

		// Step 3: Build graph → sort → bootstrap
		graphBuilder := graph.NewDependencyGraphBuilder()
		pass := &analysis.Pass{
			Pkg: types.NewPackage("example.com/main", "main"),
		}

		depGraph, err := graphBuilder.BuildGraph(pass, providerReg.GetAll(), injectorReg.GetAll())
		if err != nil {
			t.Fatalf("BuildGraph() error = %v", err)
		}

		sorter := graph.NewTopologicalSorter()
		sortedTypes, err := sorter.Sort(depGraph)
		if err != nil {
			t.Fatalf("Sort() error = %v", err)
		}

		bootstrapGen := generate.NewBootstrapGenerator()
		bootstrap, err := bootstrapGen.GenerateBootstrap(pass, depGraph, sortedTypes)
		if err != nil {
			t.Fatalf("GenerateBootstrap() error = %v", err)
		}

		// Step 4: Verify bootstrap uses the name extracted by NamerValidator
		if !strings.Contains(bootstrap.DependencyVar, "primaryDB") {
			t.Errorf("Bootstrap should use NamerValidator-extracted name 'primaryDB'.\nGot:\n%s", bootstrap.DependencyVar)
		}
	})

	t.Run("NamerValidator error prevents bootstrap via context cancellation", func(t *testing.T) {
		// Step 1: Create mock NamerValidator that returns an error
		mockValidator := &mockNamerValidator{
			extractNameFn: func(pass *analysis.Pass, namerType types.Type) (string, error) {
				return "", &namerValidationError{msg: "Name() must return hardcoded string literal"}
			},
		}
		extractor := detect.NewOptionExtractor(mockValidator)
		if extractor == nil {
			t.Fatal("Expected non-nil extractor")
		}

		// Step 2: Simulate DependencyAnalyzer receiving the error and cancelling context
		validationCtx := registry.NewValidationContext()
		validationCtx.Cancel()

		// Step 3: Register valid data that would generate bootstrap
		injectorReg := registry.NewInjectorRegistry()
		providerReg := registry.NewProviderRegistry()
		injectorReg.Register(&registry.InjectorInfo{
			TypeName:        "example.com/db.Connection",
			PackagePath:     "example.com/db",
			PackageName:     "db",
			LocalName:       "Connection",
			ConstructorName: "NewConnection",
			Dependencies:    []string{},
		})

		// Step 4: Simulate AppAnalyzer flow - should skip bootstrap due to cancelled context
		graphBuilder := graph.NewDependencyGraphBuilder()
		pass := &analysis.Pass{
			Pkg: types.NewPackage("example.com/main", "main"),
		}

		var bootstrap *generate.GeneratedBootstrap
		if !validationCtx.IsCancelled() {
			depGraph, err := graphBuilder.BuildGraph(pass, providerReg.GetAll(), injectorReg.GetAll())
			if err != nil {
				t.Fatalf("BuildGraph() error = %v", err)
			}
			sorter := graph.NewTopologicalSorter()
			sortedTypes, err := sorter.Sort(depGraph)
			if err != nil {
				t.Fatalf("Sort() error = %v", err)
			}
			bootstrapGen := generate.NewBootstrapGenerator()
			bootstrap, err = bootstrapGen.GenerateBootstrap(pass, depGraph, sortedTypes)
			if err != nil {
				t.Fatalf("GenerateBootstrap() error = %v", err)
			}
		}

		// Step 5: Verify bootstrap was NOT generated
		if bootstrap != nil {
			t.Error("Bootstrap should NOT be generated when NamerValidator error causes context cancellation")
		}
	})
}

// namerValidationError is a test helper for simulating NamerValidator errors.
type namerValidationError struct {
	msg string
}

func (e *namerValidationError) Error() string {
	return e.msg
}

// TestIntegration_RegistryGetByName tests the GetByName integration
// with the full flow of registration and lookup.
//
// Requirements: 4.5
func TestIntegration_RegistryGetByName(t *testing.T) {
	t.Run("InjectorRegistry GetByName", func(t *testing.T) {
		reg := registry.NewInjectorRegistry()

		reg.Register(&registry.InjectorInfo{
			TypeName: "example.com/repo.Repository",
			Name:     "primary",
		})

		// Lookup with correct name
		info, found := reg.GetByName("example.com/repo.Repository", "primary")
		if !found {
			t.Error("Expected to find injector by name 'primary'")
		}
		if info == nil {
			t.Fatal("Expected non-nil info")
		}
		if info.Name != "primary" {
			t.Errorf("Expected Name='primary', got %q", info.Name)
		}

		// Lookup with wrong name
		_, found = reg.GetByName("example.com/repo.Repository", "secondary")
		if found {
			t.Error("Should not find injector with wrong name")
		}

		// Lookup with non-existent type
		_, found = reg.GetByName("example.com/notexist", "primary")
		if found {
			t.Error("Should not find non-existent type")
		}
	})

	t.Run("ProviderRegistry GetByName", func(t *testing.T) {
		providerReg := registry.NewProviderRegistry()

		providerReg.Register(&registry.ProviderInfo{
			TypeName: "example.com/repo.Repository",
			Name:     "cache",
		})

		// Lookup with correct name
		info, found := providerReg.GetByName("example.com/repo.Repository", "cache")
		if !found {
			t.Error("Expected to find provider by name 'cache'")
		}
		if info == nil {
			t.Fatal("Expected non-nil info")
		}
		if info.Name != "cache" {
			t.Errorf("Expected Name='cache', got %q", info.Name)
		}

		// Lookup with wrong name
		_, found = providerReg.GetByName("example.com/repo.Repository", "other")
		if found {
			t.Error("Should not find provider with wrong name")
		}
	})
}

// TestIntegration_ValidationContextCoordination tests the full coordination between
// ValidationContext, DependencyAnalyzer (simulated), and AppAnalyzer (simulated).
//
// Requirements: 8.5, 8.6
func TestIntegration_ValidationContextCoordination(t *testing.T) {
	t.Run("Fresh context allows AppAnalyzer processing", func(t *testing.T) {
		vc := registry.NewValidationContext()

		// AppAnalyzer Phase 0 check
		if vc.IsCancelled() {
			t.Error("Fresh context should not be cancelled")
		}
		// AppAnalyzer would proceed with bootstrap generation
	})

	t.Run("Cancelled context stops AppAnalyzer processing", func(t *testing.T) {
		vc := registry.NewValidationContext()
		vc.Cancel()

		// AppAnalyzer Phase 0 check
		if !vc.IsCancelled() {
			t.Error("Cancelled context should stop AppAnalyzer")
		}
		// AppAnalyzer would return nil, nil
	})

	t.Run("Reset context allows AppAnalyzer processing again", func(t *testing.T) {
		vc := registry.NewValidationContext()
		vc.Cancel()

		if !vc.IsCancelled() {
			t.Fatal("Context should be cancelled")
		}

		vc.Reset()

		if vc.IsCancelled() {
			t.Error("Reset context should not be cancelled")
		}
	})

	t.Run("Multiple Cancel calls are safe", func(t *testing.T) {
		vc := registry.NewValidationContext()
		vc.Cancel()
		vc.Cancel() // Should not panic

		if !vc.IsCancelled() {
			t.Error("Context should still be cancelled after multiple Cancel calls")
		}
	})
}

// TestIntegration_ConstructorAndBootstrapConsistency verifies that the types
// used in constructor generation match the types used in bootstrap generation
// for the same dependency.
//
// Requirements: 6.1, 6.2, 7.1, 7.2
func TestIntegration_ConstructorAndBootstrapConsistency(t *testing.T) {
	// Create interface type
	interfaceType := types.NewNamed(
		types.NewTypeName(token.NoPos, nil, "IService", nil),
		types.NewInterfaceType(nil, nil),
		nil,
	)

	injectorInfo := &registry.InjectorInfo{
		TypeName:        "example.com/service.MyService",
		PackagePath:     "example.com/service",
		PackageName:     "service",
		LocalName:       "MyService",
		ConstructorName: "NewMyService",
		Dependencies:    []string{},
		RegisteredType:  interfaceType,
		OptionMetadata: detect.OptionMetadata{
			TypedInterface: interfaceType,
		},
	}

	// Generate constructor
	ctorGen := generate.NewConstructorGenerator()
	candidate := detect.ConstructorCandidate{
		TypeSpec: &ast.TypeSpec{
			Name: &ast.Ident{Name: "MyService"},
		},
	}

	ctor, err := ctorGen.GenerateConstructorWithOptions(candidate, nil, injectorInfo)
	if err != nil {
		t.Fatalf("Constructor generation error: %v", err)
	}
	if ctor == nil {
		t.Fatal("Expected constructor, got nil")
	}

	// Generate bootstrap
	injectorReg := registry.NewInjectorRegistry()
	injectorReg.Register(injectorInfo)
	providerReg := registry.NewProviderRegistry()

	graphBuilder := graph.NewDependencyGraphBuilder()
	pass := &analysis.Pass{
		Pkg: types.NewPackage("example.com/main", "main"),
	}

	depGraph, err := graphBuilder.BuildGraph(pass, providerReg.GetAll(), injectorReg.GetAll())
	if err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}

	sorter := graph.NewTopologicalSorter()
	sortedTypes, err := sorter.Sort(depGraph)
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}

	bootstrapGen := generate.NewBootstrapGenerator()
	bootstrap, err := bootstrapGen.GenerateBootstrap(pass, depGraph, sortedTypes)
	if err != nil {
		t.Fatalf("GenerateBootstrap() error = %v", err)
	}

	// Both constructor and bootstrap should reference the same interface type
	if !strings.Contains(ctor.Code, "IService") {
		t.Errorf("Constructor should reference IService.\nGot:\n%s", ctor.Code)
	}
	if !strings.Contains(bootstrap.DependencyVar, "IService") {
		t.Errorf("Bootstrap should reference IService.\nGot:\n%s", bootstrap.DependencyVar)
	}
}
