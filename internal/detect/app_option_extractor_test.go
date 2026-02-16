package detect_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"golang.org/x/tools/go/analysis"
)

// testAppMarkers holds the marker interfaces created from fake internal/annotation types.
// These are used for both createAppOptionTestPackages and manual type construction tests
// so that types.Implements checks work with consistent marker identities.
type testAppMarkers struct {
	internalAnnotationPkg *types.Package
	appContainerIface     *types.Interface // underlying interface of AppContainer
	appDefaultIface       *types.Interface // underlying interface of AppDefault
	appOptionIface        *types.Interface // underlying interface of AppOption
	markers               *detect.MarkerInterfaces
}

// createTestAppMarkers creates the internal/annotation marker types and returns them
// for use in both fake package construction and MarkerInterfaces.
// Marker interfaces use void methods (no return types) to match internal/annotation
// and ensure types.Implements works across separate packages.Load contexts.
func createTestAppMarkers() *testAppMarkers {
	internalAnnotationPkg := types.NewPackage("github.com/miyamo2/braider/internal/annotation", "annotation")

	// AppContainer interface: _IsAppContainerMarker()
	appContainerMethod := types.NewFunc(token.NoPos, internalAnnotationPkg, "_IsAppContainerMarker",
		types.NewSignatureType(nil, nil, nil, nil, nil, false))
	appContainerIface := types.NewInterfaceType([]*types.Func{appContainerMethod}, nil)
	appContainerIface.Complete()
	appContainerType := types.NewTypeName(token.NoPos, internalAnnotationPkg, "AppContainer", appContainerIface)
	internalAnnotationPkg.Scope().Insert(appContainerType)

	// AppDefault interface: _IsAppDefault()
	appDefaultMethod := types.NewFunc(token.NoPos, internalAnnotationPkg, "_IsAppDefault",
		types.NewSignatureType(nil, nil, nil, nil, nil, false))
	appDefaultIface := types.NewInterfaceType([]*types.Func{appDefaultMethod}, nil)
	appDefaultIface.Complete()
	appDefaultType := types.NewTypeName(token.NoPos, internalAnnotationPkg, "AppDefault", appDefaultIface)
	internalAnnotationPkg.Scope().Insert(appDefaultType)

	// AppOption interface: _IsAppOption()
	appOptionMethod := types.NewFunc(token.NoPos, internalAnnotationPkg, "_IsAppOption",
		types.NewSignatureType(nil, nil, nil, nil, nil, false))
	appOptionIface := types.NewInterfaceType([]*types.Func{appOptionMethod}, nil)
	appOptionIface.Complete()
	appOptionType := types.NewTypeName(token.NoPos, internalAnnotationPkg, "AppOption", appOptionIface)
	internalAnnotationPkg.Scope().Insert(appOptionType)

	internalAnnotationPkg.MarkComplete()

	return &testAppMarkers{
		internalAnnotationPkg: internalAnnotationPkg,
		appContainerIface:     appContainerIface,
		appDefaultIface:       appDefaultIface,
		appOptionIface:        appOptionIface,
		markers: &detect.MarkerInterfaces{
			AppDefault:   appDefaultIface,
			AppContainer: appContainerIface,
		},
	}
}

