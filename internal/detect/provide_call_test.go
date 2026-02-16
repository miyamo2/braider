package detect_test

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"golang.org/x/tools/go/analysis"
)

// createAnnotationPackageWithProvide creates a fake annotation package with a non-generic Provide function.
// The function signature is: func Provide(any) Provider
// where Provider is a named type in the annotation package embedding the internal marker interface.
func createAnnotationPackageWithProvide() *types.Package {
	// Create synthetic internal/annotation marker interface for Provider
	internalPkg := types.NewPackage("github.com/miyamo2/braider/internal/annotation", "annotation")
	markerSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	markerMethod := types.NewFunc(token.NoPos, internalPkg, "_IsProvider", markerSig)
	markerIface := types.NewInterfaceType([]*types.Func{markerMethod}, nil)
	markerIface.Complete()
	markerTypeName := types.NewTypeName(token.NoPos, internalPkg, "Provider", nil)
	markerNamed := types.NewNamed(markerTypeName, markerIface, nil)
	internalPkg.Scope().Insert(markerNamed.Obj())
	internalPkg.MarkComplete()

	// Create pkg/annotation package
	annotationPkg := types.NewPackage(detect.AnnotationPath, "annotation")

	// Create the Provider named type embedding the internal marker interface
	embeddedField := types.NewField(token.NoPos, nil, "", markerNamed, true)
	providerStruct := types.NewStruct([]*types.Var{embeddedField}, nil)
	providerNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, annotationPkg, "Provider", nil),
		providerStruct,
		nil,
	)
	annotationPkg.Scope().Insert(providerNamed.Obj())

	// Create the Provide function: func(any) Provider
	anyType := types.Universe.Lookup("any").Type()
	provideSig := types.NewSignatureType(
		nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "providerFunc", anyType)),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", providerNamed)),
		false,
	)
	provideFunc := types.NewFunc(token.NoPos, annotationPkg, detect.ProvideTypeName, provideSig)
	annotationPkg.Scope().Insert(provideFunc)

	annotationPkg.MarkComplete()
	return annotationPkg
}

