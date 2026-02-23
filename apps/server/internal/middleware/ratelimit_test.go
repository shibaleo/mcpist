package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	rl := &RateLimiter{
		maxRequests: 3,
		window:      time.Second,
		users:       make(map[string]*userWindow),
	}

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !rl.Allow("user1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	if rl.Allow("user1") {
		t.Error("request 4 should be denied")
	}
}

func TestRateLimiterWindowRecovery(t *testing.T) {
	rl := &RateLimiter{
		maxRequests: 2,
		window:      50 * time.Millisecond,
		users:       make(map[string]*userWindow),
	}

	// Exhaust the limit
	rl.Allow("user1")
	rl.Allow("user1")
	if rl.Allow("user1") {
		t.Error("should be denied after exhausting limit")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow("user1") {
		t.Error("should be allowed after window expiry")
	}
}

func TestRateLimiterUserIsolation(t *testing.T) {
	rl := &RateLimiter{
		maxRequests: 1,
		window:      time.Second,
		users:       make(map[string]*userWindow),
	}

	// user1 exhausts limit
	if !rl.Allow("user1") {
		t.Error("user1 first request should be allowed")
	}
	if rl.Allow("user1") {
		t.Error("user1 second request should be denied")
	}

	// user2 should still be allowed
	if !rl.Allow("user2") {
		t.Error("user2 should be allowed (independent window)")
	}
}
