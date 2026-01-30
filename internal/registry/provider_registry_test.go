package registry

import (
	"sort"
	"sync"
	"testing"
)

func TestProviderRegistry_Register(t *testing.T) {
	t.Run(
		"registers a single provider", func(t *testing.T) {
			r := NewProviderRegistry()

			info := &ProviderInfo{
				TypeName:        "example.com/repo.UserRepository",
				PackagePath:     "example.com/repo",
				LocalName:       "UserRepository",
				ConstructorName: "NewUserRepository",
				Dependencies:    []string{},
			}

			r.Register(info)

			got := r.Get("example.com/repo.UserRepository")
			if got == nil {
				t.Fatal("expected provider to be registered, got nil")
			}
			if got.TypeName != info.TypeName {
				t.Errorf("TypeName = %q, want %q", got.TypeName, info.TypeName)
			}
			if got.LocalName != info.LocalName {
				t.Errorf("LocalName = %q, want %q", got.LocalName, info.LocalName)
			}
			if got.ConstructorName != info.ConstructorName {
				t.Errorf("ConstructorName = %q, want %q", got.ConstructorName, info.ConstructorName)
			}
		},
	)

	t.Run(
		"overwrites existing provider with same type name", func(t *testing.T) {
			r := NewProviderRegistry()

			info1 := &ProviderInfo{
				TypeName:        "example.com/repo.UserRepository",
				PackagePath:     "example.com/repo",
				LocalName:       "UserRepository",
				ConstructorName: "NewUserRepository",
				Dependencies:    []string{},
			}
			info2 := &ProviderInfo{
				TypeName:        "example.com/repo.UserRepository",
				PackagePath:     "example.com/repo",
				LocalName:       "UserRepository",
				ConstructorName: "NewUserRepositoryV2",
				Dependencies:    []string{"example.com/db.DB"},
			}

			r.Register(info1)
			r.Register(info2)

			got := r.Get("example.com/repo.UserRepository")
			if got == nil {
				t.Fatal("expected provider to be registered, got nil")
			}
			if got.ConstructorName != "NewUserRepositoryV2" {
				t.Errorf("ConstructorName = %q, want %q", got.ConstructorName, "NewUserRepositoryV2")
			}
		},
	)
}

func TestProviderRegistry_GetAll(t *testing.T) {
	t.Run(
		"returns empty slice when no providers registered", func(t *testing.T) {
			r := NewProviderRegistry()

			got := r.GetAll()
			if got == nil {
				t.Fatal("expected non-nil slice, got nil")
			}
			if len(got) != 0 {
				t.Errorf("len(GetAll()) = %d, want 0", len(got))
			}
		},
	)

	t.Run(
		"returns all registered providers", func(t *testing.T) {
			r := NewProviderRegistry()

			info1 := &ProviderInfo{
				TypeName:        "example.com/repo.UserRepository",
				PackagePath:     "example.com/repo",
				LocalName:       "UserRepository",
				ConstructorName: "NewUserRepository",
			}
			info2 := &ProviderInfo{
				TypeName:        "example.com/repo.OrderRepository",
				PackagePath:     "example.com/repo",
				LocalName:       "OrderRepository",
				ConstructorName: "NewOrderRepository",
			}

			r.Register(info1)
			r.Register(info2)

			got := r.GetAll()
			if len(got) != 2 {
				t.Fatalf("len(GetAll()) = %d, want 2", len(got))
			}
		},
	)

	t.Run(
		"returns providers in deterministic alphabetical order", func(t *testing.T) {
			r := NewProviderRegistry()

			// Register in reverse alphabetical order
			infos := []*ProviderInfo{
				{TypeName: "z.com/pkg.ZType", LocalName: "ZType"},
				{TypeName: "a.com/pkg.AType", LocalName: "AType"},
				{TypeName: "m.com/pkg.MType", LocalName: "MType"},
			}
			for _, info := range infos {
				r.Register(info)
			}

			got := r.GetAll()
			if len(got) != 3 {
				t.Fatalf("len(GetAll()) = %d, want 3", len(got))
			}

			// Verify alphabetical order by TypeName
			expected := []string{"a.com/pkg.AType", "m.com/pkg.MType", "z.com/pkg.ZType"}
			for i, p := range got {
				if p.TypeName != expected[i] {
					t.Errorf("GetAll()[%d].TypeName = %q, want %q", i, p.TypeName, expected[i])
				}
			}
		},
	)

	t.Run(
		"returns copy of slice to prevent external mutation", func(t *testing.T) {
			r := NewProviderRegistry()

			info := &ProviderInfo{
				TypeName:  "example.com/repo.UserRepository",
				LocalName: "UserRepository",
			}
			r.Register(info)

			got1 := r.GetAll()
			got2 := r.GetAll()

			// Modify the first slice
			if len(got1) > 0 {
				got1[0] = nil
			}

			// Second slice should be unaffected
			if got2[0] == nil {
				t.Error("GetAll() should return a copy, but modification affected other calls")
			}
		},
	)
}

