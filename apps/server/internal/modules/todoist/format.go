package todoist

import (
	"encoding/json"
	"fmt"
	"strings"
)

// =============================================================================
// Compact formatters per tool
// =============================================================================

func formatCompact(toolName, jsonStr string) string {
	switch toolName {
	case "list_projects":
		return projectsToCSV(jsonStr)
	case "get_project":
		return projectToCompact(jsonStr)
	case "list_tasks":
		return tasksToCSV(jsonStr)
	case "get_task":
		return taskToCompact(jsonStr)
	case "create_task", "update_task":
		return pickKeys(jsonStr, "id", "content", "projectId", "due", "priority", "labels")
	case "list_sections":
		return sectionsToCSV(jsonStr)
	case "list_labels":
		return labelsToCSV(jsonStr)
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

// projectsToCSV: id,name,isFavorite,inboxProject,viewStyle
func projectsToCSV(jsonStr string) string {
	var projects []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &projects); err != nil {
		return jsonStr
	}
	if len(projects) == 0 {
		return "# 0 projects"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,isFavorite,inboxProject,viewStyle\n")
	for _, p := range projects {
		sb.WriteString(fmt.Sprintf("%s,%s,%v,%v,%s\n",
			csvEscape(str(p, "id")),
			csvEscape(str(p, "name")),
			boolVal(p, "isFavorite"),
			boolVal(p, "inboxProject"),
			str(p, "viewStyle"),
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
	if url := str(p, "url"); url != "" {
		sb.WriteString(fmt.Sprintf("- **URL**: %s\n", url))
	}
	sb.WriteString(fmt.Sprintf("- **View**: %s\n", str(p, "viewStyle")))
	if fav, ok := p["isFavorite"].(bool); ok && fav {
		sb.WriteString("- **Favorite**: Yes\n")
	}
	if inbox, ok := p["inboxProject"].(bool); ok && inbox {
		sb.WriteString("- **Inbox**: Yes\n")
	}
	if shared, ok := p["isShared"].(bool); ok && shared {
		sb.WriteString("- **Shared**: Yes\n")
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// tasksToCSV: id,content,projectId,priority,due,labels
func tasksToCSV(jsonStr string) string {
	var tasks []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &tasks); err != nil {
		return jsonStr
	}
	if len(tasks) == 0 {
		return "# 0 tasks"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,content,projectId,priority,due,labels\n")
	for _, t := range tasks {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%v,%s,%s\n",
			csvEscape(str(t, "id")),
			csvEscape(str(t, "content")),
			str(t, "projectId"),
			intVal(t, "priority"),
			csvEscape(dueStr(t)),
			csvEscape(labelsStr(t)),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// taskToCompact: single task detail
func taskToCompact(jsonStr string) string {
	var t map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &t); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(t, "content")))
	sb.WriteString(fmt.Sprintf("- **ID**: %s\n", str(t, "id")))
	sb.WriteString(fmt.Sprintf("- **Project**: %s\n", str(t, "projectId")))
	if p := intVal(t, "priority"); p > 1 {
		sb.WriteString(fmt.Sprintf("- **Priority**: %d\n", p))
	}
	if due := dueStr(t); due != "" {
		sb.WriteString(fmt.Sprintf("- **Due**: %s\n", due))
	}
	if labels := labelsStr(t); labels != "" {
		sb.WriteString(fmt.Sprintf("- **Labels**: %s\n", labels))
	}
	if desc := str(t, "description"); desc != "" {
		sb.WriteString(fmt.Sprintf("\n## Description\n%s\n", desc))
	}
	if url := str(t, "url"); url != "" {
		sb.WriteString(fmt.Sprintf("- **URL**: %s\n", url))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// sectionsToCSV: id,name,projectId,sectionOrder
func sectionsToCSV(jsonStr string) string {
	var sections []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &sections); err != nil {
		return jsonStr
	}
	if len(sections) == 0 {
		return "# 0 sections"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,projectId,sectionOrder\n")
	for _, s := range sections {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%v\n",
			csvEscape(str(s, "id")),
			csvEscape(str(s, "name")),
			str(s, "projectId"),
			intVal(s, "sectionOrder"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// labelsToCSV: id,name,color,isFavorite
func labelsToCSV(jsonStr string) string {
	var labels []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &labels); err != nil {
		return jsonStr
	}
	if len(labels) == 0 {
		return "# 0 labels"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,color,isFavorite\n")
	for _, l := range labels {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%v\n",
			csvEscape(str(l, "id")),
			csvEscape(str(l, "name")),
			str(l, "color"),
			boolVal(l, "isFavorite"),
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

// dueStr extracts a human-readable due string from a task's "due" field.
func dueStr(task map[string]any) string {
	due, ok := task["due"].(map[string]any)
	if !ok || due == nil {
		return ""
	}
	// Prefer datetime, then date, then string
	if dt := str(due, "datetime"); dt != "" {
		return dt
	}
	if d := str(due, "date"); d != "" {
		return d
	}
	return str(due, "string")
}

// labelsStr joins labels into a semicolon-separated string.
func labelsStr(task map[string]any) string {
	labels, ok := task["labels"].([]any)
	if !ok || len(labels) == 0 {
		return ""
	}
	names := make([]string, 0, len(labels))
	for _, l := range labels {
		if s, ok := l.(string); ok {
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
