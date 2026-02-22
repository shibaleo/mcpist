package broker

import (
	"testing"
	"time"
)

func TestUserContextWithinDailyLimit(t *testing.T) {
	tests := []struct {
		name  string
		used  int
		limit int
		count int
		want  bool
	}{
		{"within limit", 5, 50, 1, true},
		{"at limit", 50, 50, 0, true},
		{"exceeds limit", 50, 50, 1, false},
		{"batch within limit", 40, 50, 10, true},
		{"batch exceeds limit", 45, 50, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &UserContext{DailyUsed: tt.used, DailyLimit: tt.limit}
			if got := uc.WithinDailyLimit(tt.count); got != tt.want {
				t.Errorf("WithinDailyLimit(%d) = %v, want %v", tt.count, got, tt.want)
			}
		})
	}
}

func TestUserContextIsModuleEnabled(t *testing.T) {
	uc := &UserContext{
		EnabledModules: []string{"notion", "github", "jira"},
	}

	tests := []struct {
		module string
		want   bool
	}{
		{"notion", true},
		{"github", true},
		{"dropbox", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			if got := uc.IsModuleEnabled(tt.module); got != tt.want {
				t.Errorf("IsModuleEnabled(%q) = %v, want %v", tt.module, got, tt.want)
			}
		})
	}
}

func TestUserContextIsToolEnabled(t *testing.T) {
	uc := &UserContext{
		EnabledTools: map[string][]string{
			"notion": {"notion:search", "notion:get_page_content"},
			"github": {"github:list_issues"},
		},
	}

	tests := []struct {
		name   string
		module string
		tool   string
		want   bool
	}{
		{"enabled tool", "notion", "notion:search", true},
		{"another enabled tool", "github", "github:list_issues", true},
		{"disabled tool", "notion", "notion:delete_page", false},
		{"disabled module", "dropbox", "dropbox:list_files", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uc.IsToolEnabled(tt.module, tt.tool); got != tt.want {
				t.Errorf("IsToolEnabled(%q, %q) = %v, want %v", tt.module, tt.tool, got, tt.want)
			}
		})
	}
}

func TestCacheGetSetExpiry(t *testing.T) {
	c := &userCache{
		items: make(map[string]*userCacheItem),
		ttl:   50 * time.Millisecond,
	}

	ctx := &UserContext{AccountStatus: "active", PlanID: "free"}
	c.set("user1", ctx)

	// Should return cached value
	if got := c.get("user1"); got == nil {
		t.Fatal("expected cached value, got nil")
	}

	// Wait for expiry
	time.Sleep(60 * time.Millisecond)

	// Should return nil (expired)
	if got := c.get("user1"); got != nil {
		t.Error("expected nil after expiry")
	}

	// getStale should still return it
	if got := c.getStale("user1"); got == nil {
		t.Error("expected stale value, got nil")
	}
}

func TestCacheDelete(t *testing.T) {
	c := &userCache{
		items: make(map[string]*userCacheItem),
		ttl:   time.Minute,
	}

	ctx := &UserContext{AccountStatus: "active"}
	c.set("user1", ctx)
	c.delete("user1")

	if got := c.get("user1"); got != nil {
		t.Error("expected nil after delete")
	}
	if got := c.getStale("user1"); got != nil {
		t.Error("expected nil stale after delete")
	}
}

func TestCacheMiss(t *testing.T) {
	c := &userCache{
		items: make(map[string]*userCacheItem),
		ttl:   time.Minute,
	}

	if got := c.get("nonexistent"); got != nil {
		t.Error("expected nil for cache miss")
	}
	if got := c.getStale("nonexistent"); got != nil {
		t.Error("expected nil stale for cache miss")
	}
}
