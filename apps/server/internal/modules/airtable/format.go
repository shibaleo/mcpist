package airtable

import (
	"encoding/json"
	"fmt"
	"strings"
)

// =============================================================================
// Compact formatters per tool — pure transformation: (toolName, JSON) → string
// =============================================================================

func formatCompact(toolName, jsonStr string) string {
	switch toolName {
	case "list_bases":
		return basesToCSV(jsonStr)
	case "get_base_tables":
		return tablesToCSV(jsonStr)
	case "get_table_views":
		return viewsToCSV(jsonStr)
	case "create_table", "update_table":
		return pickKeys(jsonStr, "id", "name")
	default:
		return jsonStr
	}
}

// pickKeys extracts only the specified keys from a JSON object.
func pickKeys(jsonStr string, keys ...string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	result := make(map[string]any, len(keys))
	for _, k := range keys {
		if v, ok := data[k]; ok && v != nil {
			result[k] = v
		}
	}
	out, err := json.Marshal(result)
	if err != nil {
		return jsonStr
	}
	return string(out)
}

// basesToCSV: id,name,permissionLevel
func basesToCSV(jsonStr string) string {
	var bases []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &bases); err != nil {
		return jsonStr
	}
	if len(bases) == 0 {
		return "# 0 bases"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,permissionLevel\n")
	for _, b := range bases {
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			csvEscape(str(b, "id")),
			csvEscape(str(b, "name")),
			str(b, "permissionLevel"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// tablesToCSV: id,name
func tablesToCSV(jsonStr string) string {
	var tables []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &tables); err != nil {
		return jsonStr
	}
	if len(tables) == 0 {
		return "# 0 tables"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name\n")
	for _, t := range tables {
		sb.WriteString(fmt.Sprintf("%s,%s\n",
			csvEscape(str(t, "id")),
			csvEscape(str(t, "name")),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// viewsToCSV: id,name,type
func viewsToCSV(jsonStr string) string {
	var views []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &views); err != nil {
		return jsonStr
	}
	if len(views) == 0 {
		return "# 0 views"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,type\n")
	for _, v := range views {
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			csvEscape(str(v, "id")),
			csvEscape(str(v, "name")),
			str(v, "type"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// =============================================================================
// Helpers
// =============================================================================

func str(obj map[string]any, key string) string {
	if v, ok := obj[key].(string); ok {
		return v
	}
	return ""
}

func csvEscape(s string) string {
	if s == "" {
		return ""
	}
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}
