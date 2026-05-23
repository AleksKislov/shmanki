package middleware

import (
	"net/http"
	"sync"
	"time"

	"shmanki/internal/platform/response"
)

// RateLimiter is a simple in-memory token-bucket limiter keyed by user ID
// (or remote address for unauthenticated routes). It is not distributed
// across instances; for multi-instance deployments replace with a shared
// store (Redis, etc.). Suitable as a defense-in-depth layer against abuse
// of expensive endpoints such as LLM generation.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	capacity float64
	refill   float64 // tokens per second
}

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// NewRateLimiter creates a limiter that permits `capacity` requests in a
// burst and refills `capacity` tokens every `window`.
func NewRateLimiter(capacity int, window time.Duration) *RateLimiter {
	if capacity <= 0 {
		capacity = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		capacity: float64(capacity),
		refill:   float64(capacity) / window.Seconds(),
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) allow(key string) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &bucket{tokens: rl.capacity - 1, lastSeen: now}
		return true
	}
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens += elapsed * rl.refill
	if b.tokens > rl.capacity {
		b.tokens = rl.capacity
	}
	b.lastSeen = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-30 * time.Minute)
		rl.mu.Lock()
		for k, b := range rl.buckets {
			if b.lastSeen.Before(cutoff) {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}

// PerUser returns middleware that rate-limits by user ID from the request
// context. If no user ID is present, falls back to remote address.
func (rl *RateLimiter) PerUser() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromContext(r.Context())
			key := userID.String()
			if userID.String() == "00000000-0000-0000-0000-000000000000" {
				key = "ip:" + r.RemoteAddr
			}
			if !rl.allow(key) {
				response.WriteError(w, http.StatusTooManyRequests, "rate limit exceeded", "RATE_LIMITED")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
