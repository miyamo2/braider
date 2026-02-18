package registry

import (
	"sort"
	"sync"
	"testing"
)

func TestVariableRegistry_Register(t *testing.T) {
	t.Run(
		"registers a single variable", func(t *testing.T) {
			r := NewVariableRegistry()

			info := &VariableInfo{
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				ExpressionText: "os.Stdout",
				ExpressionPkgs: []string{"os"},
				IsQualified:    true,
				Dependencies:   []string{},
			}

			if err := r.Register(info); err != nil {
				t.Fatalf("Register() returned error: %v", err)
			}

			got, ok := r.GetByName("os.File", "")
			if !ok {
				t.Fatal("expected variable to be registered, got ok=false")
			}
			if got.TypeName != info.TypeName {
				t.Errorf("TypeName = %q, want %q", got.TypeName, info.TypeName)
			}
			if got.LocalName != info.LocalName {
				t.Errorf("LocalName = %q, want %q", got.LocalName, info.LocalName)
			}
			if got.ExpressionText != info.ExpressionText {
				t.Errorf("ExpressionText = %q, want %q", got.ExpressionText, info.ExpressionText)
			}
			if got.PackageName != info.PackageName {
				t.Errorf("PackageName = %q, want %q", got.PackageName, info.PackageName)
			}
			if got.IsQualified != info.IsQualified {
				t.Errorf("IsQualified = %v, want %v", got.IsQualified, info.IsQualified)
			}
		},
	)

	t.Run(
		"overwrites existing variable with same type name", func(t *testing.T) {
			r := NewVariableRegistry()

			info1 := &VariableInfo{
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				ExpressionText: "os.Stdout",
				Dependencies:   []string{},
			}
			info2 := &VariableInfo{
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				ExpressionText: "os.Stderr",
				Dependencies:   []string{},
			}

			r.Register(info1)
			r.Register(info2)

			got, ok := r.GetByName("os.File", "")
			if !ok {
				t.Fatal("expected variable to be registered, got ok=false")
			}
			if got.ExpressionText != "os.Stderr" {
				t.Errorf("ExpressionText = %q, want %q", got.ExpressionText, "os.Stderr")
			}
		},
	)
}

func TestVariableRegistry_Register_DuplicateNamed(t *testing.T) {
	t.Run("returns error for duplicate named variable", func(t *testing.T) {
		r := NewVariableRegistry()

		info1 := &VariableInfo{
			TypeName:       "os.File",
			PackagePath:    "pkg1",
			PackageName:    "os",
			LocalName:      "File",
			ExpressionText: "os.Stdout",
			Name:           "output",
			Dependencies:   []string{},
		}
		info2 := &VariableInfo{
			TypeName:       "os.File",
			PackagePath:    "pkg2",
			PackageName:    "os",
			LocalName:      "File",
			ExpressionText: "os.Stderr",
			Name:           "output",
			Dependencies:   []string{},
		}

		if err := r.Register(info1); err != nil {
			t.Fatalf("Register(info1) returned error: %v", err)
		}

		err := r.Register(info2)
		if err == nil {
			t.Fatal("Register(info2) should return error for duplicate named variable, got nil")
		}
	})

	t.Run("allows same type with different names", func(t *testing.T) {
		r := NewVariableRegistry()

		info1 := &VariableInfo{
			TypeName:       "os.File",
			PackagePath:    "pkg1",
			PackageName:    "os",
			LocalName:      "File",
			ExpressionText: "os.Stdout",
			Name:           "stdout",
			Dependencies:   []string{},
		}
		info2 := &VariableInfo{
			TypeName:       "os.File",
			PackagePath:    "pkg2",
			PackageName:    "os",
			LocalName:      "File",
			ExpressionText: "os.Stderr",
			Name:           "stderr",
			Dependencies:   []string{},
		}

		if err := r.Register(info1); err != nil {
			t.Fatalf("Register(info1) returned error: %v", err)
		}
		if err := r.Register(info2); err != nil {
			t.Fatalf("Register(info2) returned error: %v", err)
		}

		got1, ok1 := r.GetByName("os.File", "stdout")
		got2, ok2 := r.GetByName("os.File", "stderr")

		if !ok1 || !ok2 {
			t.Fatal("GetByName() returned ok=false for one or both named variables")
		}
		if got1.ExpressionText != "os.Stdout" {
			t.Errorf("got1.ExpressionText = %q, want %q", got1.ExpressionText, "os.Stdout")
		}
		if got2.ExpressionText != "os.Stderr" {
			t.Errorf("got2.ExpressionText = %q, want %q", got2.ExpressionText, "os.Stderr")
		}
	})

	t.Run("named and unnamed variables of same type coexist", func(t *testing.T) {
		r := NewVariableRegistry()

		unnamed := &VariableInfo{TypeName: "os.File", Name: "", ExpressionText: "os.Stdout", Dependencies: []string{}}
		named := &VariableInfo{TypeName: "os.File", Name: "special", ExpressionText: "os.Stderr", Dependencies: []string{}}

		if err := r.Register(unnamed); err != nil {
			t.Fatalf("Register(unnamed) returned error: %v", err)
		}
		if err := r.Register(named); err != nil {
			t.Fatalf("Register(named) returned error: %v", err)
		}

		got := r.GetAll()
		if len(got) != 2 {
			t.Errorf("expected 2, got %d", len(got))
		}
	})
}

