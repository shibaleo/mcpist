package mcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"mcpist/server/internal/jsonrpc"
	"mcpist/server/internal/middleware"
)

func TestAuthErrorToRPC(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{
			"usage limit exceeded",
			&middleware.AuthError{Code: "USAGE_LIMIT_EXCEEDED", Message: "limit hit", Status: http.StatusTooManyRequests},
			ErrUsageLimitExceeded,
		},
		{
			"module not enabled",
			&middleware.AuthError{Code: "MODULE_NOT_ENABLED", Message: "no access", Status: http.StatusForbidden},
			ErrPermissionDenied,
		},
		{
			"tool disabled",
			&middleware.AuthError{Code: "TOOL_DISABLED", Message: "tool off", Status: http.StatusForbidden},
			ErrPermissionDenied,
		},
		{
			"unknown auth error code",
			&middleware.AuthError{Code: "SOMETHING_ELSE", Message: "other", Status: http.StatusInternalServerError},
			InternalError,
		},
		{
			"non-AuthError",
			fmt.Errorf("plain error"),
			InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpcErr := authErrorToRPC(tt.err)
			if rpcErr.Code != tt.wantCode {
				t.Errorf("code = %d, want %d", rpcErr.Code, tt.wantCode)
			}
		})
	}
}

func TestCheckBatchPermissions(t *testing.T) {
	authCtx := &middleware.AuthContext{
		DailyUsed:      5,
		DailyLimit:     50,
		EnabledModules: []string{"notion", "github"},
		EnabledTools: map[string][]string{
			"notion": {"notion:search", "notion:get_page_content"},
			"github": {"github:list_issues"},
		},
	}

	tests := []struct {
		name     string
		commands string
		wantErr  bool
		errMsg   string
	}{
		{
			"all permitted",
			`{"module":"notion","tool":"search","params":{}}
{"module":"github","tool":"list_issues","params":{}}`,
			false,
			"",
		},
		{
			"one denied tool",
			`{"module":"notion","tool":"search","params":{}}
{"module":"notion","tool":"delete_page","params":{}}`,
			true,
			"batch rejected: one or more tools are not permitted",
		},
		{
			"disabled module",
			`{"module":"dropbox","tool":"list_files","params":{}}`,
			true,
			"batch rejected: one or more tools are not permitted",
		},
		{
			"empty commands",
			"",
			false,
			"",
		},
		{
			"blank lines skipped",
			`{"module":"notion","tool":"search","params":{}}

{"module":"github","tool":"list_issues","params":{}}
`,
			false,
			"",
		},
		{
			"malformed JSON lines skipped",
			`not-json
{"module":"notion","tool":"search","params":{}}`,
			false,
			"",
		},
		{
			"missing module/tool fields skipped",
			`{"module":"","tool":"search"}
{"module":"notion","tool":"search","params":{}}`,
			false,
			"",
		},
		{
			"batch size exceeds limit",
			strings.Repeat("{\"module\":\"notion\",\"tool\":\"search\",\"params\":{}}\n", 11),
			true,
			"batch too large: 11 commands (max 10)",
		},
		{
			"batch at exact limit",
			strings.Repeat("{\"module\":\"notion\",\"tool\":\"search\",\"params\":{}}\n", 10),
			false,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpcErr := checkBatchPermissions("test-req-id", authCtx, tt.commands)
			if tt.wantErr {
				if rpcErr == nil {
					t.Fatal("expected error, got nil")
				}
				if rpcErr.Message != tt.errMsg {
					t.Errorf("message = %q, want %q", rpcErr.Message, tt.errMsg)
				}
			} else {
				if rpcErr != nil {
					t.Errorf("unexpected error: %v", rpcErr.Message)
				}
			}
		})
	}
}

func TestCheckBatchPermissionsDailyLimit(t *testing.T) {
	authCtx := &middleware.AuthContext{
		DailyUsed:      48,
		DailyLimit:     50,
		EnabledModules: []string{"notion"},
		EnabledTools: map[string][]string{
			"notion": {"notion:search"},
		},
	}

	// 3 commands with 48 used, limit 50 → 48+3=51 > 50
	commands := `{"module":"notion","tool":"search","params":{}}
{"module":"notion","tool":"search","params":{}}
{"module":"notion","tool":"search","params":{}}`

	rpcErr := checkBatchPermissions("test-req-id", authCtx, commands)
	if rpcErr == nil {
		t.Fatal("expected usage limit error")
	}
	if rpcErr.Code != ErrUsageLimitExceeded {
		t.Errorf("code = %d, want %d", rpcErr.Code, ErrUsageLimitExceeded)
	}
}

func TestHandleInitialize(t *testing.T) {
	h := NewHandler(nil)
	req := &jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}

	result := h.handleInitialize(req)
	if result.ProtocolVersion != "2025-03-26" {
		t.Errorf("protocolVersion = %q, want %q", result.ProtocolVersion, "2025-03-26")
	}
	if result.ServerInfo.Name != "mcpist" {
		t.Errorf("serverInfo.name = %q, want %q", result.ServerInfo.Name, "mcpist")
	}
	if result.Capabilities.Tools == nil {
		t.Error("expected tools capability to be non-nil")
	}
	if result.Capabilities.Prompts == nil {
		t.Error("expected prompts capability to be non-nil")
	}
}

func TestProcessRequestMethodNotFound(t *testing.T) {
	h := NewHandler(nil)
	req := &jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "nonexistent/method",
	}

	_, rpcErr := h.ProcessRequest(context.TODO(), req)
	if rpcErr == nil {
		t.Fatal("expected error for unknown method")
	}
	if rpcErr.Code != MethodNotFound {
		t.Errorf("code = %d, want %d", rpcErr.Code, MethodNotFound)
	}
}

func TestProcessRequestInitialized(t *testing.T) {
	h := NewHandler(nil)
	req := &jsonrpc.Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "initialized",
	}

	result, rpcErr := h.ProcessRequest(context.TODO(), req)
	if rpcErr != nil {
		t.Errorf("unexpected error: %v", rpcErr.Message)
	}
	if result != nil {
		t.Errorf("expected nil result for initialized, got %v", result)
	}
}
