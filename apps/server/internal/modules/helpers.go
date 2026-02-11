package modules

import (
	"encoding/json"
	"fmt"
)

// ToJSON marshals any value to a JSON string.
// Used by ogen module handlers to serialize API responses.
func ToJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(b), nil
}

// ToStringSlice converts []interface{} (from MCP params) to []string.
// Non-string elements are silently skipped.
func ToStringSlice(v []interface{}) []string {
	out := make([]string, 0, len(v))
	for _, item := range v {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
