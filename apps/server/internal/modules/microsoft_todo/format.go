package microsoft_todo

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
	case "list_lists":
		return listsToCSV(jsonStr)
	case "list_tasks":
		return tasksToCSV(jsonStr)
	case "create_list", "update_list":
		return pickKeys(jsonStr, "id", "displayName")
	case "create_task", "update_task", "complete_task":
		return pickKeys(jsonStr, "id", "title", "status")
	case "delete_list", "delete_task":
		return pickKeys(jsonStr, "success", "message")
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

// listsToCSV: id,displayName
func listsToCSV(jsonStr string) string {
	var lists []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &lists); err != nil {
		return jsonStr
	}
	if len(lists) == 0 {
		return "# 0 lists"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,displayName\n")
	for _, l := range lists {
		sb.WriteString(fmt.Sprintf("%s,%s\n",
			csvEscape(str(l, "id")),
			csvEscape(str(l, "displayName")),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// tasksToCSV: id,title,status,importance,dueDate
func tasksToCSV(jsonStr string) string {
	var tasks []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &tasks); err != nil {
		return jsonStr
	}
	if len(tasks) == 0 {
		return "# 0 tasks"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,title,status,importance,dueDate\n")
	for _, t := range tasks {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
			csvEscape(str(t, "id")),
			csvEscape(str(t, "title")),
			str(t, "status"),
			str(t, "importance"),
			dateTimeField(t, "dueDateTime"),
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

// dateTimeField extracts the dateTime string from a nested DateTimeTimeZone object.
func dateTimeField(obj map[string]any, key string) string {
	if dt, ok := obj[key].(map[string]any); ok {
		return str(dt, "dateTime")
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
