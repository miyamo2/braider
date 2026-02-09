package registry

import (
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
)

func TestInjectorRegistry_GetByName(t *testing.T) {
	t.Run("returns nil and false for non-existent named dependency", func(t *testing.T) {
		r := NewInjectorRegistry()

		got, ok := r.GetByName("example.com/service.UserService", "primary")
		if got != nil {
			t.Errorf("GetByName(nonexistent) returned package, want nil")
		}
		if ok {
			t.Errorf("GetByName(nonexistent) returned ok=true, want false")
		}
	})

	t.Run("retrieves named dependency", func(t *testing.T) {
		r := NewInjectorRegistry()

		interfaceType := types.NewInterfaceType(nil, nil)
		info := &InjectorInfo{
			TypeName:        "example.com/service.UserService",
			PackagePath:     "example.com/service",
			LocalName:       "UserService",
			ConstructorName: "NewUserService",
			RegisteredType:  interfaceType,
			Name:            "primaryUser",
			OptionMetadata: detect.OptionMetadata{
				TypedInterface: interfaceType,
				Name:           "primaryUser",
			},
		}

		if err := r.Register(info); err != nil {
			t.Fatalf("Register() returned error: %v", err)
		}

		got, ok := r.GetByName("example.com/service.UserService", "primaryUser")
		if !ok {
			t.Fatal("GetByName() returned ok=false, want true")
		}
		if got == nil {
			t.Fatal("GetByName() returned nil, want info")
		}
		if got.Name != "primaryUser" {
			t.Errorf("GetByName().Name = %q, want %q", got.Name, "primaryUser")
		}
		if got.TypeName != "example.com/service.UserService" {
			t.Errorf("GetByName().TypeName = %q, want %q", got.TypeName, "example.com/service.UserService")
		}
	})

	t.Run("returns false for wrong name", func(t *testing.T) {
		r := NewInjectorRegistry()

		info := &InjectorInfo{
			TypeName: "example.com/service.UserService",
			Name:     "primaryUser",
		}
		if err := r.Register(info); err != nil {
			t.Fatalf("Register() returned error: %v", err)
		}

		got, ok := r.GetByName("example.com/service.UserService", "secondaryUser")
		if ok {
			t.Errorf("GetByName(wrong name) returned ok=true, want false")
		}
		if got != nil {
			t.Errorf("GetByName(wrong name) returned info, want nil")
		}
	})

	t.Run("returns false for unnamed dependency", func(t *testing.T) {
		r := NewInjectorRegistry()

		info := &InjectorInfo{
			TypeName: "example.com/service.UserService",
			Name:     "",
		}
		if err := r.Register(info); err != nil {
			t.Fatalf("Register() returned error: %v", err)
		}

		got, ok := r.GetByName("example.com/service.UserService", "anyName")
		if ok {
			t.Errorf("GetByName(unnamed dep) returned ok=true, want false")
		}
		if got != nil {
			t.Errorf("GetByName(unnamed dep) returned info, want nil")
		}
	})

	t.Run("distinguishes between multiple instances with different names", func(t *testing.T) {
		r := NewInjectorRegistry()

		info1 := &InjectorInfo{
			TypeName: "example.com/service.UserService",
			Name:     "primaryUser",
		}
		info2 := &InjectorInfo{
			TypeName: "example.com/service.UserService",
			Name:     "secondaryUser",
		}

		if err := r.Register(info1); err != nil {
			t.Fatalf("Register(info1) returned error: %v", err)
		}
		if err := r.Register(info2); err != nil {
			t.Fatalf("Register(info2) returned error: %v", err)
		}

		got1, ok1 := r.GetByName("example.com/service.UserService", "primaryUser")
		got2, ok2 := r.GetByName("example.com/service.UserService", "secondaryUser")

		if !ok1 || !ok2 {
			t.Fatal("GetByName() returned ok=false for one or both named dependencies")
		}
		if got1.Name != "primaryUser" {
			t.Errorf("got1.Name = %q, want %q", got1.Name, "primaryUser")
		}
		if got2.Name != "secondaryUser" {
			t.Errorf("got2.Name = %q, want %q", got2.Name, "secondaryUser")
		}
	})

	t.Run("registers same type with different names without collision", func(t *testing.T) {
		r := NewInjectorRegistry()

		info1 := &InjectorInfo{TypeName: "example.com/service.Svc", Name: "alpha", PackagePath: "pkg1"}
		info2 := &InjectorInfo{TypeName: "example.com/service.Svc", Name: "beta", PackagePath: "pkg2"}

		if err := r.Register(info1); err != nil {
			t.Fatalf("Register(info1) returned error: %v", err)
		}
		if err := r.Register(info2); err != nil {
			t.Fatalf("Register(info2) returned error: %v", err)
		}

		got1, ok1 := r.GetByName("example.com/service.Svc", "alpha")
		got2, ok2 := r.GetByName("example.com/service.Svc", "beta")

		if !ok1 || !ok2 {
			t.Fatal("both should exist")
		}
		if got1.Name != "alpha" || got2.Name != "beta" {
			t.Fatal("wrong names")
		}
	})

	t.Run("named and unnamed injectors of same type coexist", func(t *testing.T) {
		r := NewInjectorRegistry()

		unnamed := &InjectorInfo{TypeName: "example.com/service.Svc", Name: ""}
		named := &InjectorInfo{TypeName: "example.com/service.Svc", Name: "special"}

		if err := r.Register(unnamed); err != nil {
			t.Fatalf("Register(unnamed) returned error: %v", err)
		}
		if err := r.Register(named); err != nil {
			t.Fatalf("Register(named) returned error: %v", err)
		}

		got := r.GetAll()
		if len(got) != 2 {
			t.Errorf("expected 2, got %d", len(got))
		}
	})

	t.Run("GetAll includes all named variants", func(t *testing.T) {
		r := NewInjectorRegistry()

		r.Register(&InjectorInfo{TypeName: "example.com/service.Svc", Name: ""})
		r.Register(&InjectorInfo{TypeName: "example.com/service.Svc", Name: "a"})
		r.Register(&InjectorInfo{TypeName: "example.com/service.Svc", Name: "b"})

		got := r.GetAll()
		if len(got) != 3 {
			t.Errorf("expected 3, got %d", len(got))
		}
	})
}
