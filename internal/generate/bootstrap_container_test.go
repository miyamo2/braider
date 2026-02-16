package generate

import (
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/graph"
	"golang.org/x/tools/go/analysis"

	goast "go/ast"
)

func makeNamedType(pkgPath, pkgName, typeName string) *types.Named {
	pkg := types.NewPackage(pkgPath, pkgName)
	obj := types.NewTypeName(token.NoPos, pkg, typeName, nil)
	named := types.NewNamed(obj, types.NewStruct(nil, nil), nil)
	return named
}

func TestBootstrapGenerator_GenerateContainerBootstrap_NilInputs(t *testing.T) {
	bg := NewBootstrapGenerator(NewCodeFormatter())
	pass := &analysis.Pass{
		Pkg: types.NewPackage("main", "main"),
	}

	_, err := bg.GenerateContainerBootstrap(pass, nil, nil, &detect.ContainerDefinition{}, nil)
	if err == nil {
		t.Error("expected error for nil graph")
	}

	_, err = bg.GenerateContainerBootstrap(pass, &graph.Graph{}, nil, nil, nil)
	if err == nil {
		t.Error("expected error for nil containerDef")
	}
}

func TestBootstrapGenerator_GenerateContainerBootstrap_NamedContainer(t *testing.T) {
	bg := NewBootstrapGenerator(NewCodeFormatter())
	pass := &analysis.Pass{
		Pkg: types.NewPackage("main", "main"),
	}

	svcType := makeNamedType("example.com/pkg", "pkg", "UserService")

	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.UserService": {
				TypeName:        "example.com/pkg.UserService",
				PackagePath:     "example.com/pkg",
				PackageName:     "pkg",
				LocalName:       "UserService",
				ConstructorName: "NewUserService",
				Dependencies:    []string{},
				IsField:         true,
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.UserService": {},
		},
	}
	sortedTypes := []string{"example.com/pkg.UserService"}

	containerDef := &detect.ContainerDefinition{
		IsNamed:     true,
		TypeName:    "example.com/container.AppContainer",
		PackagePath: "example.com/container",
		PackageName: "container",
		LocalName:   "AppContainer",
		StructType:  types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name: "Svc",
				Type: svcType,
				Pos:  token.NoPos,
			},
		},
	}

	resolvedFields := []detect.ResolvedContainerField{
		{
			FieldName: "Svc",
			NodeKey:   "example.com/pkg.UserService",
			VarName:   "userService",
		},
	}

	bootstrap, err := bg.GenerateContainerBootstrap(pass, g, sortedTypes, containerDef, resolvedFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bootstrap == nil {
		t.Fatal("bootstrap is nil")
	}

	// Should use container type as IIFE return type
	if !strings.Contains(bootstrap.DependencyVar, "container.AppContainer") {
		t.Errorf("missing container return type, got: %s", bootstrap.DependencyVar)
	}

	// Should have constructor call
	if !strings.Contains(bootstrap.DependencyVar, "userService := pkg.NewUserService()") {
		t.Errorf("missing constructor call, got: %s", bootstrap.DependencyVar)
	}

	// Should have return statement with field mapping
	if !strings.Contains(bootstrap.DependencyVar, "Svc: userService") {
		t.Errorf("missing field mapping in return, got: %s", bootstrap.DependencyVar)
	}

	// Hash should be non-empty
	if bootstrap.Hash == "" {
		t.Error("hash is empty")
	}

	// Should have container package in imports
	hasContainerImport := false
	for _, imp := range bootstrap.Imports {
		if imp.Path == "example.com/container" {
			hasContainerImport = true
			break
		}
	}
	if !hasContainerImport {
		t.Error("missing container package in imports")
	}
}