func TestProvideCallDetector_DetectProviders(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	tests := []struct {
		name          string
		src           string
		pkgs          map[string]*types.Package
		expectedCount int
	}{
		{
			name: "valid annotation.Provide(NewRepo)",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

func NewRepo() *MyRepo { return nil }

type MyRepo struct{}

var _ = annotation.Provide(NewRepo)
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 1,
		},
		{
			name: "multiple Provide calls",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

func NewRepo() *MyRepo { return nil }
func NewService() *MyService { return nil }

type MyRepo struct{}
type MyService struct{}

var _ = annotation.Provide(NewRepo)
var _ = annotation.Provide(NewService)
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 2,
		},
		{
			name: "no Provide calls",
			src: `package test

func NewRepo() {}

type MyRepo struct{}
`,
			pkgs:          nil,
			expectedCount: 0,
		},
		{
			name: "wrong function name annotation.NotProvide",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

func NewRepo() *MyRepo { return nil }

type MyRepo struct{}

var _ = annotation.Provider(NewRepo)
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 0,
		},
		{
			name: "value is not call expression",
			src: `package test

var _ = 42
`,
			pkgs:          nil,
			expectedCount: 0,
		},
		{
			name: "aliased import",
			src: `package test

import ann "github.com/miyamo2/braider/pkg/annotation"

func NewRepo() *MyRepo { return nil }

type MyRepo struct{}

var _ = ann.Provide(NewRepo)
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 1,
		},
		{
			name: "wrong package path",
			src: `package test

import "github.com/other/annotation"

func NewRepo() *MyRepo { return nil }

type MyRepo struct{}

var _ = annotation.Provide(NewRepo)
`,
			pkgs: map[string]*types.Package{
				"github.com/other/annotation": func() *types.Package {
					wrongPkg := types.NewPackage("github.com/other/annotation", "annotation")
					// Add a Provide function to the wrong package, but return type is from wrong package
					providerStruct := types.NewStruct(nil, nil)
					providerNamed := types.NewNamed(
						types.NewTypeName(token.NoPos, wrongPkg, "Provider", nil),
						providerStruct,
						nil,
					)
					wrongPkg.Scope().Insert(providerNamed.Obj())
					anyType := types.Universe.Lookup("any").Type()
					provideSig := types.NewSignatureType(
						nil, nil, nil,
						types.NewTuple(types.NewVar(token.NoPos, nil, "providerFunc", anyType)),
						types.NewTuple(types.NewVar(token.NoPos, nil, "", providerNamed)),
						false,
					)
					provideFunc := types.NewFunc(token.NoPos, wrongPkg, detect.ProvideTypeName, provideSig)
					wrongPkg.Scope().Insert(provideFunc)
					wrongPkg.MarkComplete()
					return wrongPkg
				}(),
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				pass, _ := mockPass(t, tt.src, tt.pkgs)

				detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
				candidates := detector.DetectProviders(pass)

				if len(candidates) != tt.expectedCount {
					t.Errorf("DetectProviders() returned %d candidates, want %d", len(candidates), tt.expectedCount)
				}
			},
		)
	}
}

func TestProvideCallDetector_DetectProviders_WithInspector(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	tests := []struct {
		name          string
		src           string
		pkgs          map[string]*types.Package
		expectedCount int
	}{
		{
			name: "valid Provide via Inspector",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

func NewRepo() *MyRepo { return nil }

type MyRepo struct{}

var _ = annotation.Provide(NewRepo)
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 1,
		},
		{
			name: "multiple Provides via Inspector",
			src: `package test

import "github.com/miyamo2/braider/pkg/annotation"

func NewRepo() *MyRepo { return nil }
func NewService() *MyService { return nil }

type MyRepo struct{}
type MyService struct{}

var _ = annotation.Provide(NewRepo)
var _ = annotation.Provide(NewService)
`,
			pkgs:          map[string]*types.Package{detect.AnnotationPath: annotationPkg},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				pass, _ := mockPassWithInspector(t, tt.src, tt.pkgs)

				detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
				candidates := detector.DetectProviders(pass)

				if len(candidates) != tt.expectedCount {
					t.Errorf(
						"DetectProviders() with Inspector returned %d candidates, want %d",
						len(candidates),
						tt.expectedCount,
					)
				}
			},
		)
	}
}

func TestProvideCallDetector_DetectProviders_CandidateFields(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

func NewRepo() *MyRepo { return nil }

type MyRepo struct{}

var _ = annotation.Provide(NewRepo)
`
	pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
	pass, _ := mockPass(t, src, pkgs)

	detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
	candidates := detector.DetectProviders(pass)

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	c := candidates[0]

	// Verify CallExpr is set
	if c.CallExpr == nil {
		t.Error("ProviderCandidate.CallExpr should not be nil")
	}

	// Verify ProviderFunc is set
	if c.ProviderFunc == nil {
		t.Error("ProviderCandidate.ProviderFunc should not be nil")
	}

	// Verify ProviderFuncSig is set
	if c.ProviderFuncSig == nil {
		t.Error("ProviderCandidate.ProviderFuncSig should not be nil")
	}

	// Verify ProviderFuncName
	if c.ProviderFuncName != "NewRepo" {
		t.Errorf("ProviderFuncName = %q, want %q", c.ProviderFuncName, "NewRepo")
	}

	// Verify ReturnType is set
	if c.ReturnType == nil {
		t.Error("ProviderCandidate.ReturnType should not be nil")
	}

	// Verify ReturnTypeName - pointer return type should be dereferenced
	if c.ReturnTypeName != "MyRepo" {
		t.Errorf("ReturnTypeName = %q, want %q", c.ReturnTypeName, "MyRepo")
	}

	// Verify PackagePath matches pass.Pkg.Path() (return type is in the same package)
	if c.PackagePath != pass.Pkg.Path() {
		t.Errorf("PackagePath = %q, want %q", c.PackagePath, pass.Pkg.Path())
	}
}

func TestProvideCallDetector_DetectProviders_EdgeCases(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	t.Run(
		"function returning pointer type", func(t *testing.T) {
			src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

func NewRepo() *MyRepo { return nil }

type MyRepo struct{}

var _ = annotation.Provide(NewRepo)
`
			pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
			pass, _ := mockPass(t, src, pkgs)

			detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
			candidates := detector.DetectProviders(pass)

			if len(candidates) != 1 {
				t.Fatalf("expected 1 candidate, got %d", len(candidates))
			}

			// ReturnTypeName should be base type name (not pointer)
			if candidates[0].ReturnTypeName != "MyRepo" {
				t.Errorf("ReturnTypeName = %q, want %q", candidates[0].ReturnTypeName, "MyRepo")
			}
		},
	)

	t.Run(
		"selector expression function (pkg.NewRepo)", func(t *testing.T) {
			// This tests extractFuncName with SelectorExpr path.
			// We create a package with a function, then use pkg.Func as the Provide argument.
			helperPkg := types.NewPackage("example.com/helper", "helper")
			repoStruct := types.NewStruct(nil, nil)
			repoNamed := types.NewNamed(
				types.NewTypeName(token.NoPos, helperPkg, "Repo", nil),
				repoStruct,
				nil,
			)
			helperPkg.Scope().Insert(repoNamed.Obj())
			newRepoSig := types.NewSignatureType(
				nil, nil, nil,
				nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewPointer(repoNamed))),
				false,
			)
			newRepoFunc := types.NewFunc(token.NoPos, helperPkg, "NewRepo", newRepoSig)
			helperPkg.Scope().Insert(newRepoFunc)
			helperPkg.MarkComplete()

			src := `package test

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"example.com/helper"
)

var _ = annotation.Provide(helper.NewRepo)
`
			pkgs := map[string]*types.Package{
				detect.AnnotationPath: annotationPkg,
				"example.com/helper":  helperPkg,
			}
			pass, _ := mockPass(t, src, pkgs)

			detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
			candidates := detector.DetectProviders(pass)

			if len(candidates) != 1 {
				t.Fatalf("expected 1 candidate, got %d", len(candidates))
			}

			// SelectorExpr function name extraction should return the selector name
			if candidates[0].ProviderFuncName != "NewRepo" {
				t.Errorf("ProviderFuncName = %q, want %q", candidates[0].ProviderFuncName, "NewRepo")
			}

			// PackagePath should be the return type's package, not the calling package
			if candidates[0].PackagePath != "example.com/helper" {
				t.Errorf("PackagePath = %q, want %q", candidates[0].PackagePath, "example.com/helper")
			}
		},
	)

	t.Run(
		"no arguments in call", func(t *testing.T) {
			// annotation.Provide() with no args - should not produce a candidate
			// We simulate this through the call detection; since our fake Provide
			// requires an argument, type-checking makes this fail at isProvideCall.
			// So the result should be 0 candidates.
			src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

var _ = annotation.Provide()
`
			pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
			pass, _ := mockPass(t, src, pkgs)

			detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
			candidates := detector.DetectProviders(pass)

			// With no arguments, the call might fail type-checking, resulting in 0 candidates
			if len(candidates) != 0 {
				t.Errorf("expected 0 candidates for Provide() with no args, got %d", len(candidates))
			}
		},
	)

	t.Run(
		"argument is not a function", func(t *testing.T) {
			src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

var myVar = 42

var _ = annotation.Provide(myVar)
`
			pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
			pass, _ := mockPass(t, src, pkgs)

			detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
			candidates := detector.DetectProviders(pass)

			// Non-function argument: extractCandidate checks for *types.Signature, returns nil
			if len(candidates) != 0 {
				t.Errorf("expected 0 candidates for non-function argument, got %d", len(candidates))
			}
		},
	)
}

func TestProvideCallDetector_DetectImplementedInterfaces(t *testing.T) {
	tests := []struct {
		name           string
		setupPkgs      func() (map[string]*types.Package, *types.Named)
		src            string
		expectedCount  int
		expectedIfaces []string
	}{
		{
			name: "implements imported interface",
			setupPkgs: func() (map[string]*types.Package, *types.Named) {
				// Create interface package
				ifacePkg := types.NewPackage("example.com/iface", "iface")
				method := types.NewFunc(
					token.NoPos, ifacePkg, "DoWork", types.NewSignatureType(
						nil, nil, nil, nil, nil, false,
					),
				)
				iface := types.NewInterfaceType([]*types.Func{method}, nil)
				iface.Complete()
				ifaceNamed := types.NewNamed(
					types.NewTypeName(token.NoPos, ifacePkg, "Worker", nil),
					iface,
					nil,
				)
				ifacePkg.Scope().Insert(ifaceNamed.Obj())
				ifacePkg.MarkComplete()

				// Create test package with struct that implements the interface
				testPkg := types.NewPackage("test", "test")
				implStruct := types.NewStruct(nil, nil)
				implNamed := types.NewNamed(
					types.NewTypeName(token.NoPos, testPkg, "MyWorker", nil),
					implStruct,
					nil,
				)
				doWorkMethod := types.NewFunc(
					token.NoPos, testPkg, "DoWork", types.NewSignatureType(
						types.NewVar(token.NoPos, testPkg, "", implNamed), nil, nil, nil, nil, false,
					),
				)
				implNamed.AddMethod(doWorkMethod)
				testPkg.Scope().Insert(implNamed.Obj())

				// Make testPkg import ifacePkg
				testPkg.SetImports([]*types.Package{ifacePkg})
				testPkg.MarkComplete()

				return map[string]*types.Package{
					"example.com/iface": ifacePkg,
				}, implNamed
			},
			src: `package test

import "example.com/iface"

type MyWorker struct{}

func (MyWorker) DoWork() {}

var _ iface.Worker = MyWorker{}
`,
			expectedCount:  1,
			expectedIfaces: []string{"example.com/iface.Worker"},
		},
		{
			name: "implements interface in current package",
			setupPkgs: func() (map[string]*types.Package, *types.Named) {
				testPkg := types.NewPackage("test", "test")

				// Create interface in current package
				method := types.NewFunc(
					token.NoPos, testPkg, "Run", types.NewSignatureType(
						nil, nil, nil, nil, nil, false,
					),
				)
				iface := types.NewInterfaceType([]*types.Func{method}, nil)
				iface.Complete()
				ifaceNamed := types.NewNamed(
					types.NewTypeName(token.NoPos, testPkg, "Runner", nil),
					iface,
					nil,
				)
				testPkg.Scope().Insert(ifaceNamed.Obj())

				// Create implementing struct
				implStruct := types.NewStruct(nil, nil)
				implNamed := types.NewNamed(
					types.NewTypeName(token.NoPos, testPkg, "MyRunner", nil),
					implStruct,
					nil,
				)
				runMethod := types.NewFunc(
					token.NoPos, testPkg, "Run", types.NewSignatureType(
						types.NewVar(token.NoPos, testPkg, "", implNamed), nil, nil, nil, nil, false,
					),
				)
				implNamed.AddMethod(runMethod)
				testPkg.Scope().Insert(implNamed.Obj())

				testPkg.MarkComplete()

				return nil, implNamed
			},
			src: `package test

type Runner interface {
	Run()
}

type MyRunner struct{}

func (MyRunner) Run() {}
`,
			expectedCount:  1,
			expectedIfaces: []string{"test.Runner"},
		},
		{
			name: "does not implement interface",
			setupPkgs: func() (map[string]*types.Package, *types.Named) {
				ifacePkg := types.NewPackage("example.com/iface", "iface")
				method := types.NewFunc(
					token.NoPos, ifacePkg, "DoWork", types.NewSignatureType(
						nil, nil, nil, nil, nil, false,
					),
				)
				iface := types.NewInterfaceType([]*types.Func{method}, nil)
				iface.Complete()
				ifaceNamed := types.NewNamed(
					types.NewTypeName(token.NoPos, ifacePkg, "Worker", nil),
					iface,
					nil,
				)
				ifacePkg.Scope().Insert(ifaceNamed.Obj())
				ifacePkg.MarkComplete()

				// Struct WITHOUT the method
				testPkg := types.NewPackage("test", "test")
				implStruct := types.NewStruct(nil, nil)
				implNamed := types.NewNamed(
					types.NewTypeName(token.NoPos, testPkg, "NotAWorker", nil),
					implStruct,
					nil,
				)
				testPkg.Scope().Insert(implNamed.Obj())
				testPkg.SetImports([]*types.Package{ifacePkg})
				testPkg.MarkComplete()

				return map[string]*types.Package{
					"example.com/iface": ifacePkg,
				}, implNamed
			},
			src: `package test

import "example.com/iface"

type NotAWorker struct{}

var _ iface.Worker
`,
			expectedCount: 0,
		},
		{
			name: "nil TypesInfo",
			setupPkgs: func() (map[string]*types.Package, *types.Named) {
				testPkg := types.NewPackage("test", "test")
				implStruct := types.NewStruct(nil, nil)
				implNamed := types.NewNamed(
					types.NewTypeName(token.NoPos, testPkg, "MyType", nil),
					implStruct,
					nil,
				)
				testPkg.Scope().Insert(implNamed.Obj())
				testPkg.MarkComplete()
				return nil, implNamed
			},
			src:           `package test`,
			expectedCount: 0,
		},
		{
			name: "pointer receiver implements interface",
			setupPkgs: func() (map[string]*types.Package, *types.Named) {
				ifacePkg := types.NewPackage("example.com/iface", "iface")
				method := types.NewFunc(
					token.NoPos, ifacePkg, "Process", types.NewSignatureType(
						nil, nil, nil, nil, nil, false,
					),
				)
				iface := types.NewInterfaceType([]*types.Func{method}, nil)
				iface.Complete()
				ifaceNamed := types.NewNamed(
					types.NewTypeName(token.NoPos, ifacePkg, "Processor", nil),
					iface,
					nil,
				)
				ifacePkg.Scope().Insert(ifaceNamed.Obj())
				ifacePkg.MarkComplete()

				testPkg := types.NewPackage("test", "test")
				implStruct := types.NewStruct(nil, nil)
				implNamed := types.NewNamed(
					types.NewTypeName(token.NoPos, testPkg, "MyProcessor", nil),
					implStruct,
					nil,
				)
				// Method on *MyProcessor (pointer receiver)
				ptrType := types.NewPointer(implNamed)
				processMethod := types.NewFunc(
					token.NoPos, testPkg, "Process", types.NewSignatureType(
						types.NewVar(token.NoPos, testPkg, "", ptrType), nil, nil, nil, nil, false,
					),
				)
				implNamed.AddMethod(processMethod)
				testPkg.Scope().Insert(implNamed.Obj())
				testPkg.SetImports([]*types.Package{ifacePkg})
				testPkg.MarkComplete()

				return map[string]*types.Package{
					"example.com/iface": ifacePkg,
				}, implNamed
			},
			src: `package test

import "example.com/iface"

type MyProcessor struct{}

func (*MyProcessor) Process() {}

var _ iface.Processor = &MyProcessor{}
`,
			expectedCount:  1,
			expectedIfaces: []string{"example.com/iface.Processor"},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				additionalPkgs, namedType := tt.setupPkgs()

				var pass *testProvidePass
				if tt.name == "nil TypesInfo" {
					pass = &testProvidePass{
						pkg:       namedType.Obj().Pkg(),
						typesInfo: nil,
					}
				} else {
					mockP, _ := mockPass(t, tt.src, additionalPkgs)
					pass = &testProvidePass{
						pkg:       mockP.Pkg,
						typesInfo: mockP.TypesInfo,
					}
				}

				aPass := pass.toAnalysisPass()
				detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
				ifaces := detector.DetectImplementedInterfaces(aPass, namedType)

				if len(ifaces) != tt.expectedCount {
					t.Errorf(
						"DetectImplementedInterfaces() returned %d interfaces, want %d: %v",
						len(ifaces),
						tt.expectedCount,
						ifaces,
					)
				}

				for _, expected := range tt.expectedIfaces {
					found := false
					for _, got := range ifaces {
						if got == expected {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected interface %q not found in %v", expected, ifaces)
					}
				}
			},
		)
	}
}

// testProvidePass is a helper to construct an analysis.Pass for DetectImplementedInterfaces tests.
type testProvidePass struct {
	pkg       *types.Package
	typesInfo *types.Info
}

func (p *testProvidePass) toAnalysisPass() *analysis.Pass {
	return &analysis.Pass{
		Pkg:       p.pkg,
		TypesInfo: p.typesInfo,
	}
}

func TestProvideCallDetector_DetectImplementedInterfaces_MultipleInterfaces(t *testing.T) {
	// Create two interface packages
	ifacePkg := types.NewPackage("example.com/iface", "iface")

	readerMethod := types.NewFunc(
		token.NoPos, ifacePkg, "Read", types.NewSignatureType(
			nil, nil, nil, nil, nil, false,
		),
	)
	readerIface := types.NewInterfaceType([]*types.Func{readerMethod}, nil)
	readerIface.Complete()
	readerNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, ifacePkg, "Reader", nil),
		readerIface,
		nil,
	)
	ifacePkg.Scope().Insert(readerNamed.Obj())

	writerMethod := types.NewFunc(
		token.NoPos, ifacePkg, "Write", types.NewSignatureType(
			nil, nil, nil, nil, nil, false,
		),
	)
	writerIface := types.NewInterfaceType([]*types.Func{writerMethod}, nil)
	writerIface.Complete()
	writerNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, ifacePkg, "Writer", nil),
		writerIface,
		nil,
	)
	ifacePkg.Scope().Insert(writerNamed.Obj())
	ifacePkg.MarkComplete()

	// Create struct that implements both interfaces
	testPkg := types.NewPackage("test", "test")
	implStruct := types.NewStruct(nil, nil)
	implNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, testPkg, "ReadWriter", nil),
		implStruct,
		nil,
	)
	readMethod := types.NewFunc(
		token.NoPos, testPkg, "Read", types.NewSignatureType(
			types.NewVar(token.NoPos, testPkg, "", implNamed), nil, nil, nil, nil, false,
		),
	)
	writeMethod := types.NewFunc(
		token.NoPos, testPkg, "Write", types.NewSignatureType(
			types.NewVar(token.NoPos, testPkg, "", implNamed), nil, nil, nil, nil, false,
		),
	)
	implNamed.AddMethod(readMethod)
	implNamed.AddMethod(writeMethod)
	testPkg.Scope().Insert(implNamed.Obj())
	testPkg.SetImports([]*types.Package{ifacePkg})
	testPkg.MarkComplete()

	src := `package test

import "example.com/iface"

type ReadWriter struct{}

func (ReadWriter) Read() {}
func (ReadWriter) Write() {}

var _ iface.Reader = ReadWriter{}
var _ iface.Writer = ReadWriter{}
`
	pkgs := map[string]*types.Package{
		"example.com/iface": ifacePkg,
	}
	pass, _ := mockPass(t, src, pkgs)

	detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
	ifaces := detector.DetectImplementedInterfaces(pass, implNamed)

	if len(ifaces) != 2 {
		t.Fatalf("expected 2 interfaces, got %d: %v", len(ifaces), ifaces)
	}

	expectedIfaces := map[string]bool{
		"example.com/iface.Reader": false,
		"example.com/iface.Writer": false,
	}
	for _, iface := range ifaces {
		if _, ok := expectedIfaces[iface]; ok {
			expectedIfaces[iface] = true
		} else {
			t.Errorf("unexpected interface: %q", iface)
		}
	}
	for iface, found := range expectedIfaces {
		if !found {
			t.Errorf("expected interface %q not found", iface)
		}
	}
}

func TestProvideCallDetector_IsProvideCall_ASTPatterns(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	t.Run(
		"SelectorExpr pattern (non-generic)", func(t *testing.T) {
			src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

func NewRepo() *MyRepo { return nil }

type MyRepo struct{}

var _ = annotation.Provide(NewRepo)
`
			pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
			pass, _ := mockPass(t, src, pkgs)

			detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
			candidates := detector.DetectProviders(pass)

			if len(candidates) != 1 {
				t.Errorf("SelectorExpr pattern: expected 1 candidate, got %d", len(candidates))
			}
		},
	)

	t.Run(
		"non-selector call", func(t *testing.T) {
			src := `package test

func someFunc() int { return 0 }

var _ = someFunc()
`
			pass, _ := mockPass(t, src, nil)

			detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
			candidates := detector.DetectProviders(pass)

			if len(candidates) != 0 {
				t.Errorf("non-selector call: expected 0 candidates, got %d", len(candidates))
			}
		},
	)

	t.Run(
		"spec not a ValueSpec", func(t *testing.T) {
			// A GenDecl with TypeSpec (not ValueSpec) should be skipped
			src := `package test

type MyType int
`
			pass, _ := mockPass(t, src, nil)

			detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
			candidates := detector.DetectProviders(pass)

			if len(candidates) != 0 {
				t.Errorf("TypeSpec pattern: expected 0 candidates, got %d", len(candidates))
			}
		},
	)
}

func TestProvideCallDetector_DetectProviders_FunctionReturningNonNamedType(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	// Provider function returning a non-named type (e.g., int)
	src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

func NewValue() int { return 42 }

var _ = annotation.Provide(NewValue)
`
	pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
	pass, _ := mockPass(t, src, pkgs)

	detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
	candidates := detector.DetectProviders(pass)

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	// ReturnTypeName should be empty for non-named types
	if candidates[0].ReturnTypeName != "" {
		t.Errorf("ReturnTypeName for non-named return type = %q, want empty", candidates[0].ReturnTypeName)
	}
}

func TestProvideCallDetector_DetectProviders_NoReturnValue(t *testing.T) {
	annotationPkg := createAnnotationPackageWithProvide()

	src := `package test

import "github.com/miyamo2/braider/pkg/annotation"

func DoNothing() {}

var _ = annotation.Provide(DoNothing)
`
	pkgs := map[string]*types.Package{detect.AnnotationPath: annotationPkg}
	pass, _ := mockPass(t, src, pkgs)

	detector := detect.NewProvideCallDetector(detect.ResolveMarkers())
	candidates := detector.DetectProviders(pass)

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	// No return value: ReturnType should be nil
	if candidates[0].ReturnType != nil {
		t.Errorf("ReturnType for no-return function should be nil, got %v", candidates[0].ReturnType)
	}

	if candidates[0].ReturnTypeName != "" {
		t.Errorf("ReturnTypeName for no-return function should be empty, got %q", candidates[0].ReturnTypeName)
	}
}
