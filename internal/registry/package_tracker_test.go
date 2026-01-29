package registry

import (
	"sync"
	"testing"
	"time"
)

// SetTimeout sets the timeout for WaitForAllPackages.
// Used for testing with shorter timeouts.
func (t *PackageTracker) SetTimeout(timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.timeout = timeout
}

func TestPackageTracker_MarkPackageScanned(t *testing.T) {
	t.Run(
		"marks a single package as scanned", func(t *testing.T) {
			pt := NewPackageTracker()

			pt.MarkPackageScanned("example.com/repo")

			if !pt.IsPackageScanned("example.com/repo") {
				t.Error("expected package to be marked as scanned")
			}
		},
	)

	t.Run(
		"marks multiple packages as scanned", func(t *testing.T) {
			pt := NewPackageTracker()

			packages := []string{
				"example.com/repo",
				"example.com/service",
				"example.com/handler",
			}

			for _, pkg := range packages {
				pt.MarkPackageScanned(pkg)
			}

			for _, pkg := range packages {
				if !pt.IsPackageScanned(pkg) {
					t.Errorf("expected package %q to be marked as scanned", pkg)
				}
			}
		},
	)

	t.Run(
		"marking same package twice is idempotent", func(t *testing.T) {
			pt := NewPackageTracker()

			pt.MarkPackageScanned("example.com/repo")
			pt.MarkPackageScanned("example.com/repo")

			if !pt.IsPackageScanned("example.com/repo") {
				t.Error("expected package to be marked as scanned")
			}
		},
	)
}

func TestPackageTracker_IsPackageScanned(t *testing.T) {
	t.Run(
		"returns false for unscanned package", func(t *testing.T) {
			pt := NewPackageTracker()

			if pt.IsPackageScanned("example.com/nonexistent") {
				t.Error("expected false for unscanned package")
			}
		},
	)

	t.Run(
		"returns true for scanned package", func(t *testing.T) {
			pt := NewPackageTracker()

			pt.MarkPackageScanned("example.com/repo")

			if !pt.IsPackageScanned("example.com/repo") {
				t.Error("expected true for scanned package")
			}
		},
	)
}

func TestPackageTracker_WaitForAllPackages_AllScanned(t *testing.T) {
	t.Run(
		"returns immediately when all packages already scanned", func(t *testing.T) {
			pt := NewPackageTracker()

			packages := []string{
				"example.com/repo",
				"example.com/service",
			}

			// Mark all packages as scanned before waiting
			for _, pkg := range packages {
				pt.MarkPackageScanned(pkg)
			}

			start := time.Now()
			err := pt.WaitForAllPackages(packages)
			elapsed := time.Since(start)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if elapsed > 100*time.Millisecond {
				t.Errorf("WaitForAllPackages took too long: %v", elapsed)
			}
		},
	)

	t.Run(
		"waits for packages to be scanned", func(t *testing.T) {
			pt := NewPackageTracker()

			packages := []string{
				"example.com/repo",
				"example.com/service",
			}

			// Start goroutine to mark packages after a short delay
			go func() {
				time.Sleep(50 * time.Millisecond)
				for _, pkg := range packages {
					pt.MarkPackageScanned(pkg)
				}
			}()

			err := pt.WaitForAllPackages(packages)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify all packages are scanned
			for _, pkg := range packages {
				if !pt.IsPackageScanned(pkg) {
					t.Errorf("expected package %q to be scanned", pkg)
				}
			}
		},
	)

	t.Run(
		"returns nil for empty package list", func(t *testing.T) {
			pt := NewPackageTracker()

			err := pt.WaitForAllPackages([]string{})
			if err != nil {
				t.Errorf("unexpected error for empty package list: %v", err)
			}
		},
	)
}

func TestPackageTracker_WaitForAllPackages_Timeout(t *testing.T) {
	t.Run(
		"returns error on timeout", func(t *testing.T) {
			pt := NewPackageTracker()

			// Set a very short timeout for testing
			pt.SetTimeout(100 * time.Millisecond)

			packages := []string{
				"example.com/repo",
				"example.com/service",
			}

			// Don't mark any packages as scanned - should timeout
			err := pt.WaitForAllPackages(packages)
			if err == nil {
				t.Error("expected timeout error, got nil")
			}
		},
	)
}

func TestPackageTracker_WaitForAllPackages_PreScanned(t *testing.T) {
	t.Run(
		"handles packages scanned before WaitForAllPackages is called", func(t *testing.T) {
			pt := NewPackageTracker()

			packages := []string{
				"example.com/repo",
				"example.com/service",
				"example.com/handler",
			}

			// Mark some packages before wait
			pt.MarkPackageScanned("example.com/repo")
			pt.MarkPackageScanned("example.com/service")

			// Mark remaining package after a short delay
			go func() {
				time.Sleep(50 * time.Millisecond)
				pt.MarkPackageScanned("example.com/handler")
			}()

			err := pt.WaitForAllPackages(packages)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		},
	)
}

func TestPackageTracker_ThreadSafety(t *testing.T) {
	pt := NewPackageTracker()

	const numGoroutines = 50
	const numPackages = 10

	var wg sync.WaitGroup

	// Generate package names
	packages := make([]string, numPackages)
	for i := range packages {
		packages[i] = "example.com/pkg" + string(rune('A'+i))
	}

	// Concurrent MarkPackageScanned
	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			pkgIdx := id % numPackages
			pt.MarkPackageScanned(packages[pkgIdx])
		}(i)
	}

	// Concurrent IsPackageScanned
	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			pkgIdx := id % numPackages
			_ = pt.IsPackageScanned(packages[pkgIdx])
		}(i)
	}

	wg.Wait()

	// All packages should be scanned at this point
	for _, pkg := range packages {
		if !pt.IsPackageScanned(pkg) {
			t.Errorf("expected package %q to be scanned after concurrent operations", pkg)
		}
	}
}

func TestGlobalPackageTracker(t *testing.T) {
	GlobalPackageTracker.MarkPackageScanned("example.com/test")

	if !GlobalPackageTracker.IsPackageScanned("example.com/test") {
		t.Error("GlobalPackageTracker.IsPackageScanned() returned false")
	}
}

func TestPackageTracker_WaitForAllPackages_PartialScanning(t *testing.T) {
	t.Run(
		"handles staggered package scanning", func(t *testing.T) {
			pt := NewPackageTracker()

			packages := []string{
				"example.com/pkg1",
				"example.com/pkg2",
				"example.com/pkg3",
			}

			// Mark packages one by one with delays
			go func() {
				for i, pkg := range packages {
					time.Sleep(time.Duration(20*(i+1)) * time.Millisecond)
					pt.MarkPackageScanned(pkg)
				}
			}()

			err := pt.WaitForAllPackages(packages)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		},
	)
}