func TestVariableRegistry_GetAll(t *testing.T) {
	t.Run(
		"returns empty slice when no variables registered", func(t *testing.T) {
			r := NewVariableRegistry()

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
		"returns all registered variables", func(t *testing.T) {
			r := NewVariableRegistry()

			info1 := &VariableInfo{
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				ExpressionText: "os.Stdout",
				Dependencies:   []string{},
			}
			info2 := &VariableInfo{
				TypeName:       "config.Config",
				PackagePath:    "example.com/config",
				PackageName:    "config",
				LocalName:      "Config",
				ExpressionText: "config.DefaultConfig",
				Dependencies:   []string{},
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
		"returns variables in deterministic alphabetical order", func(t *testing.T) {
			r := NewVariableRegistry()

			// Register in reverse alphabetical order
			infos := []*VariableInfo{
				{TypeName: "z.com/pkg.ZType", LocalName: "ZType", ExpressionText: "z", Dependencies: []string{}},
				{TypeName: "a.com/pkg.AType", LocalName: "AType", ExpressionText: "a", Dependencies: []string{}},
				{TypeName: "m.com/pkg.MType", LocalName: "MType", ExpressionText: "m", Dependencies: []string{}},
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
			for i, v := range got {
				if v.TypeName != expected[i] {
					t.Errorf("GetAll()[%d].TypeName = %q, want %q", i, v.TypeName, expected[i])
				}
			}
		},
	)

	t.Run(
		"sorts by name within same type name", func(t *testing.T) {
			r := NewVariableRegistry()

			r.Register(&VariableInfo{TypeName: "os.File", Name: "beta", ExpressionText: "os.Stderr", Dependencies: []string{}})
			r.Register(&VariableInfo{TypeName: "os.File", Name: "alpha", ExpressionText: "os.Stdout", Dependencies: []string{}})
			r.Register(&VariableInfo{TypeName: "os.File", Name: "", ExpressionText: "os.Stdin", Dependencies: []string{}})

			got := r.GetAll()
			if len(got) != 3 {
				t.Fatalf("len(GetAll()) = %d, want 3", len(got))
			}

			// Empty name sorts first, then "alpha", then "beta"
			expectedNames := []string{"", "alpha", "beta"}
			for i, v := range got {
				if v.Name != expectedNames[i] {
					t.Errorf("GetAll()[%d].Name = %q, want %q", i, v.Name, expectedNames[i])
				}
			}
		},
	)

	t.Run(
		"returns copy of slice to prevent external mutation", func(t *testing.T) {
			r := NewVariableRegistry()

			info := &VariableInfo{
				TypeName:       "os.File",
				LocalName:      "File",
				ExpressionText: "os.Stdout",
				Dependencies:   []string{},
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

	t.Run("GetAll includes all named variants", func(t *testing.T) {
		r := NewVariableRegistry()

		r.Register(&VariableInfo{TypeName: "os.File", Name: "", ExpressionText: "os.Stdout", Dependencies: []string{}})
		r.Register(&VariableInfo{TypeName: "os.File", Name: "a", ExpressionText: "os.Stderr", Dependencies: []string{}})
		r.Register(&VariableInfo{TypeName: "os.File", Name: "b", ExpressionText: "os.Stdin", Dependencies: []string{}})

		got := r.GetAll()
		if len(got) != 3 {
			t.Errorf("expected 3, got %d", len(got))
		}
	})
}

func TestVariableRegistry_Get(t *testing.T) {
	t.Run(
		"returns nil when variable not found", func(t *testing.T) {
			r := NewVariableRegistry()

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
		"returns variable when found", func(t *testing.T) {
			r := NewVariableRegistry()

			info := &VariableInfo{
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				ExpressionText: "os.Stdout",
				Dependencies:   []string{},
			}
			r.Register(info)

			got, ok := r.GetByName("os.File", "")
			if !ok {
				t.Fatal("expected variable, got ok=false")
			}
			if got.TypeName != info.TypeName {
				t.Errorf("TypeName = %q, want %q", got.TypeName, info.TypeName)
			}
		},
	)
}

func TestVariableRegistry_GetByName(t *testing.T) {
	t.Run("returns nil and false for non-existent named variable", func(t *testing.T) {
		r := NewVariableRegistry()

		got, ok := r.GetByName("os.File", "output")
		if got != nil {
			t.Errorf("GetByName(nonexistent) returned variable, want nil")
		}
		if ok {
			t.Errorf("GetByName(nonexistent) returned ok=true, want false")
		}
	})

	t.Run("retrieves named variable", func(t *testing.T) {
		r := NewVariableRegistry()

		info := &VariableInfo{
			TypeName:       "os.File",
			PackagePath:    "os",
			PackageName:    "os",
			LocalName:      "File",
			ExpressionText: "os.Stdout",
			Name:           "output",
			Dependencies:   []string{},
		}

		if err := r.Register(info); err != nil {
			t.Fatal(err)
		}

		got, ok := r.GetByName("os.File", "output")
		if !ok {
			t.Fatal("GetByName() returned ok=false, want true")
		}
		if got == nil {
			t.Fatal("GetByName() returned nil, want info")
		}
		if got.Name != "output" {
			t.Errorf("GetByName().Name = %q, want %q", got.Name, "output")
		}
		if got.TypeName != "os.File" {
			t.Errorf("GetByName().TypeName = %q, want %q", got.TypeName, "os.File")
		}
	})

	t.Run("returns false for wrong name", func(t *testing.T) {
		r := NewVariableRegistry()

		info := &VariableInfo{
			TypeName:       "os.File",
			Name:           "output",
			ExpressionText: "os.Stdout",
			Dependencies:   []string{},
		}
		if err := r.Register(info); err != nil {
			t.Fatal(err)
		}

		got, ok := r.GetByName("os.File", "wrongName")
		if ok {
			t.Errorf("GetByName(wrong name) returned ok=true, want false")
		}
		if got != nil {
			t.Errorf("GetByName(wrong name) returned info, want nil")
		}
	})

	t.Run("returns false for unnamed variable", func(t *testing.T) {
		r := NewVariableRegistry()

		info := &VariableInfo{
			TypeName:       "os.File",
			Name:           "",
			ExpressionText: "os.Stdout",
			Dependencies:   []string{},
		}
		if err := r.Register(info); err != nil {
			t.Fatal(err)
		}

		got, ok := r.GetByName("os.File", "anyName")
		if ok {
			t.Errorf("GetByName(unnamed variable) returned ok=true, want false")
		}
		if got != nil {
			t.Errorf("GetByName(unnamed variable) returned info, want nil")
		}
	})

	t.Run("distinguishes between multiple instances with different names", func(t *testing.T) {
		r := NewVariableRegistry()

		info1 := &VariableInfo{
			TypeName:       "os.File",
			Name:           "stdout",
			ExpressionText: "os.Stdout",
			Dependencies:   []string{},
		}
		info2 := &VariableInfo{
			TypeName:       "os.File",
			Name:           "stderr",
			ExpressionText: "os.Stderr",
			Dependencies:   []string{},
		}

		if err := r.Register(info1); err != nil {
			t.Fatalf("Register(info1) returned error: %v", err)
		}
		if err := r.Register(info2); err != nil {
			t.Fatalf("Register(info2) returned error: %v", err)
		}

		got1, ok1 := r.GetByName("os.File", "stdout")
		got2, ok2 := r.GetByName("os.File", "stderr")

		if !ok1 || !ok2 {
			t.Fatal("GetByName() returned ok=false for one or both named variables")
		}
		if got1.Name != "stdout" {
			t.Errorf("got1.Name = %q, want %q", got1.Name, "stdout")
		}
		if got2.Name != "stderr" {
			t.Errorf("got2.Name = %q, want %q", got2.Name, "stderr")
		}
	})
}

func TestVariableRegistry_MetadataStorage(t *testing.T) {
	t.Run("stores all metadata fields correctly", func(t *testing.T) {
		r := NewVariableRegistry()

		info := &VariableInfo{
			TypeName:       "os.File",
			PackagePath:    "os",
			PackageName:    "os",
			LocalName:      "File",
			ExpressionText: "os.Stdout",
			ExpressionPkgs: []string{"os"},
			IsQualified:    true,
			Dependencies:   []string{},
			Implements:     []string{"io.Writer"},
			Name:           "output",
		}

		if err := r.Register(info); err != nil {
			t.Fatal(err)
		}

		got, ok := r.GetByName("os.File", "output")
		if !ok || got == nil {
			t.Fatal("expected variable to be registered, got nil")
		}
		if got.PackagePath != "os" {
			t.Errorf("PackagePath = %q, want %q", got.PackagePath, "os")
		}
		if got.PackageName != "os" {
			t.Errorf("PackageName = %q, want %q", got.PackageName, "os")
		}
		if got.LocalName != "File" {
			t.Errorf("LocalName = %q, want %q", got.LocalName, "File")
		}
		if got.ExpressionText != "os.Stdout" {
			t.Errorf("ExpressionText = %q, want %q", got.ExpressionText, "os.Stdout")
		}
		if len(got.ExpressionPkgs) != 1 || got.ExpressionPkgs[0] != "os" {
			t.Errorf("ExpressionPkgs = %v, want [\"os\"]", got.ExpressionPkgs)
		}
		if !got.IsQualified {
			t.Error("IsQualified should be true")
		}
		if len(got.Implements) != 1 || got.Implements[0] != "io.Writer" {
			t.Errorf("Implements = %v, want [\"io.Writer\"]", got.Implements)
		}
		if got.Name != "output" {
			t.Errorf("Name = %q, want %q", got.Name, "output")
		}
	})

	t.Run("dependencies are always empty", func(t *testing.T) {
		r := NewVariableRegistry()

		info := &VariableInfo{
			TypeName:       "os.File",
			ExpressionText: "os.Stdout",
			Dependencies:   []string{},
		}
		r.Register(info)

		got, ok := r.GetByName("os.File", "")
		if !ok {
			t.Fatal("expected variable, got ok=false")
		}
		if len(got.Dependencies) != 0 {
			t.Errorf("Dependencies should be empty, got %v", got.Dependencies)
		}
	})
}

func TestVariableInfo_DependencyInfoInterface(t *testing.T) {
	t.Run("GetTypeName returns TypeName", func(t *testing.T) {
		info := &VariableInfo{
			TypeName:     "os.File",
			Dependencies: []string{},
		}
		if got := info.GetTypeName(); got != "os.File" {
			t.Errorf("GetTypeName() = %q, want %q", got, "os.File")
		}
	})

	t.Run("GetDependencies returns empty slice", func(t *testing.T) {
		info := &VariableInfo{
			TypeName:     "os.File",
			Dependencies: []string{},
		}
		deps := info.GetDependencies()
		if len(deps) != 0 {
			t.Errorf("GetDependencies() = %v, want empty slice", deps)
		}
	})

	t.Run("GetName returns Name", func(t *testing.T) {
		info := &VariableInfo{
			TypeName:     "os.File",
			Name:         "output",
			Dependencies: []string{},
		}
		if got := info.GetName(); got != "output" {
			t.Errorf("GetName() = %q, want %q", got, "output")
		}
	})

	t.Run("GetName returns empty string for unnamed variable", func(t *testing.T) {
		info := &VariableInfo{
			TypeName:     "os.File",
			Name:         "",
			Dependencies: []string{},
		}
		if got := info.GetName(); got != "" {
			t.Errorf("GetName() = %q, want empty string", got)
		}
	})
}

func TestVariableRegistry_ThreadSafety(t *testing.T) {
	r := NewVariableRegistry()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 types of operations

	// Concurrent writes
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for range numOperations {
				info := &VariableInfo{
					TypeName:       "example.com/pkg.Type" + string(rune('A'+id%26)),
					LocalName:      "Type" + string(rune('A'+id%26)),
					ExpressionText: "expr",
					Dependencies:   []string{},
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
	variables := r.GetAll()
	if variables == nil {
		t.Fatal("GetAll() returned nil after concurrent operations")
	}

	// Verify alphabetical ordering is maintained
	typeNames := make([]string, len(variables))
	for i, v := range variables {
		typeNames[i] = v.TypeName
	}
	if !sort.StringsAreSorted(typeNames) {
		t.Error("GetAll() did not return variables in sorted order after concurrent operations")
	}
}

func TestVariableRegistry_GetNamesByType(t *testing.T) {
	t.Run("returns nil for non-existent type", func(t *testing.T) {
		r := NewVariableRegistry()

		got := r.GetNamesByType("nonexistent.Type")
		if got != nil {
			t.Errorf("GetNamesByType(nonexistent) = %v, want nil", got)
		}
	})

	t.Run("returns nil when only unnamed variable exists", func(t *testing.T) {
		r := NewVariableRegistry()

		r.Register(&VariableInfo{TypeName: "os.File", Name: "", ExpressionText: "os.Stdout", Dependencies: []string{}})

		got := r.GetNamesByType("os.File")
		if got != nil {
			t.Errorf("GetNamesByType(unnamed only) = %v, want nil", got)
		}
	})

	t.Run("returns sorted names for named variables", func(t *testing.T) {
		r := NewVariableRegistry()

		r.Register(&VariableInfo{TypeName: "os.File", Name: "beta", ExpressionText: "os.Stderr", Dependencies: []string{}})
		r.Register(&VariableInfo{TypeName: "os.File", Name: "alpha", ExpressionText: "os.Stdout", Dependencies: []string{}})

		got := r.GetNamesByType("os.File")
		if len(got) != 2 {
			t.Fatalf("len(GetNamesByType) = %d, want 2", len(got))
		}
		if got[0] != "alpha" || got[1] != "beta" {
			t.Errorf("GetNamesByType = %v, want [\"alpha\", \"beta\"]", got)
		}
	})

	t.Run("excludes empty name from results", func(t *testing.T) {
		r := NewVariableRegistry()

		r.Register(&VariableInfo{TypeName: "os.File", Name: "", ExpressionText: "os.Stdin", Dependencies: []string{}})
		r.Register(&VariableInfo{TypeName: "os.File", Name: "output", ExpressionText: "os.Stdout", Dependencies: []string{}})

		got := r.GetNamesByType("os.File")
		if len(got) != 1 {
			t.Fatalf("len(GetNamesByType) = %d, want 1", len(got))
		}
		if got[0] != "output" {
			t.Errorf("GetNamesByType = %v, want [\"output\"]", got)
		}
	})
}

func TestVariableRegistry_ThreadSafety_GetByName(t *testing.T) {
	r := NewVariableRegistry()

	const numGoroutines = 50
	const numOperations = 100

	// Pre-register some named variables
	for i := range 26 {
		name := string(rune('a' + i))
		r.Register(&VariableInfo{
			TypeName:       "example.com/pkg.Type",
			Name:           name,
			ExpressionText: "expr" + name,
			Dependencies:   []string{},
		})
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Concurrent GetByName reads
	for range numGoroutines {
		go func() {
			defer wg.Done()
			for j := range numOperations {
				name := string(rune('a' + j%26))
				_, _ = r.GetByName("example.com/pkg.Type", name)
			}
		}()
	}

	// Concurrent GetNamesByType reads
	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range numOperations {
				_ = r.GetNamesByType("example.com/pkg.Type")
			}
		}()
	}

	wg.Wait()
}

func TestGlobalVariableRegistry(t *testing.T) {
	r := NewVariableRegistry()

	info := &VariableInfo{
		TypeName:       "os.File",
		PackagePath:    "os",
		PackageName:    "os",
		LocalName:      "File",
		ExpressionText: "os.Stdout",
		Dependencies:   []string{},
	}

	r.Register(info)

	got, ok := r.GetByName("os.File", "")
	if !ok {
		t.Fatal("VariableRegistry.GetByName() returned ok=false")
	}
	if got.TypeName != info.TypeName {
		t.Errorf("TypeName = %q, want %q", got.TypeName, info.TypeName)
	}
}