// createAppOptionTestPackages creates the fake packages needed for App option extractor tests.
// Returns both the package map (for the fake importer) and MarkerInterfaces (for the extractor).
func createAppOptionTestPackages() (map[string]*types.Package, *detect.MarkerInterfaces) {
	annotationPkg := createAnnotationPackageWithApp()
	appMarkers := createTestAppMarkers()

	// Create app options package
	appOptionsPkg := types.NewPackage("github.com/miyamo2/braider/pkg/annotation/app", "app")

	// app.Option interface (embeds annotation.AppOption)
	optionIface := types.NewInterfaceType(nil, []types.Type{appMarkers.appOptionIface})
	optionIface.Complete()
	optionType := types.NewTypeName(token.NoPos, appOptionsPkg, "Option", optionIface)
	appOptionsPkg.Scope().Insert(optionType)

	// app.Default interface (embeds Option + AppDefault)
	defaultIface := types.NewInterfaceType(nil, []types.Type{optionIface, appMarkers.appDefaultIface})
	defaultIface.Complete()
	defaultType := types.NewTypeName(token.NoPos, appOptionsPkg, "Default", defaultIface)
	appOptionsPkg.Scope().Insert(defaultType)

	// app.Container[T] - this is tricky with go/types as it's a generic interface.
	// For testing, we create a non-generic named type that embeds AppContainer.
	// The real Container[T] is generic, but for type checking in our tests,
	// we need a simulated version that the type checker can work with.
	containerIface := types.NewInterfaceType(nil, []types.Type{optionIface, appMarkers.appContainerIface})
	containerIface.Complete()
	containerType := types.NewTypeName(token.NoPos, appOptionsPkg, "Container", containerIface)
	appOptionsPkg.Scope().Insert(containerType)

	appOptionsPkg.MarkComplete()

	pkgs := map[string]*types.Package{
		detect.AnnotationPath:                             annotationPkg,
		"github.com/miyamo2/braider/internal/annotation": appMarkers.internalAnnotationPkg,
		"github.com/miyamo2/braider/pkg/annotation/app":  appOptionsPkg,
	}
	return pkgs, appMarkers.markers
}

// mockPassForAppOption creates a mock analysis.Pass for App option extractor tests.
func mockPassForAppOption(t *testing.T, src string, pkgs map[string]*types.Package) (*analysis.Pass, *ast.File) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse test source: %v", err)
	}

	conf := types.Config{
		Importer: &fakeAppImporter{
			packages: pkgs,
			fallback: importer.Default(),
		},
		Error: func(err error) {
			// Suppress type errors
		},
	}

	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	pkg, _ := conf.Check("main", fset, []*ast.File{file}, info)

	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       pkg,
		TypesInfo: info,
	}

	return pass, file
}

func TestAppOptionExtractor_NilTypeArgExpr(t *testing.T) {
	pkgs, markers := createAppOptionTestPackages()
	extractor := detect.NewAppOptionExtractorImpl(markers)

	src := `package main

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.App(main)

func main() {}
`
	pass, _ := mockPassForAppOption(t, src, pkgs)

	detector := detect.NewAppDetector(detect.ResolveMarkers())
	apps := detector.DetectAppAnnotations(pass)

	if len(apps) != 1 {
		t.Fatalf("expected 1 App annotation, got %d", len(apps))
	}

	// Non-generic call should have nil TypeArgExpr
	if apps[0].TypeArgExpr != nil {
		t.Fatal("expected nil TypeArgExpr for non-generic call")
	}

	metadata, err := extractor.ExtractAppOption(pass, apps[0])
	if err != nil {
		t.Fatalf("ExtractAppOption() returned error: %v", err)
	}

	if !metadata.IsDefault {
		t.Error("expected IsDefault=true for nil TypeArgExpr")
	}
	if metadata.ContainerDef != nil {
		t.Error("expected ContainerDef=nil for nil TypeArgExpr")
	}
}

func TestAppOptionExtractor_AppDefault(t *testing.T) {
	pkgs, markers := createAppOptionTestPackages()
	extractor := detect.NewAppOptionExtractorImpl(markers)

	src := `package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Default](main)

func main() {}
`
	pass, _ := mockPassForAppOption(t, src, pkgs)

	detector := detect.NewAppDetector(detect.ResolveMarkers())
	apps := detector.DetectAppAnnotations(pass)

	if len(apps) != 1 {
		t.Fatalf("expected 1 App annotation, got %d", len(apps))
	}

	if apps[0].TypeArgExpr == nil {
		t.Fatal("expected non-nil TypeArgExpr for generic call")
	}

	metadata, err := extractor.ExtractAppOption(pass, apps[0])
	if err != nil {
		t.Fatalf("ExtractAppOption() returned error: %v", err)
	}

	if !metadata.IsDefault {
		t.Error("expected IsDefault=true for app.Default")
	}
	if metadata.ContainerDef != nil {
		t.Error("expected ContainerDef=nil for app.Default")
	}
}

