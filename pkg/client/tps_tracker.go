package client

import (
	"sync"
	"time"
)

// tpsTracker tracks transactions per second internally within the SDK.
// This provides automatic TPS measurement when the application does not
// provide a custom TPSProvider helper function.
//
// Implementation uses a sliding window approach to count requests within
// the last second.
type tpsTracker struct {
	mu       sync.RWMutex
	requests []time.Time
	window   time.Duration
}

// newTPSTracker creates a new TPS tracker with a 1-second window
func newTPSTracker() *tpsTracker {
	return &tpsTracker{
		requests: make([]time.Time, 0, 100),
		window:   time.Second,
	}
}

// RecordRequest records a new request timestamp
// This should be called whenever a product-level API method is invoked
func (t *tpsTracker) RecordRequest() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.requests = append(t.requests, now)

	// Clean old requests outside the window
	// This prevents unbounded memory growth
	cutoff := now.Add(-t.window)
	validIdx := 0
	for i, req := range t.requests {
		if req.After(cutoff) {
			validIdx = i
			break
		}
	}

	// Only keep requests within the window
	if validIdx > 0 {
		t.requests = t.requests[validIdx:]
	}
}

// getCurrentRate returns the current transactions per second
// Counts all requests within the last window (default: 1 second)
func (t *tpsTracker) getCurrentRate() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-t.window)

	count := 0
	for _, req := range t.requests {
		if req.After(cutoff) {
			count++
		}
	}

	return float64(count)
}

// Reset clears all tracked requests
// Useful for testing or when resetting metrics
func (t *tpsTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.requests = make([]time.Time, 0, 100)
}
