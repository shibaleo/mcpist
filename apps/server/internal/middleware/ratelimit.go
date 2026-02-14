package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements per-user sliding window rate limiting.
// Uses in-memory state — each Go Server instance enforces independently.
type RateLimiter struct {
	maxRequests int
	window      time.Duration
	mu          sync.Mutex
	users       map[string]*userWindow
}

type userWindow struct {
	timestamps []time.Time
	lastAccess time.Time
}

// NewRateLimiter creates a rate limiter with the given requests-per-second limit.
func NewRateLimiter(maxPerSecond int) *RateLimiter {
	rl := &RateLimiter{
		maxRequests: maxPerSecond,
		window:      time.Second,
		users:       make(map[string]*userWindow),
	}
	go rl.cleanup()
	return rl
}

// Allow checks if a request from the given user is allowed.
func (rl *RateLimiter) Allow(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	uw, ok := rl.users[userID]
	if !ok {
		uw = &userWindow{}
		rl.users[userID] = uw
	}

	// Remove timestamps outside the window
	cutoff := now.Add(-rl.window)
	start := 0
	for start < len(uw.timestamps) && uw.timestamps[start].Before(cutoff) {
		start++
	}
	uw.timestamps = uw.timestamps[start:]
	uw.lastAccess = now

	if len(uw.timestamps) >= rl.maxRequests {
		return false
	}

	uw.timestamps = append(uw.timestamps, now)
	return true
}

// cleanup removes stale user entries every 60 seconds.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-5 * time.Minute)
		for userID, uw := range rl.users {
			if uw.lastAccess.Before(cutoff) {
				delete(rl.users, userID)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns an HTTP middleware that applies rate limiting.
// Must be placed AFTER Authorize middleware (reads userID from context).
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r.Context())
		if authCtx == nil {
			next.ServeHTTP(w, r)
			return
		}
		userID := authCtx.UserID

		if !rl.Allow(userID) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "RATE_LIMIT_EXCEEDED",
				"message": "Too many requests. Please slow down or use batch mode.",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
