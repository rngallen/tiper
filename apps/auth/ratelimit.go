package auth

import (
	"sync"
	"time"
)

// RateLimiter is a small in-memory fixed-window limiter keyed by client IP,
// used to throttle login attempts. It is process-local (sufficient for a single
// service instance); a distributed deployment would back this with a shared
// store.
type RateLimiter struct {
	mu       sync.Mutex
	window   time.Duration
	max      int
	counters map[string]*rlEntry
	stop     chan struct{}
	stopped  bool
}

type rlEntry struct {
	count int
	until time.Time
}

func newRateLimiter(max int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		window:   window,
		max:      max,
		counters: make(map[string]*rlEntry),
		stop:     make(chan struct{}),
	}
	go rl.janitor()
	return rl
}

// Allowed reports whether the IP is currently under its failure budget.
func (rl *RateLimiter) Allowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	e, ok := rl.counters[ip]
	if !ok || time.Now().After(e.until) {
		return true
	}
	return e.count < rl.max
}

// RegisterFailure records a failed attempt for the IP.
func (rl *RateLimiter) RegisterFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	e, ok := rl.counters[ip]
	if !ok || time.Now().After(e.until) {
		rl.counters[ip] = &rlEntry{count: 1, until: time.Now().Add(rl.window)}
		return
	}
	e.count++
}

// Reset clears the failure budget for the IP (used on successful login).
func (rl *RateLimiter) Reset(ip string) {
	rl.mu.Lock()
	delete(rl.counters, ip)
	rl.mu.Unlock()
}

// Stop terminates the background janitor.
func (rl *RateLimiter) Stop() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if !rl.stopped {
		close(rl.stop)
		rl.stopped = true
	}
}

func (rl *RateLimiter) janitor() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, e := range rl.counters {
				if now.After(e.until) {
					delete(rl.counters, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}
