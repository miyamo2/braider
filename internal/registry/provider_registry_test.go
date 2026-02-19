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

			got, ok := r.GetByName("example.com/repo.UserRepository", "")
			if !ok {
				t.Fatal("expected provider to be registered, got ok=false")
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

			got, ok := r.GetByName("example.com/repo.UserRepository", "")
			if !ok {
				t.Fatal("expected provider to be registered, got ok=false")
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

			got, ok := r.GetByName("nonexistent.Type", "")
			if ok {
				t.Errorf("GetByName(nonexistent) returned ok=true, want false")
			}
			if got != nil {
				t.Errorf("GetByName(nonexistent) = %v, want nil", got)
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

			got, ok := r.GetByName("example.com/repo.UserRepository", "")
			if !ok {
				t.Fatal("expected provider, got ok=false")
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
				_, _ = r.GetByName("example.com/pkg.Type"+string(rune('A'+id%26)), "")
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

func TestProviderInfo_GetTypeName(t *testing.T) {
	info := &ProviderInfo{TypeName: "example.com/pkg.Repository"}
	if got := info.GetTypeName(); got != "example.com/pkg.Repository" {
		t.Errorf("GetTypeName() = %q, want %q", got, "example.com/pkg.Repository")
	}
}

func TestProviderInfo_GetDependencies(t *testing.T) {
	deps := []string{"example.com/pkg.DB", "example.com/pkg.Config"}
	info := &ProviderInfo{Dependencies: deps}
	got := info.GetDependencies()
	if len(got) != 2 {
		t.Fatalf("GetDependencies() len = %d, want 2", len(got))
	}
	if got[0] != "example.com/pkg.DB" || got[1] != "example.com/pkg.Config" {
		t.Errorf("GetDependencies() = %v, want %v", got, deps)
	}
}

func TestProviderInfo_GetName(t *testing.T) {
	info := &ProviderInfo{Name: "userRepo"}
	if got := info.GetName(); got != "userRepo" {
		t.Errorf("GetName() = %q, want %q", got, "userRepo")
	}

	// Empty name
	info2 := &ProviderInfo{}
	if got := info2.GetName(); got != "" {
		t.Errorf("GetName() = %q, want empty string", got)
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

	got, ok := r.GetByName("example.com/repo.TestRepository", "")
	if !ok {
		t.Fatal("GlobalProviderRegistry.GetByName() returned ok=false")
	}
	if got.TypeName != info.TypeName {
		t.Errorf("TypeName = %q, want %q", got.TypeName, info.TypeName)
	}
}