func TestBootstrapGenerator_GenerateContainerBootstrap_SamePackageContainer(t *testing.T) {
	bg := NewBootstrapGenerator(NewCodeFormatter())
	pass := &analysis.Pass{
		Pkg: types.NewPackage("main", "main"),
	}

	svcType := makeNamedType("example.com/pkg", "pkg", "UserService")

	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.UserService": {
				TypeName:        "example.com/pkg.UserService",
				PackagePath:     "example.com/pkg",
				PackageName:     "pkg",
				LocalName:       "UserService",
				ConstructorName: "NewUserService",
				Dependencies:    []string{},
				IsField:         true,
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.UserService": {},
		},
	}
	sortedTypes := []string{"example.com/pkg.UserService"}

	// Container in the same main package
	containerDef := &detect.ContainerDefinition{
		IsNamed:     true,
		TypeName:    "main.AppContainer",
		PackagePath: "main",
		PackageName: "main",
		LocalName:   "AppContainer",
		StructType:  types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name: "Svc",
				Type: svcType,
				Pos:  token.NoPos,
			},
		},
	}

	resolvedFields := []detect.ResolvedContainerField{
		{
			FieldName: "Svc",
			NodeKey:   "example.com/pkg.UserService",
			VarName:   "userService",
		},
	}

	bootstrap, err := bg.GenerateContainerBootstrap(pass, g, sortedTypes, containerDef, resolvedFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use unqualified container name (same package)
	if !strings.Contains(bootstrap.DependencyVar, "AppContainer") {
		t.Errorf("should use unqualified container name, got: %s", bootstrap.DependencyVar)
	}
	// Should NOT have "main.AppContainer"
	if strings.Contains(bootstrap.DependencyVar, "main.AppContainer") {
		t.Errorf("should not qualify same-package container, got: %s", bootstrap.DependencyVar)
	}
}

func TestBootstrapGenerator_GenerateContainerBootstrap_MultipleFields(t *testing.T) {
	bg := NewBootstrapGenerator(NewCodeFormatter())
	pass := &analysis.Pass{
		Pkg: types.NewPackage("main", "main"),
	}

	svcType := makeNamedType("example.com/pkg", "pkg", "UserService")
	logType := makeNamedType("example.com/pkg", "pkg", "Logger")

	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Logger": {
				TypeName:        "example.com/pkg.Logger",
				PackagePath:     "example.com/pkg",
				PackageName:     "pkg",
				LocalName:       "Logger",
				ConstructorName: "NewLogger",
				Dependencies:    []string{},
				IsField:         true,
			},
			"example.com/pkg.UserService": {
				TypeName:        "example.com/pkg.UserService",
				PackagePath:     "example.com/pkg",
				PackageName:     "pkg",
				LocalName:       "UserService",
				ConstructorName: "NewUserService",
				Dependencies:    []string{"example.com/pkg.Logger"},
				IsField:         true,
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.Logger":      {},
			"example.com/pkg.UserService": {"example.com/pkg.Logger"},
		},
	}
	sortedTypes := []string{"example.com/pkg.Logger", "example.com/pkg.UserService"}

	containerDef := &detect.ContainerDefinition{
		IsNamed:     true,
		TypeName:    "example.com/container.AppContainer",
		PackagePath: "example.com/container",
		PackageName: "container",
		LocalName:   "AppContainer",
		StructType:  types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{Name: "Svc", Type: svcType, Pos: token.NoPos},
			{Name: "Log", Type: logType, Pos: token.NoPos},
		},
	}

	resolvedFields := []detect.ResolvedContainerField{
		{FieldName: "Svc", NodeKey: "example.com/pkg.UserService", VarName: "userService"},
		{FieldName: "Log", NodeKey: "example.com/pkg.Logger", VarName: "logger"},
	}

	bootstrap, err := bg.GenerateContainerBootstrap(pass, g, sortedTypes, containerDef, resolvedFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both fields should appear in the return statement
	if !strings.Contains(bootstrap.DependencyVar, "Svc:") {
		t.Errorf("missing Svc field in return, got: %s", bootstrap.DependencyVar)
	}
	if !strings.Contains(bootstrap.DependencyVar, "Log:") {
		t.Errorf("missing Log field in return, got: %s", bootstrap.DependencyVar)
	}

	// Constructor chain should be correct
	if !strings.Contains(bootstrap.DependencyVar, "logger := pkg.NewLogger()") {
		t.Errorf("missing NewLogger call, got: %s", bootstrap.DependencyVar)
	}
	if !strings.Contains(bootstrap.DependencyVar, "userService := pkg.NewUserService(logger)") {
		t.Errorf("missing NewUserService call with logger dep, got: %s", bootstrap.DependencyVar)
	}
}

