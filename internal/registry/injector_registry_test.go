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

			got, ok := r.GetByName("example.com/service.UserService", "")
			if !ok {
				t.Fatal("expected injector to be registered, got ok=false")
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

			got, ok := r.GetByName("example.com/service.UserService", "")
			if !ok {
				t.Fatal("expected injector to be registered, got ok=false")
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
		"returns injector when found", func(t *testing.T) {
			r := NewInjectorRegistry()

			info := &InjectorInfo{
				TypeName:        "example.com/service.UserService",
				PackagePath:     "example.com/service",
				LocalName:       "UserService",
				ConstructorName: "NewUserService",
			}
			r.Register(info)

			got, ok := r.GetByName("example.com/service.UserService", "")
			if !ok {
				t.Fatal("expected injector, got ok=false")
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
				_, _ = r.GetByName("example.com/pkg.Type"+string(rune('A'+id%26)), "")
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

func TestInjectorInfo_GetTypeName(t *testing.T) {
	info := &InjectorInfo{TypeName: "example.com/pkg.Service"}
	if got := info.GetTypeName(); got != "example.com/pkg.Service" {
		t.Errorf("GetTypeName() = %q, want %q", got, "example.com/pkg.Service")
	}
}

func TestInjectorInfo_GetDependencies(t *testing.T) {
	deps := []string{"example.com/pkg.Repo", "example.com/pkg.Logger"}
	info := &InjectorInfo{Dependencies: deps}
	got := info.GetDependencies()
	if len(got) != 2 {
		t.Fatalf("GetDependencies() len = %d, want 2", len(got))
	}
	if got[0] != "example.com/pkg.Repo" || got[1] != "example.com/pkg.Logger" {
		t.Errorf("GetDependencies() = %v, want %v", got, deps)
	}
}

func TestInjectorInfo_GetName(t *testing.T) {
	info := &InjectorInfo{Name: "primaryDB"}
	if got := info.GetName(); got != "primaryDB" {
		t.Errorf("GetName() = %q, want %q", got, "primaryDB")
	}

	// Empty name
	info2 := &InjectorInfo{}
	if got := info2.GetName(); got != "" {
		t.Errorf("GetName() = %q, want empty string", got)
	}
}

func TestGlobalInjectorRegistry(t *testing.T) {
	r := NewInjectorRegistry()

	info := &InjectorInfo{
		TypeName:        "example.com/service.TestService",
		PackagePath:     "example.com/service",
		LocalName:       "TestService",
		ConstructorName: "NewTestService",
	}

	r.Register(info)

	got, ok := r.GetByName("example.com/service.TestService", "")
	if !ok {
		t.Fatal("GlobalInjectorRegistry.GetByName() returned ok=false")
	}
	if got.TypeName != info.TypeName {
		t.Errorf("TypeName = %q, want %q", got.TypeName, info.TypeName)
	}
}
