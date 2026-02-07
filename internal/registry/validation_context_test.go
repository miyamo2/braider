package registry

import (
	"testing"
	"time"
)

func TestValidationContext_Creation(t *testing.T) {
	vc := NewValidationContext()
	if vc == nil {
		t.Fatal("NewValidationContext should return non-nil context")
	}

	// Initially not cancelled
	if vc.IsCancelled() {
		t.Error("New context should not be cancelled")
	}
}

func TestValidationContext_Cancel(t *testing.T) {
	vc := NewValidationContext()

	// Cancel the context
	vc.Cancel()

	// Should be cancelled now
	if !vc.IsCancelled() {
		t.Error("Context should be cancelled after Cancel()")
	}

	// Get should return a cancelled context
	ctx := vc.Get()
	select {
	case <-ctx.Done():
		// Expected: context is cancelled
	case <-time.After(10 * time.Millisecond):
		t.Error("Context.Done() should be closed after Cancel()")
	}
}

func TestValidationContext_Reset(t *testing.T) {
	vc := NewValidationContext()

	// Cancel the context
	vc.Cancel()
	if !vc.IsCancelled() {
		t.Error("Context should be cancelled after Cancel()")
	}

	// Reset
	vc.Reset()

	// Should not be cancelled after reset
	if vc.IsCancelled() {
		t.Error("Context should not be cancelled after Reset()")
	}
}

func TestValidationContext_ConcurrentAccess(t *testing.T) {
	vc := NewValidationContext()

	// Simulate concurrent access
	done := make(chan bool)

	// Goroutine 1: Check cancellation
	go func() {
		for i := 0; i < 100; i++ {
			_ = vc.IsCancelled()
		}
		done <- true
	}()

	// Goroutine 2: Get context
	go func() {
		for i := 0; i < 100; i++ {
			_ = vc.Get()
		}
		done <- true
	}()

	// Goroutine 3: Cancel
	go func() {
		time.Sleep(1 * time.Millisecond)
		vc.Cancel()
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done

	// After all operations, context should be cancelled
	if !vc.IsCancelled() {
		t.Error("Context should be cancelled after concurrent Cancel()")
	}
}