func TestAppOptionExtractor_ContainerWithNamedStruct(t *testing.T) {
	pkgs, markers := createAppOptionTestPackages()
	extractor := detect.NewAppOptionExtractorImpl(markers)

	// We need to test that when the type argument resolves to app.Container[NamedStruct],
	// we get a ContainerDefinition. Since we can't easily create generic instantiation
	// with go/types, we test the behavior through the type checker.
	// However, our fake app.Container is non-generic, so annotation.App[app.Container] will
	// be treated as a direct Container type implementing AppContainer.
	src := `package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[app.Container](main)

func main() {}
`
	pass, _ := mockPassForAppOption(t, src, pkgs)

	detector := detect.NewAppDetector(detect.ResolveMarkers())
	apps := detector.DetectAppAnnotations(pass)

	if len(apps) != 1 {
		t.Fatalf("expected 1 App annotation, got %d", len(apps))
	}

	if apps[0].TypeArgExpr == nil {
		t.Fatal("expected non-nil TypeArgExpr for generic call")
	}

	metadata, err := extractor.ExtractAppOption(pass, apps[0])

	// The Container type in our fake package has no type arguments (not generic),
	// so extractContainerDefinition will find it as Container name but typeArgs will be empty.
	// This means it will be treated as unresolvable. The test validates the detection path.
	// In a real scenario, Container[T] would be a generic instantiation with T as a struct.
	if err != nil {
		// Expected: the non-generic Container type can't provide a struct type parameter
		t.Logf("Expected behavior - Container without type args: %v", err)
	} else {
		// If it somehow resolves, check the result
		t.Logf("metadata: IsDefault=%v, ContainerDef=%v", metadata.IsDefault, metadata.ContainerDef)
	}
}

func TestAppOptionExtractor_DirectAppAnnotation(t *testing.T) {
	// Test with a manually constructed AppAnnotation to test ExtractAppOption
	// with controlled TypeArgExpr
	appMarkers := createTestAppMarkers()
	extractor := detect.NewAppOptionExtractorImpl(appMarkers.markers)

	// Test nil TypeArgExpr
	metadata, err := extractor.ExtractAppOption(&analysis.Pass{
		TypesInfo: &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
		},
	}, &detect.AppAnnotation{
		TypeArgExpr: nil,
	})

	if err != nil {
		t.Fatalf("ExtractAppOption() returned error: %v", err)
	}

	if !metadata.IsDefault {
		t.Error("expected IsDefault=true for nil TypeArgExpr")
	}
}

func TestAppOptionExtractor_UnresolvableTypeArg(t *testing.T) {
	appMarkers := createTestAppMarkers()
	extractor := detect.NewAppOptionExtractorImpl(appMarkers.markers)

	// Create a TypeArgExpr that doesn't resolve to anything in TypesInfo
	fakeExpr := &ast.Ident{Name: "Unknown"}

	metadata, err := extractor.ExtractAppOption(&analysis.Pass{
		TypesInfo: &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
		},
	}, &detect.AppAnnotation{
		TypeArgExpr: fakeExpr,
	})

	if err != nil {
		t.Fatalf("ExtractAppOption() returned error: %v", err)
	}

	// Unresolvable type defaults to IsDefault
	if !metadata.IsDefault {
		t.Error("expected IsDefault=true for unresolvable type argument")
	}
}

func TestAppOptionExtractor_MixedOptionWithDefault(t *testing.T) {
	pkgs, markers := createAppOptionTestPackages()
	extractor := detect.NewAppOptionExtractorImpl(markers)

	// Test mixed option that embeds app.Default
	src := `package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[interface{ app.Default }](main)

func main() {}
`
	pass, _ := mockPassForAppOption(t, src, pkgs)

	detector := detect.NewAppDetector(detect.ResolveMarkers())
	apps := detector.DetectAppAnnotations(pass)

	if len(apps) != 1 {
		t.Fatalf("expected 1 App annotation, got %d", len(apps))
	}

	if apps[0].TypeArgExpr == nil {
		t.Fatal("expected non-nil TypeArgExpr")
	}

	metadata, err := extractor.ExtractAppOption(pass, apps[0])
	if err != nil {
		t.Fatalf("ExtractAppOption() returned error: %v", err)
	}

	if !metadata.IsDefault {
		t.Error("expected IsDefault=true for interface{ app.Default }")
	}
	if metadata.ContainerDef != nil {
		t.Error("expected ContainerDef=nil for interface{ app.Default }")
	}
}

