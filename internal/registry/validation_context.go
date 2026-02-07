package registry

import (
	"context"
	"sync"
)

// ValidationContext holds the cancellable context for coordinating validation errors between analyzers.
// When DependencyAnalyzer encounters fatal validation errors, it cancels this context to signal
// AppAnalyzer to skip bootstrap generation (Requirement 8.5).
// Thread-safe for concurrent access.
type ValidationContext struct {
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewValidationContext creates a new validation context holder.
func NewValidationContext() *ValidationContext {
	ctx, cancel := context.WithCancel(context.Background())
	return &ValidationContext{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Get returns the current context.
func (vc *ValidationContext) Get() context.Context {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.ctx
}

// Cancel cancels the validation context due to fatal error.
func (vc *ValidationContext) Cancel() {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	if vc.cancel != nil {
		vc.cancel()
	}
}

// IsCancelled checks if the validation context has been cancelled.
func (vc *ValidationContext) IsCancelled() bool {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	select {
	case <-vc.ctx.Done():
		return true
	default:
		return false
	}
}

// Reset resets the validation context (for testing).
func (vc *ValidationContext) Reset() {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	vc.ctx = ctx
	vc.cancel = cancel
}
