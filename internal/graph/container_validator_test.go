package graph

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
)

// helper to create a simple named type for testing
func makeTestNamedType(pkgPath, pkgName, typeName string) *types.Named {
	pkg := types.NewPackage(pkgPath, pkgName)
	obj := types.NewTypeName(token.NoPos, pkg, typeName, nil)
	named := types.NewNamed(obj, types.NewStruct(nil, nil), nil)
	return named
}

// helper to create an interface type for testing
func makeTestInterfaceType(pkgPath, pkgName, typeName string) *types.Named {
	pkg := types.NewPackage(pkgPath, pkgName)
	obj := types.NewTypeName(token.NoPos, pkg, typeName, nil)
	iface := types.NewInterfaceType(nil, nil)
	iface.Complete()
	named := types.NewNamed(obj, iface, nil)
	return named
}

func TestContainerValidator_Validate_BraiderTagDash(t *testing.T) {
	reg := NewInterfaceRegistry()
	v := NewContainerValidatorImpl(reg)

	namedType := makeTestNamedType("example.com/pkg", "pkg", "Foo")

	def := &detect.ContainerDefinition{
		StructType: types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name:          "Svc",
				Type:          namedType,
				HasBraiderTag: true,
				Tag:           "-",
				Pos:           token.NoPos,
			},
		},
	}
	g := &Graph{Nodes: map[string]*Node{}}
	errs := v.Validate(def, g)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Message != `braider:"-" tag is not permitted in container struct fields` {
		t.Errorf("unexpected message: %s", errs[0].Message)
	}
	if errs[0].FieldName != "Svc" {
		t.Errorf("unexpected field name: %s", errs[0].FieldName)
	}
}

func TestContainerValidator_Validate_BraiderTagEmpty(t *testing.T) {
	reg := NewInterfaceRegistry()
	v := NewContainerValidatorImpl(reg)

	namedType := makeTestNamedType("example.com/pkg", "pkg", "Foo")

	def := &detect.ContainerDefinition{
		StructType: types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name:          "Svc",
				Type:          namedType,
				HasBraiderTag: true,
				Tag:           "",
				Pos:           token.NoPos,
			},
		},
	}
	g := &Graph{Nodes: map[string]*Node{}}
	errs := v.Validate(def, g)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	expected := `braider:"" empty tag is not permitted, use braider:"name" to specify a dependency name`
	if errs[0].Message != expected {
		t.Errorf("unexpected message: %s", errs[0].Message)
	}
}

func TestContainerValidator_Validate_ConcreteTypeMatch(t *testing.T) {
	reg := NewInterfaceRegistry()
	v := NewContainerValidatorImpl(reg)

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
				TypeName: "example.com/pkg.UserService",
			},
		},
	}
	errs := v.Validate(def, g)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func TestContainerValidator_Validate_ConcreteTypeNotFound(t *testing.T) {
	reg := NewInterfaceRegistry()
	v := NewContainerValidatorImpl(reg)

	namedType := makeTestNamedType("example.com/pkg", "pkg", "MissingService")

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
	g := &Graph{Nodes: map[string]*Node{}}
	errs := v.Validate(def, g)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Message != "no matching dependency found in the graph" {
		t.Errorf("unexpected message: %s", errs[0].Message)
	}
}

func TestContainerValidator_Validate_AmbiguousConcreteType(t *testing.T) {
	reg := NewInterfaceRegistry()
	v := NewContainerValidatorImpl(reg)

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
				TypeName: "example.com/pkg.UserService",
			},
			"example.com/pkg.UserService#primary": {
				TypeName: "example.com/pkg.UserService",
				Name:     "primary",
			},
		},
	}
	errs := v.Validate(def, g)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Message != `ambiguous: multiple named registrations match this type, use braider:"name" tag to disambiguate` {
		t.Errorf("unexpected message: %s", errs[0].Message)
	}
}

func TestContainerValidator_Validate_NamedFieldMatch(t *testing.T) {
	reg := NewInterfaceRegistry()
	v := NewContainerValidatorImpl(reg)

	namedType := makeTestNamedType("example.com/pkg", "pkg", "UserService")

	def := &detect.ContainerDefinition{
		StructType: types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name:          "Svc",
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
				TypeName: "example.com/pkg.UserService",
				Name:     "primary",
			},
		},
	}
	errs := v.Validate(def, g)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func TestContainerValidator_Validate_NamedFieldNotFound(t *testing.T) {
	reg := NewInterfaceRegistry()
	v := NewContainerValidatorImpl(reg)

	namedType := makeTestNamedType("example.com/pkg", "pkg", "UserService")

	def := &detect.ContainerDefinition{
		StructType: types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name:          "Svc",
				Type:          namedType,
				HasBraiderTag: true,
				Tag:           "secondary",
				Pos:           token.NoPos,
			},
		},
	}
	g := &Graph{
		Nodes: map[string]*Node{
			"example.com/pkg.UserService#primary": {
				TypeName: "example.com/pkg.UserService",
				Name:     "primary",
			},
		},
	}
	errs := v.Validate(def, g)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Message != "no matching named dependency found in the graph" {
		t.Errorf("unexpected message: %s", errs[0].Message)
	}
}