func TestBootstrapGenerator_GenerateContainerBootstrap_WithVariableNode(t *testing.T) {
	bg := NewBootstrapGenerator(NewCodeFormatter())
	pass := &analysis.Pass{
		Pkg: types.NewPackage("main", "main"),
	}

	svcType := makeNamedType("example.com/pkg", "pkg", "UserService")

	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"os.File": {
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				Dependencies:   []string{},
				IsField:        false,
				ExpressionText: "os.Stdout",
				IsQualified:    true,
			},
			"example.com/pkg.UserService": {
				TypeName:        "example.com/pkg.UserService",
				PackagePath:     "example.com/pkg",
				PackageName:     "pkg",
				LocalName:       "UserService",
				ConstructorName: "NewUserService",
				Dependencies:    []string{"os.File"},
				IsField:         true,
			},
		},
		Edges: map[string][]string{
			"os.File":                     {},
			"example.com/pkg.UserService": {"os.File"},
		},
	}
	sortedTypes := []string{"os.File", "example.com/pkg.UserService"}

	containerDef := &detect.ContainerDefinition{
		IsNamed:     true,
		TypeName:    "main.AppContainer",
		PackagePath: "main",
		PackageName: "main",
		LocalName:   "AppContainer",
		StructType:  types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{Name: "Svc", Type: svcType, Pos: token.NoPos},
		},
	}

	resolvedFields := []detect.ResolvedContainerField{
		{FieldName: "Svc", NodeKey: "example.com/pkg.UserService", VarName: "userService"},
	}

	bootstrap, err := bg.GenerateContainerBootstrap(pass, g, sortedTypes, containerDef, resolvedFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Variable node should use expression assignment (depended upon by UserService)
	if !strings.Contains(bootstrap.DependencyVar, "file := os.Stdout") {
		t.Errorf("missing Variable expression assignment, got: %s", bootstrap.DependencyVar)
	}

	// Service should use constructor with file dep
	if !strings.Contains(bootstrap.DependencyVar, "userService := pkg.NewUserService(file)") {
		t.Errorf("missing constructor call with variable dep, got: %s", bootstrap.DependencyVar)
	}
}

func TestBootstrapGenerator_CheckContainerBootstrapCurrent(t *testing.T) {
	bg := NewBootstrapGenerator(NewCodeFormatter())

	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"example.com/pkg.Service": {
				TypeName:        "example.com/pkg.Service",
				PackagePath:     "example.com/pkg",
				LocalName:       "Service",
				ConstructorName: "NewService",
				IsField:         true,
			},
		},
		Edges: map[string][]string{
			"example.com/pkg.Service": {},
		},
	}

	svcType := makeNamedType("example.com/pkg", "pkg", "Service")

	containerDef := &detect.ContainerDefinition{
		IsNamed:     true,
		TypeName:    "main.AppContainer",
		PackagePath: "main",
		PackageName: "main",
		LocalName:   "AppContainer",
		StructType:  types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{Name: "Svc", Type: svcType, Pos: token.NoPos},
		},
	}

	hash := ComputeContainerHash(g, containerDef)

	source := `package main

// braider:hash:` + hash + `
var dependency = func() AppContainer {
	service := pkg.NewService()
	return AppContainer{
		Svc: service,
	}
}()
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	pass := &analysis.Pass{
		Files: []*goast.File{file},
	}

	existing := bg.DetectExistingBootstrap(pass)
	if existing == nil {
		t.Fatal("Failed to detect existing bootstrap")
	}

	isCurrent := bg.CheckContainerBootstrapCurrent(pass, existing, g, containerDef)
	if !isCurrent {
		t.Error("CheckContainerBootstrapCurrent() should return true for matching hash")
	}

	// Different container def should produce different hash
	containerDef2 := &detect.ContainerDefinition{
		IsNamed:     true,
		TypeName:    "main.AppContainer",
		PackagePath: "main",
		PackageName: "main",
		LocalName:   "AppContainer",
		StructType:  types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{Name: "Svc", Type: svcType, Pos: token.NoPos},
			{Name: "Extra", Type: svcType, Pos: token.NoPos},
		},
	}

	isCurrent2 := bg.CheckContainerBootstrapCurrent(pass, existing, g, containerDef2)
	if isCurrent2 {
		t.Error("CheckContainerBootstrapCurrent() should return false for different container def")
	}

	// Nil inputs
	if bg.CheckContainerBootstrapCurrent(pass, nil, g, containerDef) {
		t.Error("should return false for nil existing")
	}
	if bg.CheckContainerBootstrapCurrent(pass, existing, nil, containerDef) {
		t.Error("should return false for nil graph")
	}
}
