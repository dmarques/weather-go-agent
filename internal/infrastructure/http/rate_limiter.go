package http

import (
	"sync"
	"time"

	"github.com/dmarques/weather-go-agent/pkg/constant"
)

type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*rateEntry
}

type rateEntry struct {
	started time.Time
	count   int
}

// NewRateLimiter creates a new RateLimiter instance with an empty map of clients.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{clients: make(map[string]*rateEntry)}
}

// Allow checks if a request from the given key (e.g., client IP) is allowed based on the rate limiting rules.
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	entry, ok := r.clients[key]
	if !ok || now.Sub(entry.started) >= constant.PerIPWindow {
		r.clients[key] = &rateEntry{started: now, count: 1}
		return true
	}

	if entry.count >= constant.PerIPLimit {
		return false
	}
	entry.count++
	return true
}
