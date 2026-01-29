package registry

import (
	"sort"
	"sync"
	"testing"
)

func TestInjectorRegistry_Register(t *testing.T) {
	t.Run(
		"registers a single injector", func(t *testing.T) {
			r := NewInjectorRegistry()

			info := &InjectorInfo{
				TypeName:        "example.com/service.UserService",
				PackagePath:     "example.com/service",
				LocalName:       "UserService",
				ConstructorName: "NewUserService",
				Dependencies:    []string{"example.com/repo.UserRepository"},
			}

			r.Register(info)

			got := r.Get("example.com/service.UserService")
			if got == nil {
				t.Fatal("expected injector to be registered, got nil")
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
		"overwrites existing injector with same type name", func(t *testing.T) {
			r := NewInjectorRegistry()

			info1 := &InjectorInfo{
				TypeName:        "example.com/service.UserService",
				PackagePath:     "example.com/service",
				LocalName:       "UserService",
				ConstructorName: "NewUserService",
				Dependencies:    []string{},
			}
			info2 := &InjectorInfo{
				TypeName:        "example.com/service.UserService",
				PackagePath:     "example.com/service",
				LocalName:       "UserService",
				ConstructorName: "NewUserServiceV2",
				Dependencies:    []string{"example.com/repo.UserRepository"},
			}

			r.Register(info1)
			r.Register(info2)

			got := r.Get("example.com/service.UserService")
			if got == nil {
				t.Fatal("expected injector to be registered, got nil")
			}
			if got.ConstructorName != "NewUserServiceV2" {
				t.Errorf("ConstructorName = %q, want %q", got.ConstructorName, "NewUserServiceV2")
			}
		},
	)
}

func TestInjectorRegistry_GetAll(t *testing.T) {
	t.Run(
		"returns empty slice when no injectors registered", func(t *testing.T) {
			r := NewInjectorRegistry()

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
		"returns all registered injectors", func(t *testing.T) {
			r := NewInjectorRegistry()

			info1 := &InjectorInfo{
				TypeName:        "example.com/service.UserService",
				PackagePath:     "example.com/service",
				LocalName:       "UserService",
				ConstructorName: "NewUserService",
			}
			info2 := &InjectorInfo{
				TypeName:        "example.com/service.OrderService",
				PackagePath:     "example.com/service",
				LocalName:       "OrderService",
				ConstructorName: "NewOrderService",
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
		"returns injectors in deterministic alphabetical order", func(t *testing.T) {
			r := NewInjectorRegistry()

			// Register in reverse alphabetical order
			infos := []*InjectorInfo{
				{TypeName: "z.com/pkg.ZService", LocalName: "ZService"},
				{TypeName: "a.com/pkg.AService", LocalName: "AService"},
				{TypeName: "m.com/pkg.MService", LocalName: "MService"},
			}
			for _, info := range infos {
				r.Register(info)
			}

			got := r.GetAll()
			if len(got) != 3 {
				t.Fatalf("len(GetAll()) = %d, want 3", len(got))
			}

			// Verify alphabetical order by TypeName
			expected := []string{"a.com/pkg.AService", "m.com/pkg.MService", "z.com/pkg.ZService"}
			for i, inj := range got {
				if inj.TypeName != expected[i] {
					t.Errorf("GetAll()[%d].TypeName = %q, want %q", i, inj.TypeName, expected[i])
				}
			}
		},
	)

	t.Run(
		"returns copy of slice to prevent external mutation", func(t *testing.T) {
			r := NewInjectorRegistry()

			info := &InjectorInfo{
				TypeName:  "example.com/service.UserService",
				LocalName: "UserService",
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

func TestInjectorRegistry_Get(t *testing.T) {
	t.Run(
		"returns nil when injector not found", func(t *testing.T) {
			r := NewInjectorRegistry()

			got := r.Get("nonexistent.Type")
			if got != nil {
				t.Errorf("Get(nonexistent) = %v, want nil", got)
			}
		},
	)

	t.Run(
		"returns injector when found", func(t *testing.T) {
			r := NewInjectorRegistry()

			info := &InjectorInfo{
				TypeName:        "example.com/service.UserService",
				PackagePath:     "example.com/service",
				LocalName:       "UserService",
				ConstructorName: "NewUserService",
			}
			r.Register(info)

			got := r.Get("example.com/service.UserService")
			if got == nil {
				t.Fatal("expected injector, got nil")
			}
			if got.TypeName != info.TypeName {
				t.Errorf("TypeName = %q, want %q", got.TypeName, info.TypeName)
			}
		},
	)
}

func TestInjectorRegistry_ThreadSafety(t *testing.T) {
	r := NewInjectorRegistry()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 types of operations

	// Concurrent writes
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for range numOperations {
				info := &InjectorInfo{
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
	injectors := r.GetAll()
	if injectors == nil {
		t.Fatal("GetAll() returned nil after concurrent operations")
	}

	// Verify alphabetical ordering is maintained
	typeNames := make([]string, len(injectors))
	for i, inj := range injectors {
		typeNames[i] = inj.TypeName
	}
	if !sort.StringsAreSorted(typeNames) {
		t.Error("GetAll() did not return injectors in sorted order after concurrent operations")
	}
}

func TestGlobalInjectorRegistry(t *testing.T) {
	info := &InjectorInfo{
		TypeName:        "example.com/service.TestService",
		PackagePath:     "example.com/service",
		LocalName:       "TestService",
		ConstructorName: "NewTestService",
	}

	GlobalInjectorRegistry.Register(info)

	got := GlobalInjectorRegistry.Get("example.com/service.TestService")
	if got == nil {
		t.Fatal("GlobalInjectorRegistry.Get() returned nil")
	}
	if got.TypeName != info.TypeName {
		t.Errorf("TypeName = %q, want %q", got.TypeName, info.TypeName)
	}
}
