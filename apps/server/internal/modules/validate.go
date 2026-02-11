package modules

import (
	"fmt"
	"strings"
)

// ValidateParams checks params against InputSchema.
// - Required fields: returns error if missing
// - Type check: verifies value matches declared property type
// - Type coercion: JSON numbers (float64) are kept as-is (handlers already expect float64)
// Returns validated params (shallow copy) or error.
func ValidateParams(schema InputSchema, params map[string]any) (map[string]any, error) {
	if params == nil {
		params = make(map[string]any)
	}

	// Check required fields
	var missing []string
	for _, key := range schema.Required {
		val, exists := params[key]
		if !exists || val == nil {
			missing = append(missing, key)
			continue
		}
		// Check for zero-value strings on required fields
		if s, ok := val.(string); ok && s == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required parameter(s): %s", strings.Join(missing, ", "))
	}

	// Type check provided params against schema properties
	for key, val := range params {
		prop, declared := schema.Properties[key]
		if !declared {
			// Extra params not in schema are passed through (lenient)
			continue
		}
		if val == nil {
			continue
		}
		if err := checkType(key, val, prop.Type); err != nil {
			return nil, err
		}
	}

	return params, nil
}

// checkType verifies that val matches the expected JSON Schema type.
func checkType(key string, val any, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := val.(string); !ok {
			return fmt.Errorf("parameter %q: expected string, got %T", key, val)
		}
	case "number", "integer":
		// JSON numbers arrive as float64
		if _, ok := val.(float64); !ok {
			return fmt.Errorf("parameter %q: expected number, got %T", key, val)
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("parameter %q: expected boolean, got %T", key, val)
		}
	case "array":
		if _, ok := val.([]interface{}); !ok {
			return fmt.Errorf("parameter %q: expected array, got %T", key, val)
		}
	case "object":
		if _, ok := val.(map[string]interface{}); !ok {
			return fmt.Errorf("parameter %q: expected object, got %T", key, val)
		}
	// "" or unknown types: skip check (lenient)
	}
	return nil
}

// findTool looks up a tool by name from a tool list.
func findTool(tools []Tool, name string) (Tool, bool) {
	for _, t := range tools {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}