func TestProviderRegistry_Get(t *testing.T) {
	t.Run(
		"returns nil when provider not found", func(t *testing.T) {
			r := NewProviderRegistry()

			got := r.Get("nonexistent.Type")
			if got != nil {
				t.Errorf("Get(nonexistent) = %v, want nil", got)
			}
		},
	)

	t.Run(
		"returns provider when found", func(t *testing.T) {
			r := NewProviderRegistry()

			info := &ProviderInfo{
				TypeName:        "example.com/repo.UserRepository",
				PackagePath:     "example.com/repo",
				LocalName:       "UserRepository",
				ConstructorName: "NewUserRepository",
			}
			r.Register(info)

			got := r.Get("example.com/repo.UserRepository")
			if got == nil {
				t.Fatal("expected provider, got nil")
			}
			if got.TypeName != info.TypeName {
				t.Errorf("TypeName = %q, want %q", got.TypeName, info.TypeName)
			}
		},
	)
}

func TestProviderRegistry_ThreadSafety(t *testing.T) {
	r := NewProviderRegistry()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 types of operations

	// Concurrent writes
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for range numOperations {
				info := &ProviderInfo{
					TypeName:  "example.com/pkg.Type" + string(rune('A'+id%26)),
					LocalName: "Type" + string(rune('A'+id%26)),
				}
				r.Register(info)
			}
		}(i)
	}

	// Concurrent reads (GetAll)
	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range numOperations {
				_ = r.GetAll()
			}
		}()
	}

	// Concurrent reads (Get)
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for range numOperations {
				_ = r.Get("example.com/pkg.Type" + string(rune('A'+id%26)))
			}
		}(i)
	}

	wg.Wait()

	// Verify the registry is in a consistent state
	providers := r.GetAll()
	if providers == nil {
		t.Fatal("GetAll() returned nil after concurrent operations")
	}

	// Verify alphabetical ordering is maintained
	typeNames := make([]string, len(providers))
	for i, p := range providers {
		typeNames[i] = p.TypeName
	}
	if !sort.StringsAreSorted(typeNames) {
		t.Error("GetAll() did not return providers in sorted order after concurrent operations")
	}
}

func TestGlobalProviderRegistry(t *testing.T) {
	r := NewProviderRegistry()

	info := &ProviderInfo{
		TypeName:        "example.com/repo.TestRepository",
		PackagePath:     "example.com/repo",
		LocalName:       "TestRepository",
		ConstructorName: "NewTestRepository",
	}

	r.Register(info)

	got := r.Get("example.com/repo.TestRepository")
	if got == nil {
		t.Fatal("GlobalProviderRegistry.Get() returned nil")
	}
	if got.TypeName != info.TypeName {
		t.Errorf("TypeName = %q, want %q", got.TypeName, info.TypeName)
	}
}
