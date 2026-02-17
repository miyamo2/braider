package detect

import (
	"go/token"
	"go/types"
	"reflect"
	"testing"
)

func TestLookupMarkerInterface_NilObject(t *testing.T) {
	pkg := types.NewPackage("test/pkg", "pkg")
	pkg.MarkComplete()

	result := lookupMarkerInterface(pkg, "NonExistent")
	if result != nil {
		t.Errorf("expected nil for non-existent name, got %v", result)
	}
}

func TestLookupMarkerInterface_NotTypeName(t *testing.T) {
	pkg := types.NewPackage("test/pkg", "pkg")
	v := types.NewVar(token.NoPos, pkg, "NotAType", types.Typ[types.Int])
	pkg.Scope().Insert(v)
	pkg.MarkComplete()

	result := lookupMarkerInterface(pkg, "NotAType")
	if result != nil {
		t.Errorf("expected nil for non-TypeName object, got %v", result)
	}
}

func TestLookupMarkerInterface_NotInterface(t *testing.T) {
	pkg := types.NewPackage("test/pkg", "pkg")
	tn := types.NewTypeName(token.NoPos, pkg, "MyStruct", nil)
	types.NewNamed(tn, types.NewStruct(nil, nil), nil)
	pkg.Scope().Insert(tn)
	pkg.MarkComplete()

	result := lookupMarkerInterface(pkg, "MyStruct")
	if result != nil {
		t.Errorf("expected nil for non-interface type, got %v", result)
	}
}

func TestLookupMarkerInterface_ValidInterface(t *testing.T) {
	pkg := types.NewPackage("test/pkg", "pkg")
	method := types.NewFunc(token.NoPos, pkg, "Method", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	iface := types.NewInterfaceType([]*types.Func{method}, nil)
	iface.Complete()
	tn := types.NewTypeName(token.NoPos, pkg, "MyInterface", nil)
	types.NewNamed(tn, iface, nil)
	pkg.Scope().Insert(tn)
	pkg.MarkComplete()

	result := lookupMarkerInterface(pkg, "MyInterface")
	if result == nil {
		t.Error("expected non-nil for valid interface type")
	}
}

func TestResolveMarkers_AllFieldsNonNil(t *testing.T) {
	markers, err := ResolveMarkers()
	if err != nil {
		t.Fatalf("ResolveMarkers() failed: %v", err)
	}
	if markers == nil {
		t.Fatal("ResolveMarkers() returned nil markers")
	}

	v := reflect.ValueOf(markers).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := v.Type().Field(i).Name
		if field.IsNil() {
			t.Errorf("MarkerInterfaces.%s is nil, expected non-nil", fieldName)
		}
	}
}
