package modules

import (
	"sync"
	"testing"
)

func TestFilterTools(t *testing.T) {
	tools := []Tool{
		{ID: "notion:search", Name: "search"},
		{ID: "notion:get_page_content", Name: "get_page_content"},
		{ID: "notion:delete_page", Name: "delete_page"},
	}

	tests := []struct {
		name         string
		moduleName   string
		enabledTools map[string][]string
		wantCount    int
	}{
		{
			"nil enabledTools returns all",
			"notion",
			nil,
			3,
		},
		{
			"partial whitelist",
			"notion",
			map[string][]string{
				"notion": {"notion:search", "notion:get_page_content"},
			},
			2,
		},
		{
			"module not in enabledTools",
			"notion",
			map[string][]string{
				"github": {"github:list_issues"},
			},
			0,
		},
		{
			"empty whitelist for module",
			"notion",
			map[string][]string{
				"notion": {},
			},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterTools(tt.moduleName, tools, tt.enabledTools)
			if len(got) != tt.wantCount {
				t.Errorf("filterTools() returned %d tools, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestDetectCycle(t *testing.T) {
	tests := []struct {
		name      string
		tasks     map[string]*taskState
		wantCycle bool
	}{
		{
			"no cycle (linear)",
			map[string]*taskState{
				"a": {cmd: BatchCommand{ID: "a", After: nil}},
				"b": {cmd: BatchCommand{ID: "b", After: []string{"a"}}},
				"c": {cmd: BatchCommand{ID: "c", After: []string{"b"}}},
			},
			false,
		},
		{
			"no cycle (independent)",
			map[string]*taskState{
				"a": {cmd: BatchCommand{ID: "a"}},
				"b": {cmd: BatchCommand{ID: "b"}},
			},
			false,
		},
		{
			"cycle A→B→A",
			map[string]*taskState{
				"a": {cmd: BatchCommand{ID: "a", After: []string{"b"}}},
				"b": {cmd: BatchCommand{ID: "b", After: []string{"a"}}},
			},
			true,
		},
		{
			"self-reference",
			map[string]*taskState{
				"a": {cmd: BatchCommand{ID: "a", After: []string{"a"}}},
			},
			true,
		},
		{
			"empty tasks",
			map[string]*taskState{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectCycle(tt.tasks)
			if tt.wantCycle && result == "" {
				t.Error("expected cycle, got empty string")
			}
			if !tt.wantCycle && result != "" {
				t.Errorf("expected no cycle, got %q", result)
			}
		})
	}
}

func TestResolveStringVariables(t *testing.T) {
	store := &sync.Map{}
	store.Store("search", `[{"id":"page-123","title":"Design"}]`)
	store.Store("nested", `{"results":[{"id":"abc-456","name":"nested"}]}`)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"resolve from array",
			"${search.results[0].id}",
			"page-123",
		},
		{
			"resolve from object with results key",
			"${nested.results[0].id}",
			"abc-456",
		},
		{
			"no variable reference",
			"plain string",
			"plain string",
		},
		{
			"unknown task ID",
			"${unknown.results[0].id}",
			"${unknown.results[0].id}",
		},
		{
			"out of bounds index",
			"${search.results[99].id}",
			"${search.results[99].id}",
		},
		{
			"embedded in text",
			"Page ID is ${search.results[0].id} here",
			"Page ID is page-123 here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveStringVariables(tt.input, store)
			if got != tt.want {
				t.Errorf("resolveStringVariables(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveVariables(t *testing.T) {
	store := &sync.Map{}
	store.Store("task1", `[{"id":"abc"}]`)

	t.Run("nil params", func(t *testing.T) {
		got := resolveVariables(nil, store)
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("nested map with variable", func(t *testing.T) {
		params := map[string]interface{}{
			"page_id": "${task1.results[0].id}",
			"nested": map[string]interface{}{
				"ref": "${task1.results[0].id}",
			},
		}
		got := resolveVariables(params, store)
		if got["page_id"] != "abc" {
			t.Errorf("page_id = %q, want %q", got["page_id"], "abc")
		}
		nested := got["nested"].(map[string]interface{})
		if nested["ref"] != "abc" {
			t.Errorf("nested.ref = %q, want %q", nested["ref"], "abc")
		}
	})

	t.Run("array with variable", func(t *testing.T) {
		params := map[string]interface{}{
			"ids": []interface{}{"${task1.results[0].id}", "static"},
		}
		got := resolveVariables(params, store)
		ids := got["ids"].([]interface{})
		if ids[0] != "abc" {
			t.Errorf("ids[0] = %q, want %q", ids[0], "abc")
		}
		if ids[1] != "static" {
			t.Errorf("ids[1] = %q, want %q", ids[1], "static")
		}
	})
}

func TestAvailableModuleNames(t *testing.T) {
	// Save and restore registry
	origRegistry := registry
	defer func() { registry = origRegistry }()

	registry = map[string]Module{}
	// Register a mock module
	registry["notion"] = nil
	registry["github"] = nil

	t.Run("nil returns all", func(t *testing.T) {
		got := availableModuleNames(nil)
		if len(got) != 2 {
			t.Errorf("expected 2, got %d", len(got))
		}
	})

	t.Run("filter to registered", func(t *testing.T) {
		got := availableModuleNames([]string{"notion", "dropbox"})
		if len(got) != 1 {
			t.Errorf("expected 1, got %d", len(got))
		}
		if got[0] != "notion" {
			t.Errorf("expected notion, got %s", got[0])
		}
	})

	t.Run("no matches", func(t *testing.T) {
		got := availableModuleNames([]string{"dropbox", "trello"})
		if len(got) != 0 {
			t.Errorf("expected 0, got %d", len(got))
		}
	})
}
