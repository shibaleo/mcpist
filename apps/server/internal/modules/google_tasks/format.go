package google_tasks

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
	case "list_task_lists":
		return taskListsCSV(jsonStr)
	case "list_tasks":
		return tasksCSV(jsonStr)
	case "create_task", "update_task", "complete_task":
		return pickKeys(jsonStr, "id", "title", "status")
	default:
		return jsonStr
	}
}

// taskListsCSV formats list_task_lists response → CSV: id, title, updated.
func taskListsCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	items, ok := data["items"].([]any)
	if !ok || len(items) == 0 {
		return "# 0 task lists"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,title,updated\n")
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			csvEscape(str(m, "id")),
			csvEscape(str(m, "title")),
			str(m, "updated"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// tasksCSV formats list_tasks response → CSV: id, title, status, due, parent.
func tasksCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	items, ok := data["items"].([]any)
	if !ok || len(items) == 0 {
		return "# 0 tasks"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,title,status,due,parent\n")
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
			csvEscape(str(m, "id")),
			csvEscape(str(m, "title")),
			str(m, "status"),
			str(m, "due"),
			str(m, "parent"),
		))
	}
	sb.WriteString("```")

	if token := str(data, "nextPageToken"); token != "" {
		sb.WriteString(fmt.Sprintf("\nnextPageToken=%s", token))
	}
	return sb.String()
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
