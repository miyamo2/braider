package registry

import (
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
)

func TestProviderInfo_OptionMetadataFields(t *testing.T) {
	t.Run("registers provider with option metadata", func(t *testing.T) {
		r := NewProviderRegistry()

		interfaceType := types.NewInterfaceType(nil, nil)
		info := &ProviderInfo{
			TypeName:        "example.com/repo.UserRepository",
			PackagePath:     "example.com/repo",
			LocalName:       "UserRepository",
			ConstructorName: "NewUserRepository",
			RegisteredType:  interfaceType,
			Name:            "primaryRepo",
			OptionMetadata: detect.OptionMetadata{
				TypedInterface: interfaceType,
				Name:           "primaryRepo",
			},
		}

		r.Register(info)

		got := r.Get("example.com/repo.UserRepository")
		if got == nil {
			t.Fatal("expected provider to be registered, got nil")
		}
		if got.RegisteredType == nil {
			t.Error("RegisteredType should be set")
		}
		if got.Name != "primaryRepo" {
			t.Errorf("Name = %q, want %q", got.Name, "primaryRepo")
		}
		if got.OptionMetadata.TypedInterface == nil {
			t.Error("OptionMetadata.TypedInterface should be set")
		}
		if got.OptionMetadata.Name != "primaryRepo" {
			t.Errorf("OptionMetadata.Name = %q, want %q", got.OptionMetadata.Name, "primaryRepo")
		}
	})

	t.Run("registers provider with default option", func(t *testing.T) {
		r := NewProviderRegistry()

		info := &ProviderInfo{
			TypeName:        "example.com/repo.OrderRepository",
			PackagePath:     "example.com/repo",
			LocalName:       "OrderRepository",
			ConstructorName: "NewOrderRepository",
			OptionMetadata: detect.OptionMetadata{
				IsDefault: true,
			},
		}

		r.Register(info)

		got := r.Get("example.com/repo.OrderRepository")
		if got == nil {
			t.Fatal("expected provider to be registered, got nil")
		}
		if !got.OptionMetadata.IsDefault {
			t.Error("OptionMetadata.IsDefault should be true")
		}
	})

	t.Run("unnamed provider has empty Name field", func(t *testing.T) {
		r := NewProviderRegistry()

		info := &ProviderInfo{
			TypeName:        "example.com/repo.BasicRepository",
			PackagePath:     "example.com/repo",
			LocalName:       "BasicRepository",
			ConstructorName: "NewBasicRepository",
			Name:            "",
			OptionMetadata: detect.OptionMetadata{
				IsDefault: true,
			},
		}

		r.Register(info)

		got := r.Get("example.com/repo.BasicRepository")
		if got == nil {
			t.Fatal("expected provider to be registered, got nil")
		}
		if got.Name != "" {
			t.Errorf("Name should be empty, got %q", got.Name)
		}
	})
}
