package registry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/provide"
)

// PackageStatusPollInterval is the interval for checking package scan status.
const PackageStatusPollInterval = 10 * time.Millisecond

var _ = annotation.Provide[provide.Default](NewPackageTracker)

// PackageTracker tracks scanned packages and provides synchronization.
// Thread-safe for parallel analyzer execution.
//
// Concurrency safety:
//   - All public methods are protected by mutex to prevent race conditions
//   - MarkPackageScanned can be called concurrently from multiple DependencyAnalyzer runs
//   - WaitForAllPackagesWithContext uses polling to handle channel recreation safely
//   - Channel notifications are best-effort; the polling mechanism ensures correctness
//
// The polling mechanism (PackageStatusPollInterval) ensures that even if channel
// notifications are lost during channel recreation, the waiting goroutine will
// eventually detect completion by periodically checking scannedPackages.
type PackageTracker struct {
	mu              sync.Mutex
	scannedPackages map[string]bool
	completionChan  chan struct{}
}

// NewPackageTracker creates a new empty tracker.
func NewPackageTracker() *PackageTracker {
	return &PackageTracker{
		scannedPackages: make(map[string]bool),
	}
}

// MarkPackageScanned marks a package as scanned.
// Called by DependencyAnalyzer at the end of Run().
// This method sends a notification to the completion channel for
// any waiting WaitForAllPackagesWithContext calls.
func (t *PackageTracker) MarkPackageScanned(pkgPath string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.scannedPackages[pkgPath] {
		// Already scanned, no need to notify again
		return
	}

	t.scannedPackages[pkgPath] = true

	// Send notification (non-blocking)
	// Channel may be nil if WaitForAllPackagesWithContext hasn't been called yet
	if t.completionChan != nil {
		select {
		case t.completionChan <- struct{}{}:
		default:
			// Channel full, but that's okay - the status is recorded in scannedPackages
		}
	}
}

// WaitForAllPackagesWithContext waits until all expected packages are scanned.
// Called by AppAnalyzer after detecting annotation.App.
// Returns error if context is cancelled or times out before all packages are scanned.
func (t *PackageTracker) WaitForAllPackagesWithContext(ctx context.Context, expectedPkgs []string) error {
	if len(expectedPkgs) == 0 {
		return nil
	}

	// Build set of expected packages
	expected := make(map[string]bool, len(expectedPkgs))
	for _, pkg := range expectedPkgs {
		expected[pkg] = true
	}

	// Ensure channel has sufficient capacity for expected packages
	// Note: Channel recreation here is safe even if MarkPackageScanned is called
	// concurrently, because we have a polling mechanism (time.After) that periodically
	// rechecks the scannedPackages map regardless of channel notifications.
	t.mu.Lock()
	if t.completionChan == nil || cap(t.completionChan) < len(expectedPkgs) {
		t.completionChan = make(chan struct{}, len(expectedPkgs))
	}
	t.mu.Unlock()

	for {
		// Check if all expected packages are scanned (subset check)
		t.mu.Lock()
		allScanned := true
		for pkg := range expected {
			if !t.scannedPackages[pkg] {
				allScanned = false
				break
			}
		}
		t.mu.Unlock()

		if allScanned {
			return nil
		}

		// Wait for a package notification, poll interval, or context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for packages: %w", ctx.Err())
		case <-t.completionChan:
			// A package was scanned, check again
		case <-time.After(PackageStatusPollInterval):
			// Poll interval to check status
		}
	}
}
