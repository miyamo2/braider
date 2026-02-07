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

		r.Register(info)

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
		r.Register(info)

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
		r.Register(info)

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

		// Note: In real usage, we would validate duplicate (TypeName, Name) pairs
		// For this test, we manually create distinct entries by using different TypeNames
		info1 := &InjectorInfo{
			TypeName: "example.com/service.UserService#primary",
			Name:     "primaryUser",
		}
		info2 := &InjectorInfo{
			TypeName: "example.com/service.UserService#secondary",
			Name:     "secondaryUser",
		}

		r.Register(info1)
		r.Register(info2)

		got1, ok1 := r.GetByName("example.com/service.UserService#primary", "primaryUser")
		got2, ok2 := r.GetByName("example.com/service.UserService#secondary", "secondaryUser")

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
}