func TestAppOptionExtractor_MixedOptionWithContainer(t *testing.T) {
	pkgs, markers := createAppOptionTestPackages()
	extractor := detect.NewAppOptionExtractorImpl(markers)

	// Test mixed option that embeds app.Container
	src := `package main

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/app"
)

var _ = annotation.App[interface{ app.Container }](main)

func main() {}
`
	pass, _ := mockPassForAppOption(t, src, pkgs)

	detector := detect.NewAppDetector(detect.ResolveMarkers())
	apps := detector.DetectAppAnnotations(pass)

	if len(apps) != 1 {
		t.Fatalf("expected 1 App annotation, got %d", len(apps))
	}

	if apps[0].TypeArgExpr == nil {
		t.Fatal("expected non-nil TypeArgExpr")
	}

	metadata, err := extractor.ExtractAppOption(pass, apps[0])

	// Container without type args won't produce a valid ContainerDef
	// because the non-generic Container type has no type arguments to extract.
	// In production, Container[T] would be instantiated with a struct type.
	t.Logf("metadata: IsDefault=%v, ContainerDef=%v, err=%v", metadata.IsDefault, metadata.ContainerDef, err)

	// The key assertion: it should not be treated as Default
	if metadata.IsDefault {
		t.Error("mixed option with Container embedding should not be treated as Default")
	}
}

func TestAppOptionExtractor_ContainerWithNamedStruct_FullIntegration(t *testing.T) {
	// This test uses manually constructed types to test buildContainerDefinition
	// through ExtractAppOption.

	appMarkers := createTestAppMarkers()
	extractor := detect.NewAppOptionExtractorImpl(appMarkers.markers)

	// Create a package with a named struct type to use as container
	containerPkg := types.NewPackage("example.com/myapp", "myapp")
	handlerField := types.NewField(token.NoPos, containerPkg, "Handler", types.Typ[types.String], false)
	repoField := types.NewField(token.NoPos, containerPkg, "Repo", types.Typ[types.Int], false)
	containerStruct := types.NewStruct(
		[]*types.Var{handlerField, repoField},
		[]string{`braider:"handler"`, ``},
	)
	containerTypeName := types.NewTypeName(token.NoPos, containerPkg, "MyContainer", nil)
	containerNamed := types.NewNamed(containerTypeName, containerStruct, nil)
	containerPkg.Scope().Insert(containerTypeName)
	containerPkg.MarkComplete()

	// Create a fake app.Container[MyContainer] named type with the container as type arg.
	// The underlying interface embeds AppContainer so types.Implements works.
	appOptionsPkg := types.NewPackage("github.com/miyamo2/braider/pkg/annotation/app", "app")
	containerIfaceName := types.NewTypeName(token.NoPos, appOptionsPkg, "Container", nil)

	// Create a Named type with type parameters and arguments to simulate Container[T]
	tparam := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, "T", nil), types.NewInterfaceType(nil, nil))
	containerUnderlying := types.NewInterfaceType(nil, []types.Type{appMarkers.appContainerIface})
	containerUnderlying.Complete()
	containerNamedGeneric := types.NewNamed(containerIfaceName, containerUnderlying, nil)
	containerNamedGeneric.SetTypeParams([]*types.TypeParam{tparam})

	// Instantiate Container[MyContainer]
	instantiated, err := types.Instantiate(nil, containerNamedGeneric, []types.Type{containerNamed}, false)
	if err != nil {
		t.Fatalf("failed to instantiate Container[MyContainer]: %v", err)
	}

	// Create a minimal pass
	fset := token.NewFileSet()
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}
	mainPkg := types.NewPackage("main", "main")
	pass := &analysis.Pass{
		Fset:      fset,
		Pkg:       mainPkg,
		TypesInfo: info,
	}

	// Create a fake AppAnnotation with a fake TypeArgExpr that resolves to the instantiated type
	fakeExpr := &ast.Ident{Name: "Container"}
	info.Types[fakeExpr] = types.TypeAndValue{Type: instantiated}

	app := &detect.AppAnnotation{
		TypeArgExpr: fakeExpr,
		Pos:         token.NoPos,
	}

	metadata, extractErr := extractor.ExtractAppOption(pass, app)
	if extractErr != nil {
		t.Fatalf("ExtractAppOption() returned error: %v", extractErr)
	}

	if metadata.IsDefault {
		t.Error("expected IsDefault=false for Container[MyContainer]")
	}

	if metadata.ContainerDef == nil {
		t.Fatal("expected ContainerDef to be non-nil")
	}

	def := metadata.ContainerDef

	if !def.IsNamed {
		t.Error("expected IsNamed=true for named struct")
	}

	if def.LocalName != "MyContainer" {
		t.Errorf("expected LocalName='MyContainer', got %q", def.LocalName)
	}

	if def.PackagePath != "example.com/myapp" {
		t.Errorf("expected PackagePath='example.com/myapp', got %q", def.PackagePath)
	}

	if def.PackageName != "myapp" {
		t.Errorf("expected PackageName='myapp', got %q", def.PackageName)
	}

	if def.TypeName != "example.com/myapp.MyContainer" {
		t.Errorf("expected TypeName='example.com/myapp.MyContainer', got %q", def.TypeName)
	}

	if len(def.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(def.Fields))
	}

	if def.Fields[0].Name != "Handler" {
		t.Errorf("expected first field name 'Handler', got %q", def.Fields[0].Name)
	}
	if def.Fields[0].Tag != "handler" {
		t.Errorf("expected first field tag 'handler', got %q", def.Fields[0].Tag)
	}

	if def.Fields[1].Name != "Repo" {
		t.Errorf("expected second field name 'Repo', got %q", def.Fields[1].Name)
	}
	if def.Fields[1].Tag != "" {
		t.Errorf("expected second field tag '', got %q", def.Fields[1].Tag)
	}
}

