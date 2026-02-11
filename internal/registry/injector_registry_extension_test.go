package registry

import (
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
)

func TestInjectorInfo_OptionMetadataFields(t *testing.T) {
	t.Run("registers injector with option metadata", func(t *testing.T) {
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
		if !ok || got == nil {
			t.Fatal("expected injector to be registered, got nil")
		}
		if got.RegisteredType == nil {
			t.Error("RegisteredType should be set")
		}
		if got.Name != "primaryUser" {
			t.Errorf("Name = %q, want %q", got.Name, "primaryUser")
		}
		if got.OptionMetadata.TypedInterface == nil {
			t.Error("OptionMetadata.TypedInterface should be set")
		}
		if got.OptionMetadata.Name != "primaryUser" {
			t.Errorf("OptionMetadata.Name = %q, want %q", got.OptionMetadata.Name, "primaryUser")
		}
	})

	t.Run("registers injector with default option", func(t *testing.T) {
		r := NewInjectorRegistry()

		info := &InjectorInfo{
			TypeName:        "example.com/service.OrderService",
			PackagePath:     "example.com/service",
			LocalName:       "OrderService",
			ConstructorName: "NewOrderService",
			OptionMetadata: detect.OptionMetadata{
				IsDefault: true,
			},
		}

		if err := r.Register(info); err != nil {
			t.Fatalf("Register() returned error: %v", err)
		}

		got := r.Get("example.com/service.OrderService")
		if got == nil {
			t.Fatal("expected injector to be registered, got nil")
		}
		if !got.OptionMetadata.IsDefault {
			t.Error("OptionMetadata.IsDefault should be true")
		}
	})

	t.Run("registers injector with WithoutConstructor option", func(t *testing.T) {
		r := NewInjectorRegistry()

		info := &InjectorInfo{
			TypeName:        "example.com/service.CustomService",
			PackagePath:     "example.com/service",
			LocalName:       "CustomService",
			ConstructorName: "NewCustomService",
			OptionMetadata: detect.OptionMetadata{
				WithoutConstructor: true,
			},
		}

		if err := r.Register(info); err != nil {
			t.Fatalf("Register() returned error: %v", err)
		}

		got := r.Get("example.com/service.CustomService")
		if got == nil {
			t.Fatal("expected injector to be registered, got nil")
		}
		if !got.OptionMetadata.WithoutConstructor {
			t.Error("OptionMetadata.WithoutConstructor should be true")
		}
	})

	t.Run("unnamed injector has empty Name field", func(t *testing.T) {
		r := NewInjectorRegistry()

		info := &InjectorInfo{
			TypeName:        "example.com/service.BasicService",
			PackagePath:     "example.com/service",
			LocalName:       "BasicService",
			ConstructorName: "NewBasicService",
			Name:            "",
			OptionMetadata: detect.OptionMetadata{
				IsDefault: true,
			},
		}

		if err := r.Register(info); err != nil {
			t.Fatal(err)
		}

		got := r.Get("example.com/service.BasicService")
		if got == nil {
			t.Fatal("expected injector to be registered, got nil")
		}
		if got.Name != "" {
			t.Errorf("Name should be empty, got %q", got.Name)
		}
	})
}
