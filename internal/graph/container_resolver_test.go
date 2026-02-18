package graph

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
)

func TestContainerResolver_ResolveFields_ConcreteType(t *testing.T) {
	reg := NewInterfaceRegistry()
	r := NewContainerResolverImpl(reg)

	namedType := makeTestNamedType("example.com/pkg", "pkg", "UserService")

	def := &detect.ContainerDefinition{
		StructType: types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name: "Svc",
				Type: namedType,
				Pos:  token.NoPos,
			},
		},
	}
	g := &Graph{
		Nodes: map[string]*Node{
			"example.com/pkg.UserService": {
				TypeName:  "example.com/pkg.UserService",
				LocalName: "UserService",
			},
		},
	}
	resolved, err := r.ResolveFields(def, g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved field, got %d", len(resolved))
	}
	if resolved[0].FieldName != "Svc" {
		t.Errorf("FieldName = %s, want Svc", resolved[0].FieldName)
	}
	if resolved[0].NodeKey != "example.com/pkg.UserService" {
		t.Errorf("NodeKey = %s, want example.com/pkg.UserService", resolved[0].NodeKey)
	}
	if resolved[0].VarName != "userService" {
		t.Errorf("VarName = %s, want userService", resolved[0].VarName)
	}
}

func TestContainerResolver_ResolveFields_PointerType(t *testing.T) {
	reg := NewInterfaceRegistry()
	r := NewContainerResolverImpl(reg)

	namedType := makeTestNamedType("example.com/pkg", "pkg", "UserService")
	ptrType := types.NewPointer(namedType)

	def := &detect.ContainerDefinition{
		StructType: types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name: "Svc",
				Type: ptrType,
				Pos:  token.NoPos,
			},
		},
	}
	g := &Graph{
		Nodes: map[string]*Node{
			"example.com/pkg.UserService": {
				TypeName:  "example.com/pkg.UserService",
				LocalName: "UserService",
			},
		},
	}
	resolved, err := r.ResolveFields(def, g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved field, got %d", len(resolved))
	}
	if resolved[0].NodeKey != "example.com/pkg.UserService" {
		t.Errorf("NodeKey = %s, want example.com/pkg.UserService", resolved[0].NodeKey)
	}
}

func TestContainerResolver_ResolveFields_NamedField(t *testing.T) {
	reg := NewInterfaceRegistry()
	r := NewContainerResolverImpl(reg)

	namedType := makeTestNamedType("example.com/pkg", "pkg", "UserService")

	def := &detect.ContainerDefinition{
		StructType: types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name:          "Primary",
				Type:          namedType,
				HasBraiderTag: true,
				Tag:           "primary",
				Pos:           token.NoPos,
			},
		},
	}
	g := &Graph{
		Nodes: map[string]*Node{
			"example.com/pkg.UserService#primary": {
				TypeName:  "example.com/pkg.UserService",
				LocalName: "UserService",
				Name:      "primary",
			},
		},
	}
	resolved, err := r.ResolveFields(def, g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved field, got %d", len(resolved))
	}
	if resolved[0].VarName != "primary" {
		t.Errorf("VarName = %s, want primary", resolved[0].VarName)
	}
	if resolved[0].NodeKey != "example.com/pkg.UserService#primary" {
		t.Errorf("NodeKey = %s, want example.com/pkg.UserService#primary", resolved[0].NodeKey)
	}
}

func TestContainerResolver_ResolveFields_InterfaceType(t *testing.T) {
	reg := NewInterfaceRegistry()
	reg.interfaces["example.com/pkg.Repository"] = []string{"example.com/pkg.RepositoryImpl"}

	r := NewContainerResolverImpl(reg)

	ifaceType := makeTestInterfaceType("example.com/pkg", "pkg", "Repository")

	def := &detect.ContainerDefinition{
		StructType: types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name: "Repo",
				Type: ifaceType,
				Pos:  token.NoPos,
			},
		},
	}
	g := &Graph{
		Nodes: map[string]*Node{
			"example.com/pkg.RepositoryImpl": {
				TypeName:  "example.com/pkg.RepositoryImpl",
				LocalName: "RepositoryImpl",
			},
		},
	}
	resolved, err := r.ResolveFields(def, g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved field, got %d", len(resolved))
	}
	if resolved[0].VarName != "repositoryImpl" {
		t.Errorf("VarName = %s, want repositoryImpl", resolved[0].VarName)
	}
}

func TestContainerResolver_ResolveFields_NotFound(t *testing.T) {
	reg := NewInterfaceRegistry()
	r := NewContainerResolverImpl(reg)

	namedType := makeTestNamedType("example.com/pkg", "pkg", "Missing")

	def := &detect.ContainerDefinition{
		StructType: types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name: "M",
				Type: namedType,
				Pos:  token.NoPos,
			},
		},
	}
	g := &Graph{Nodes: map[string]*Node{}}

	_, err := r.ResolveFields(def, g)
	if err == nil {
		t.Error("expected error for unresolvable field")
	}
}

func TestContainerResolver_ResolveFields_MultipleFields(t *testing.T) {
	reg := NewInterfaceRegistry()
	r := NewContainerResolverImpl(reg)

	svcType := makeTestNamedType("example.com/pkg", "pkg", "UserService")
	logType := makeTestNamedType("example.com/pkg", "pkg", "Logger")

	def := &detect.ContainerDefinition{
		StructType: types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name: "Svc",
				Type: svcType,
				Pos:  token.NoPos,
			},
			{
				Name: "Log",
				Type: logType,
				Pos:  token.NoPos,
			},
		},
	}
	g := &Graph{
		Nodes: map[string]*Node{
			"example.com/pkg.UserService": {
				TypeName:  "example.com/pkg.UserService",
				LocalName: "UserService",
			},
			"example.com/pkg.Logger": {
				TypeName:  "example.com/pkg.Logger",
				LocalName: "Logger",
			},
		},
	}
	resolved, err := r.ResolveFields(def, g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved fields, got %d", len(resolved))
	}
	// Order should be preserved from containerDef.Fields
	if resolved[0].FieldName != "Svc" {
		t.Errorf("first field = %s, want Svc", resolved[0].FieldName)
	}
	if resolved[1].FieldName != "Log" {
		t.Errorf("second field = %s, want Log", resolved[1].FieldName)
	}
}

func TestDeriveFieldNameFromType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com/pkg.UserService", "userService"},
		{"example.com/pkg.HTTPClient", "httpClient"},
		{"example.com/pkg.DB", "db"},
		{"Service", "service"},
		{"", "field"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := deriveFieldNameFromType(tt.input)
			if result != tt.expected {
				t.Errorf("deriveFieldNameFromType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
