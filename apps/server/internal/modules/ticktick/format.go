package ticktick

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
	case "list_projects":
		return projectsToCSV(jsonStr)
	case "get_project":
		return projectToCompact(jsonStr)
	case "get_project_data":
		return projectDataToCompact(jsonStr)
	case "get_task":
		return taskToCompact(jsonStr)
	case "create_project", "update_project":
		return pickKeys(jsonStr, "id", "name", "kind", "viewMode")
	case "create_task", "update_task":
		return pickKeys(jsonStr, "id", "title", "projectId", "priority", "dueDate", "status")
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

// projectsToCSV: id,name,kind,viewMode,closed
func projectsToCSV(jsonStr string) string {
	var projects []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &projects); err != nil {
		return jsonStr
	}
	if len(projects) == 0 {
		return "# 0 projects"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,kind,viewMode,closed\n")
	for _, p := range projects {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%v\n",
			csvEscape(str(p, "id")),
			csvEscape(str(p, "name")),
			str(p, "kind"),
			str(p, "viewMode"),
			boolVal(p, "closed"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// projectToCompact: single project summary
func projectToCompact(jsonStr string) string {
	var p map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(p, "name")))
	sb.WriteString(fmt.Sprintf("- **ID**: %s\n", str(p, "id")))
	if kind := str(p, "kind"); kind != "" {
		sb.WriteString(fmt.Sprintf("- **Kind**: %s\n", kind))
	}
	if vm := str(p, "viewMode"); vm != "" {
		sb.WriteString(fmt.Sprintf("- **View**: %s\n", vm))
	}
	if color := str(p, "color"); color != "" {
		sb.WriteString(fmt.Sprintf("- **Color**: %s\n", color))
	}
	if boolVal(p, "closed") {
		sb.WriteString("- **Closed**: Yes\n")
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// projectDataToCompact: project + task count + column count
func projectDataToCompact(jsonStr string) string {
	var pd map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &pd); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	// Project info
	if proj, ok := pd["project"].(map[string]any); ok {
		sb.WriteString(fmt.Sprintf("# %s\n", str(proj, "name")))
		sb.WriteString(fmt.Sprintf("- **ID**: %s\n", str(proj, "id")))
		if kind := str(proj, "kind"); kind != "" {
			sb.WriteString(fmt.Sprintf("- **Kind**: %s\n", kind))
		}
	}
	// Tasks
	if tasks, ok := pd["tasks"].([]any); ok {
		sb.WriteString(fmt.Sprintf("\n## Tasks (%d)\n", len(tasks)))
		sb.WriteString("```csv\nid,title,priority,status,dueDate\n")
		for _, raw := range tasks {
			t, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			sb.WriteString(fmt.Sprintf("%s,%s,%v,%v,%s\n",
				csvEscape(str(t, "id")),
				csvEscape(str(t, "title")),
				intVal(t, "priority"),
				intVal(t, "status"),
				str(t, "dueDate"),
			))
		}
		sb.WriteString("```")
	}
	// Columns
	if cols, ok := pd["columns"].([]any); ok && len(cols) > 0 {
		sb.WriteString(fmt.Sprintf("\n\n## Columns (%d)\n", len(cols)))
		sb.WriteString("```csv\nid,name\n")
		for _, raw := range cols {
			col, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			sb.WriteString(fmt.Sprintf("%s,%s\n",
				csvEscape(str(col, "id")),
				csvEscape(str(col, "name")),
			))
		}
		sb.WriteString("```")
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// taskToCompact: single task detail
func taskToCompact(jsonStr string) string {
	var t map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &t); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(t, "title")))
	sb.WriteString(fmt.Sprintf("- **ID**: %s\n", str(t, "id")))
	sb.WriteString(fmt.Sprintf("- **Project**: %s\n", str(t, "projectId")))
	if p := intVal(t, "priority"); p > 0 {
		sb.WriteString(fmt.Sprintf("- **Priority**: %d\n", p))
	}
	if s := intVal(t, "status"); s > 0 {
		sb.WriteString(fmt.Sprintf("- **Status**: %d\n", s))
	}
	if due := str(t, "dueDate"); due != "" {
		sb.WriteString(fmt.Sprintf("- **Due**: %s\n", due))
	}
	if start := str(t, "startDate"); start != "" {
		sb.WriteString(fmt.Sprintf("- **Start**: %s\n", start))
	}
	if tags := tagsStr(t); tags != "" {
		sb.WriteString(fmt.Sprintf("- **Tags**: %s\n", tags))
	}
	if content := str(t, "content"); content != "" {
		sb.WriteString(fmt.Sprintf("\n## Content\n%s\n", content))
	}
	if desc := str(t, "desc"); desc != "" {
		sb.WriteString(fmt.Sprintf("\n## Description\n%s\n", desc))
	}
	// Subtasks
	if items, ok := t["items"].([]any); ok && len(items) > 0 {
		sb.WriteString(fmt.Sprintf("\n## Subtasks (%d)\n", len(items)))
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			check := "[ ]"
			if intVal(item, "status") > 0 {
				check = "[x]"
			}
			sb.WriteString(fmt.Sprintf("- %s %s\n", check, str(item, "title")))
		}
	}
	return strings.TrimSuffix(sb.String(), "\n")
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

func intVal(obj map[string]any, key string) int {
	if v, ok := obj[key].(float64); ok {
		return int(v)
	}
	return 0
}

func boolVal(obj map[string]any, key string) bool {
	if v, ok := obj[key].(bool); ok {
		return v
	}
	return false
}

// tagsStr joins tags into a semicolon-separated string.
func tagsStr(task map[string]any) string {
	tags, ok := task["tags"].([]any)
	if !ok || len(tags) == 0 {
		return ""
	}
	names := make([]string, 0, len(tags))
	for _, t := range tags {
		if s, ok := t.(string); ok {
			names = append(names, s)
		}
	}
	return strings.Join(names, ";")
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
