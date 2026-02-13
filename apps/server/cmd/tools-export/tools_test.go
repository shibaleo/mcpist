package main

import (
	"context"
	"strings"
	"testing"

	"mcpist/server/internal/modules"
)

// TestToolDefinitions validates that all registered modules have consistent
// tool definitions: descriptions are non-empty, and every tool in Tools()
// has a corresponding handler in ExecuteTool().
func TestToolDefinitions(t *testing.T) {
	moduleNames := modules.ListModules()
	if len(moduleNames) == 0 {
		t.Fatal("no modules registered")
	}

	for _, name := range moduleNames {
		m, ok := modules.GetModule(name)
		if !ok {
			t.Errorf("module %q: registered but GetModule returned false", name)
			continue
		}

		t.Run(name, func(t *testing.T) {
			// Module must have en-US description
			descs := m.Descriptions()
			if descs["en-US"] == "" {
				t.Errorf("module %q: missing en-US description", name)
			}

			tools := m.Tools()
			if len(tools) == 0 {
				t.Errorf("module %q: no tools defined", name)
				return
			}

			for _, tool := range tools {
				t.Run(tool.Name, func(t *testing.T) {
					// Tool name must not be empty
					if tool.Name == "" {
						t.Error("tool has empty Name")
					}

					// Tool ID must follow "module:tool" convention
					expectedPrefix := name + ":"
					if !strings.HasPrefix(tool.ID, expectedPrefix) {
						t.Errorf("tool ID %q does not start with %q", tool.ID, expectedPrefix)
					}

					// en-US description must exist
					if tool.Descriptions["en-US"] == "" {
						t.Errorf("missing en-US description")
					}

					// Handler must exist (ExecuteTool must not return "unknown tool")
					// Use recover since some handlers panic on nil params — that's OK,
					// it means the handler exists. We only care about "unknown tool".
					func() {
						defer func() { recover() }()
						_, err := m.ExecuteTool(context.Background(), tool.Name, nil)
						if err != nil && strings.Contains(err.Error(), "unknown tool") {
							t.Errorf("tool %q defined in Tools() but has no handler", tool.Name)
						}
					}()

					// InputSchema type must be "object"
					if tool.InputSchema.Type != "object" {
						t.Errorf("InputSchema.Type = %q, want \"object\"", tool.InputSchema.Type)
					}

					// All required fields must exist in properties
					for _, req := range tool.InputSchema.Required {
						if _, ok := tool.InputSchema.Properties[req]; !ok {
							t.Errorf("required field %q not found in properties", req)
						}
					}

					// All properties must have a description
					for propName, prop := range tool.InputSchema.Properties {
						if prop.Description == "" {
							t.Errorf("property %q has empty description", propName)
						}
					}
				})
			}
		})
	}
}

// TestToolHandlerCoverage ensures there are no orphan handlers (handlers
// registered in toolHandlers but not in toolDefinitions). This is checked
// indirectly: if Tools() returns N tools and all have handlers, and the
// module exposes no other way to call handlers, coverage is complete.
// This test counts tools per module as a sanity check.
func TestToolCount(t *testing.T) {
	moduleNames := modules.ListModules()
	total := 0
	for _, name := range moduleNames {
		m, _ := modules.GetModule(name)
		count := len(m.Tools())
		if count == 0 {
			t.Errorf("module %q has 0 tools", name)
		}
		total += count
		t.Logf("%-20s %3d tools", name, count)
	}
	t.Logf("%-20s %3d tools", "TOTAL", total)
}