func TestContainerValidator_Validate_InterfaceTypeResolved(t *testing.T) {
	reg := NewInterfaceRegistry()
	// Register implementation
	reg.interfaces["example.com/pkg.Repository"] = []string{"example.com/pkg.RepositoryImpl"}

	v := NewContainerValidatorImpl(reg)

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
				TypeName: "example.com/pkg.RepositoryImpl",
			},
		},
	}
	errs := v.Validate(def, g)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func TestContainerValidator_Validate_InterfaceTypeUnresolved(t *testing.T) {
	reg := NewInterfaceRegistry()
	v := NewContainerValidatorImpl(reg)

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
	g := &Graph{Nodes: map[string]*Node{}}
	errs := v.Validate(def, g)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Message != "no matching dependency found in the graph" {
		t.Errorf("unexpected message: %s", errs[0].Message)
	}
}

func TestContainerValidator_Validate_PointerFieldType(t *testing.T) {
	reg := NewInterfaceRegistry()
	v := NewContainerValidatorImpl(reg)

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
				TypeName: "example.com/pkg.UserService",
			},
		},
	}
	errs := v.Validate(def, g)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func TestContainerValidator_Validate_NoBraiderTagResolvesByType(t *testing.T) {
	reg := NewInterfaceRegistry()
	v := NewContainerValidatorImpl(reg)

	namedType := makeTestNamedType("example.com/pkg", "pkg", "Logger")

	def := &detect.ContainerDefinition{
		StructType: types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name:          "Log",
				Type:          namedType,
				HasBraiderTag: false,
				Tag:           "",
				Pos:           token.NoPos,
			},
		},
	}
	g := &Graph{
		Nodes: map[string]*Node{
			"example.com/pkg.Logger": {
				TypeName: "example.com/pkg.Logger",
			},
		},
	}
	errs := v.Validate(def, g)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func TestContainerValidator_Validate_MultipleFields(t *testing.T) {
	reg := NewInterfaceRegistry()
	v := NewContainerValidatorImpl(reg)

	svcType := makeTestNamedType("example.com/pkg", "pkg", "UserService")
	missingType := makeTestNamedType("example.com/pkg", "pkg", "Missing")

	def := &detect.ContainerDefinition{
		StructType: types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name: "Svc",
				Type: svcType,
				Pos:  token.NoPos,
			},
			{
				Name: "Other",
				Type: missingType,
				Pos:  token.NoPos,
			},
		},
	}
	g := &Graph{
		Nodes: map[string]*Node{
			"example.com/pkg.UserService": {
				TypeName: "example.com/pkg.UserService",
			},
		},
	}
	errs := v.Validate(def, g)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error (for Missing), got %d: %v", len(errs), errs)
	}
	if errs[0].FieldName != "Other" {
		t.Errorf("unexpected field name: %s", errs[0].FieldName)
	}
}

func TestContainerValidator_Validate_InterfaceFieldNilRegistry(t *testing.T) {
	v := NewContainerValidatorImpl(nil) // nil registry

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
	g := &Graph{Nodes: map[string]*Node{}}
	errs := v.Validate(def, g)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Message != "no matching dependency found in the graph (interface registry unavailable)" {
		t.Errorf("unexpected message: %s", errs[0].Message)
	}
}

func TestContainerValidator_Validate_NamedFieldInterfaceResolution(t *testing.T) {
	reg := NewInterfaceRegistry()
	reg.interfaces["example.com/pkg.Repository"] = []string{"example.com/pkg.RepositoryImpl"}

	v := NewContainerValidatorImpl(reg)

	ifaceType := makeTestInterfaceType("example.com/pkg", "pkg", "Repository")

	def := &detect.ContainerDefinition{
		StructType: types.NewStruct(nil, nil),
		Fields: []detect.ContainerField{
			{
				Name:          "Repo",
				Type:          ifaceType,
				HasBraiderTag: true,
				Tag:           "primary",
				Pos:           token.NoPos,
			},
		},
	}
	g := &Graph{
		Nodes: map[string]*Node{
			"example.com/pkg.RepositoryImpl#primary": {
				TypeName: "example.com/pkg.RepositoryImpl",
				Name:     "primary",
			},
		},
	}
	errs := v.Validate(def, g)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func TestFullyQualifiedTypeName(t *testing.T) {
	tests := []struct {
		name     string
		typ      types.Type
		expected string
	}{
		{
			name:     "named type",
			typ:      makeTestNamedType("example.com/pkg", "pkg", "Foo"),
			expected: "example.com/pkg.Foo",
		},
		{
			name:     "pointer to named type",
			typ:      types.NewPointer(makeTestNamedType("example.com/pkg", "pkg", "Bar")),
			expected: "example.com/pkg.Bar",
		},
		{
			name:     "double pointer",
			typ:      types.NewPointer(types.NewPointer(makeTestNamedType("example.com/pkg", "pkg", "Baz"))),
			expected: "example.com/pkg.Baz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fullyQualifiedTypeName(tt.typ)
			if result != tt.expected {
				t.Errorf("fullyQualifiedTypeName() = %s, want %s", result, tt.expected)
			}
		})
	}
}