func TestAppOptionExtractor_ContainerWithAnonymousStruct(t *testing.T) {
	appMarkers := createTestAppMarkers()
	extractor := detect.NewAppOptionExtractorImpl(appMarkers.markers)

	// Create an anonymous struct type
	handlerField := types.NewField(token.NoPos, nil, "handler", types.Typ[types.String], false)
	anonStruct := types.NewStruct(
		[]*types.Var{handlerField},
		[]string{`braider:"handler"`},
	)

	// Create a fake app.Container[struct{...}] with the anonymous struct as type arg.
	// The underlying interface embeds AppContainer so types.Implements works.
	appOptionsPkg := types.NewPackage("github.com/miyamo2/braider/pkg/annotation/app", "app")
	containerIfaceName := types.NewTypeName(token.NoPos, appOptionsPkg, "Container", nil)

	tparam := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, "T", nil), types.NewInterfaceType(nil, nil))
	containerUnderlying := types.NewInterfaceType(nil, []types.Type{appMarkers.appContainerIface})
	containerUnderlying.Complete()
	containerNamedGeneric := types.NewNamed(containerIfaceName, containerUnderlying, nil)
	containerNamedGeneric.SetTypeParams([]*types.TypeParam{tparam})

	// Instantiate Container[struct{handler string}]
	instantiated, err := types.Instantiate(nil, containerNamedGeneric, []types.Type{anonStruct}, false)
	if err != nil {
		t.Fatalf("failed to instantiate Container[struct]: %v", err)
	}

	// Create a minimal pass
	fset := token.NewFileSet()
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}
	mainPkg := types.NewPackage("main", "main")
	pass := &analysis.Pass{
		Fset:      fset,
		Pkg:       mainPkg,
		TypesInfo: info,
	}

	fakeExpr := &ast.Ident{Name: "Container"}
	info.Types[fakeExpr] = types.TypeAndValue{Type: instantiated}

	app := &detect.AppAnnotation{
		TypeArgExpr: fakeExpr,
		Pos:         token.NoPos,
	}

	metadata, extractErr := extractor.ExtractAppOption(pass, app)
	if extractErr != nil {
		t.Fatalf("ExtractAppOption() returned error: %v", extractErr)
	}

	if metadata.IsDefault {
		t.Error("expected IsDefault=false for Container[struct{...}]")
	}

	if metadata.ContainerDef == nil {
		t.Fatal("expected ContainerDef to be non-nil")
	}

	def := metadata.ContainerDef

	if def.IsNamed {
		t.Error("expected IsNamed=false for anonymous struct")
	}

	if len(def.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(def.Fields))
	}

	if def.Fields[0].Name != "handler" {
		t.Errorf("expected field name 'handler', got %q", def.Fields[0].Name)
	}
	if def.Fields[0].Tag != "handler" {
		t.Errorf("expected field tag 'handler', got %q", def.Fields[0].Tag)
	}
}

