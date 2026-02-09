package registry

import (
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
)

func TestProviderRegistry_GetByName(t *testing.T) {
	t.Run("returns nil and false for non-existent named dependency", func(t *testing.T) {
		r := NewProviderRegistry()

		got, ok := r.GetByName("example.com/repo.UserRepository", "primary")
		if got != nil {
			t.Errorf("GetByName(nonexistent) returned package, want nil")
		}
		if ok {
			t.Errorf("GetByName(nonexistent) returned ok=true, want false")
		}
	})

	t.Run("retrieves named provider", func(t *testing.T) {
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

		if err := r.Register(info); err != nil {
			t.Fatal(err)
		}

		got, ok := r.GetByName("example.com/repo.UserRepository", "primaryRepo")
		if !ok {
			t.Fatal("GetByName() returned ok=false, want true")
		}
		if got == nil {
			t.Fatal("GetByName() returned nil, want info")
		}
		if got.Name != "primaryRepo" {
			t.Errorf("GetByName().Name = %q, want %q", got.Name, "primaryRepo")
		}
		if got.TypeName != "example.com/repo.UserRepository" {
			t.Errorf("GetByName().TypeName = %q, want %q", got.TypeName, "example.com/repo.UserRepository")
		}
	})

	t.Run("returns false for wrong name", func(t *testing.T) {
		r := NewProviderRegistry()

		info := &ProviderInfo{
			TypeName: "example.com/repo.UserRepository",
			Name:     "primaryRepo",
		}
		if err := r.Register(info); err != nil {
			t.Fatal(err)
		}

		got, ok := r.GetByName("example.com/repo.UserRepository", "secondaryRepo")
		if ok {
			t.Errorf("GetByName(wrong name) returned ok=true, want false")
		}
		if got != nil {
			t.Errorf("GetByName(wrong name) returned info, want nil")
		}
	})

	t.Run("returns false for unnamed provider", func(t *testing.T) {
		r := NewProviderRegistry()

		info := &ProviderInfo{
			TypeName: "example.com/repo.UserRepository",
			Name:     "",
		}
		if err := r.Register(info); err != nil {
			t.Fatal(err)
		}

		got, ok := r.GetByName("example.com/repo.UserRepository", "anyName")
		if ok {
			t.Errorf("GetByName(unnamed provider) returned ok=true, want false")
		}
		if got != nil {
			t.Errorf("GetByName(unnamed provider) returned info, want nil")
		}
	})

	t.Run("distinguishes between multiple instances with different names", func(t *testing.T) {
		r := NewProviderRegistry()

		info1 := &ProviderInfo{
			TypeName: "example.com/repo.UserRepository#primary",
			Name:     "primaryRepo",
		}
		info2 := &ProviderInfo{
			TypeName: "example.com/repo.UserRepository#secondary",
			Name:     "secondaryRepo",
		}

		if err := r.Register(info1); err != nil {
			t.Fatal(err)
		}
		if err := r.Register(info2); err != nil {
			t.Fatal(err)
		}

		got1, ok1 := r.GetByName("example.com/repo.UserRepository#primary", "primaryRepo")
		got2, ok2 := r.GetByName("example.com/repo.UserRepository#secondary", "secondaryRepo")

		if !ok1 || !ok2 {
			t.Fatal("GetByName() returned ok=false for one or both named providers")
		}
		if got1.Name != "primaryRepo" {
			t.Errorf("got1.Name = %q, want %q", got1.Name, "primaryRepo")
		}
		if got2.Name != "secondaryRepo" {
			t.Errorf("got2.Name = %q, want %q", got2.Name, "secondaryRepo")
		}
	})
}
