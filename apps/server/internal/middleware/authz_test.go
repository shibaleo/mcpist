package middleware

import (
	"testing"
)

func TestWithinDailyLimit(t *testing.T) {
	tests := []struct {
		name  string
		used  int
		limit int
		count int
		want  bool
	}{
		{"within limit", 5, 50, 1, true},
		{"at limit", 50, 50, 0, true},
		{"exactly reaching limit", 49, 50, 1, true},
		{"exceeds limit by 1", 50, 50, 1, false},
		{"far exceeds limit", 100, 50, 1, false},
		{"zero usage", 0, 50, 1, true},
		{"batch within limit", 5, 50, 10, true},
		{"batch exceeds limit", 45, 50, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &AuthContext{
				DailyUsed:  tt.used,
				DailyLimit: tt.limit,
			}
			if got := ctx.WithinDailyLimit(tt.count); got != tt.want {
				t.Errorf("WithinDailyLimit(%d) = %v, want %v (used=%d, limit=%d)",
					tt.count, got, tt.want, tt.used, tt.limit)
			}
		})
	}
}

func TestCanAccessModule(t *testing.T) {
	ctx := &AuthContext{
		EnabledModules: []string{"notion", "github", "jira"},
	}

	tests := []struct {
		name    string
		module  string
		wantErr bool
		errCode string
	}{
		{"enabled module", "notion", false, ""},
		{"another enabled module", "github", false, ""},
		{"disabled module", "dropbox", true, "MODULE_NOT_ENABLED"},
		{"empty module name", "", true, "MODULE_NOT_ENABLED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ctx.CanAccessModule(tt.module)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				authErr, ok := err.(*AuthError)
				if !ok {
					t.Fatalf("expected *AuthError, got %T", err)
				}
				if authErr.Code != tt.errCode {
					t.Errorf("error code = %q, want %q", authErr.Code, tt.errCode)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCanAccessTool(t *testing.T) {
	ctx := &AuthContext{
		DailyUsed:  5,
		DailyLimit: 50,
		EnabledModules: []string{"notion", "github"},
		EnabledTools: map[string][]string{
			"notion": {"notion:search", "notion:get_page_content"},
			"github": {"github:list_issues", "github:create_issue"},
		},
	}

	tests := []struct {
		name       string
		module     string
		tool       string
		usageCount int
		wantErr    bool
		errCode    string
	}{
		{"enabled tool", "notion", "search", 1, false, ""},
		{"another enabled tool", "github", "list_issues", 1, false, ""},
		{"disabled tool", "notion", "delete_page", 1, true, "TOOL_DISABLED"},
		{"disabled module", "dropbox", "list_files", 1, true, "MODULE_NOT_ENABLED"},
		{"exceeds daily limit", "notion", "search", 46, true, "USAGE_LIMIT_EXCEEDED"},
		{"zero usage count", "notion", "search", 0, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ctx.CanAccessTool(tt.module, tt.tool, tt.usageCount)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				authErr, ok := err.(*AuthError)
				if !ok {
					t.Fatalf("expected *AuthError, got %T", err)
				}
				if authErr.Code != tt.errCode {
					t.Errorf("error code = %q, want %q", authErr.Code, tt.errCode)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCanAccessToolHTTPStatus(t *testing.T) {
	ctx := &AuthContext{
		DailyUsed:  50,
		DailyLimit: 50,
		EnabledTools: map[string][]string{
			"notion": {"notion:search"},
		},
		EnabledModules: []string{"notion"},
	}

	tests := []struct {
		name       string
		module     string
		tool       string
		usageCount int
		wantStatus int
	}{
		{"module not enabled → 403", "dropbox", "list_files", 1, 403},
		{"tool disabled → 403", "notion", "delete_page", 1, 403},
		{"usage exceeded → 429", "notion", "search", 1, 429},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ctx.CanAccessTool(tt.module, tt.tool, tt.usageCount)
			if err == nil {
				t.Fatal("expected error")
			}
			authErr := err.(*AuthError)
			if authErr.Status != tt.wantStatus {
				t.Errorf("HTTP status = %d, want %d", authErr.Status, tt.wantStatus)
			}
		})
	}
}
