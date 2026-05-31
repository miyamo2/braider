package registry

import (
	"reflect"
	"sync"
	"testing"
)

func TestEntryPointRegistry_RegisterMainPackage_Idempotent(t *testing.T) {
	r := NewEntryPointRegistry()
	r.RegisterMainPackage("example.com/cmd/a")
	r.RegisterMainPackage("example.com/cmd/a")
	r.RegisterMainPackage("example.com/cmd/b")

	got := r.MainPackagePaths()
	want := []string{"example.com/cmd/a", "example.com/cmd/b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MainPackagePaths() = %v, want %v", got, want)
	}
}

func TestEntryPointRegistry_MainPackagePaths_Sorted(t *testing.T) {
	r := NewEntryPointRegistry()
	r.RegisterMainPackage("zzz/cmd")
	r.RegisterMainPackage("aaa/cmd")
	r.RegisterMainPackage("mmm/cmd")

	got := r.MainPackagePaths()
	want := []string{"aaa/cmd", "mmm/cmd", "zzz/cmd"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MainPackagePaths() = %v, want sorted %v", got, want)
	}
}

func TestEntryPointRegistry_MainPackagePaths_EmptyWhenUnused(t *testing.T) {
	r := NewEntryPointRegistry()
	got := r.MainPackagePaths()
	if len(got) != 0 {
		t.Errorf("MainPackagePaths() = %v, want empty", got)
	}
}

func TestEntryPointRegistry_HasExplicitApp(t *testing.T) {
	r := NewEntryPointRegistry()
	if r.HasExplicitApp() {
		t.Error("HasExplicitApp() = true on empty registry, want false")
	}
	r.RegisterExplicitApp("example.com/cmd/a")
	if !r.HasExplicitApp() {
		t.Error("HasExplicitApp() = false after RegisterExplicitApp, want true")
	}
	// idempotent
	r.RegisterExplicitApp("example.com/cmd/a")
	r.RegisterExplicitApp("example.com/cmd/b")
	if !r.HasExplicitApp() {
		t.Error("HasExplicitApp() = false after additional registrations, want true")
	}
}

func TestEntryPointRegistry_ConcurrentRegistration(t *testing.T) {
	r := NewEntryPointRegistry()
	const N = 100
	var wg sync.WaitGroup
	wg.Add(2 * N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			r.RegisterMainPackage("example.com/cmd/concurrent")
		}()
		go func() {
			defer wg.Done()
			r.RegisterExplicitApp("example.com/cmd/concurrent")
		}()
	}
	wg.Wait()
	got := r.MainPackagePaths()
	if !reflect.DeepEqual(got, []string{"example.com/cmd/concurrent"}) {
		t.Errorf("MainPackagePaths() = %v, want single deduplicated entry", got)
	}
	if !r.HasExplicitApp() {
		t.Error("HasExplicitApp() = false after concurrent registrations, want true")
	}
}