func TestAppOptionExtractor_MixedOptionContainerDef(t *testing.T) {
	appMarkers := createTestAppMarkers()
	extractor := detect.NewAppOptionExtractorImpl(appMarkers.markers)

	// Create a named struct for the container
	containerPkg := types.NewPackage("example.com/myapp", "myapp")
	svcField := types.NewField(token.NoPos, containerPkg, "Service", types.Typ[types.Int], false)
	containerStruct := types.NewStruct([]*types.Var{svcField}, []string{``})
	containerTypeName := types.NewTypeName(token.NoPos, containerPkg, "AppContainer", nil)
	containerNamed := types.NewNamed(containerTypeName, containerStruct, nil)
	containerPkg.Scope().Insert(containerTypeName)
	containerPkg.MarkComplete()

	// Create app.Container[AppContainer] as an instantiated generic type.
	// The underlying interface embeds AppContainer so types.Implements works.
	appOptionsPkg := types.NewPackage("github.com/miyamo2/braider/pkg/annotation/app", "app")
	containerIfaceName := types.NewTypeName(token.NoPos, appOptionsPkg, "Container", nil)
	tparam := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, "T", nil), types.NewInterfaceType(nil, nil))
	containerUnderlying := types.NewInterfaceType(nil, []types.Type{appMarkers.appContainerIface})
	containerUnderlying.Complete()
	containerNamedGeneric := types.NewNamed(containerIfaceName, containerUnderlying, nil)
	containerNamedGeneric.SetTypeParams([]*types.TypeParam{tparam})

	instantiated, err := types.Instantiate(nil, containerNamedGeneric, []types.Type{containerNamed}, false)
	if err != nil {
		t.Fatalf("failed to instantiate Container[AppContainer]: %v", err)
	}

	// Create mixed option: interface{ Container[AppContainer] }
	// Simulate by creating an interface that embeds the instantiated type
	mixedIface := types.NewInterfaceType(nil, []types.Type{instantiated})
	mixedIface.Complete()

	// Create a minimal pass
	fset := token.NewFileSet()
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}
	mainPkg := types.NewPackage("main", "main")
	pass := &analysis.Pass{
		Fset:      fset,
		Pkg:       mainPkg,
		TypesInfo: info,
	}

	fakeExpr := &ast.Ident{Name: "MixedOption"}
	info.Types[fakeExpr] = types.TypeAndValue{Type: mixedIface}

	app := &detect.AppAnnotation{
		TypeArgExpr: fakeExpr,
		Pos:         token.NoPos,
	}

	metadata, extractErr := extractor.ExtractAppOption(pass, app)
	if extractErr != nil {
		t.Fatalf("ExtractAppOption() returned error: %v", extractErr)
	}

	if metadata.IsDefault {
		t.Error("expected IsDefault=false for mixed option with Container")
	}

	if metadata.ContainerDef == nil {
		t.Fatal("expected ContainerDef to be non-nil for mixed option with Container")
	}

	def := metadata.ContainerDef
	if !def.IsNamed {
		t.Error("expected IsNamed=true for named struct in mixed option")
	}
	if def.LocalName != "AppContainer" {
		t.Errorf("expected LocalName='AppContainer', got %q", def.LocalName)
	}
	if len(def.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(def.Fields))
	}
	if def.Fields[0].Name != "Service" {
		t.Errorf("expected field name 'Service', got %q", def.Fields[0].Name)
	}
}
