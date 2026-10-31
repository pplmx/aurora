// The per-source request budget: how often one oracle feed may be fetched
// inside a window. It is a separate file from the fetcher because the scheduler
// consults it directly, and because the sweep that bounds its map has its own
// failure mode — a source removed from the store must not keep its key forever.
package http

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (r *RateLimiter) Allow(sourceID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-r.window)

	// Reclaim keys idle past the window so the requests map stays bounded by
	// sources active within a window rather than every source ever fetched
	// (Reset exists but was never called, so keys leaked forever; TASK-136,
	// ISS-129). Per-source slices are pruned below, so an expired key has no
	// remaining recent entry.
	if len(r.requests) >= limiterSweepThreshold {
		for key, ts := range r.requests {
			if len(ts) == 0 || ts[len(ts)-1].Before(windowStart) {
				delete(r.requests, key)
			}
		}
	}

	requests := r.requests[sourceID]
	valid := make([]time.Time, 0, len(requests))
	for _, t := range requests {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= r.limit {
		r.requests[sourceID] = valid
		return false
	}

	valid = append(valid, now)
	r.requests[sourceID] = valid
	return true
}

func (r *RateLimiter) Reset(sourceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.requests, sourceID)
}

func (r *RateLimiter) Remaining(sourceID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	windowStart := now.Add(-r.window)

	count := 0
	for _, t := range r.requests[sourceID] {
		if t.After(windowStart) {
			count++
		}
	}

	remaining := r.limit - count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// limiterSweepThreshold bounds the RateLimiter's requests map: when it grows
// past this many tracked sources, Allow sweeps keys idle beyond the window.
// Without it every sourceID ever fetched stayed in the map forever, since
// Reset was never wired to the source lifecycle (TASK-136, ISS-129).
const limiterSweepThreshold = 1024
