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
			TypeName: "example.com/repo.UserRepository",
			Name:     "primaryRepo",
		}
		info2 := &ProviderInfo{
			TypeName: "example.com/repo.UserRepository",
			Name:     "secondaryRepo",
		}

		if err := r.Register(info1); err != nil {
			t.Fatal(err)
		}
		if err := r.Register(info2); err != nil {
			t.Fatal(err)
		}

		got1, ok1 := r.GetByName("example.com/repo.UserRepository", "primaryRepo")
		got2, ok2 := r.GetByName("example.com/repo.UserRepository", "secondaryRepo")

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

	t.Run("registers same type with different names without collision", func(t *testing.T) {
		r := NewProviderRegistry()

		info1 := &ProviderInfo{TypeName: "example.com/repo.Repo", Name: "alpha", PackagePath: "pkg1"}
		info2 := &ProviderInfo{TypeName: "example.com/repo.Repo", Name: "beta", PackagePath: "pkg2"}

		if err := r.Register(info1); err != nil {
			t.Fatal(err)
		}
		if err := r.Register(info2); err != nil {
			t.Fatal(err)
		}

		got1, ok1 := r.GetByName("example.com/repo.Repo", "alpha")
		got2, ok2 := r.GetByName("example.com/repo.Repo", "beta")

		if !ok1 || !ok2 {
			t.Fatal("both should exist")
		}
		if got1.Name != "alpha" || got2.Name != "beta" {
			t.Fatal("wrong names")
		}
	})

	t.Run("named and unnamed providers of same type coexist", func(t *testing.T) {
		r := NewProviderRegistry()

		unnamed := &ProviderInfo{TypeName: "example.com/repo.Repo", Name: ""}
		named := &ProviderInfo{TypeName: "example.com/repo.Repo", Name: "special"}

		if err := r.Register(unnamed); err != nil {
			t.Fatal(err)
		}
		if err := r.Register(named); err != nil {
			t.Fatal(err)
		}

		got := r.GetAll()
		if len(got) != 2 {
			t.Errorf("expected 2, got %d", len(got))
		}
	})

	t.Run("GetAll includes all named variants", func(t *testing.T) {
		r := NewProviderRegistry()

		r.Register(&ProviderInfo{TypeName: "example.com/repo.Repo", Name: ""})
		r.Register(&ProviderInfo{TypeName: "example.com/repo.Repo", Name: "a"})
		r.Register(&ProviderInfo{TypeName: "example.com/repo.Repo", Name: "b"})

		got := r.GetAll()
		if len(got) != 3 {
			t.Errorf("expected 3, got %d", len(got))
		}
	})
}
