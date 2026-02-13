package asana

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
	// Read: lists → CSV
	case "list_workspaces":
		return workspacesToCSV(jsonStr)
	case "list_projects":
		return projectsToCSV(jsonStr)
	case "list_sections":
		return sectionsToCSV(jsonStr)
	case "list_tasks":
		return tasksToCSV(jsonStr)
	case "list_subtasks":
		return subtasksToCSV(jsonStr)
	case "list_stories":
		return storiesToCompact(jsonStr)
	case "list_tags":
		return tagsToCSV(jsonStr)
	case "search_tasks":
		return tasksToCSV(jsonStr)
	// Read: single item → MD
	case "get_me":
		return userToCompact(jsonStr)
	case "get_workspace":
		return workspaceToCompact(jsonStr)
	case "get_project":
		return projectToCompact(jsonStr)
	case "get_task":
		return taskToCompact(jsonStr)
	// Write
	case "create_project", "update_project":
		return pickKeys(jsonStr, "gid", "name")
	case "delete_project":
		return pickKeys(jsonStr, "success", "message")
	case "create_section":
		return pickKeys(jsonStr, "gid", "name")
	case "create_task", "update_task", "complete_task":
		return pickKeys(jsonStr, "gid", "name", "completed")
	case "delete_task":
		return pickKeys(jsonStr, "success", "message")
	case "create_subtask":
		return pickKeys(jsonStr, "gid", "name")
	case "add_comment":
		return pickKeys(jsonStr, "gid", "text")
	case "create_tag":
		return pickKeys(jsonStr, "gid", "name")
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

// userToCompact: user profile
func userToCompact(jsonStr string) string {
	var u map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &u); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(u, "name")))
	sb.WriteString(fmt.Sprintf("- **GID**: %s\n", str(u, "gid")))
	if email := str(u, "email"); email != "" {
		sb.WriteString(fmt.Sprintf("- **Email**: %s\n", email))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// workspacesToCSV: gid,name
func workspacesToCSV(jsonStr string) string {
	var workspaces []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &workspaces); err != nil {
		return jsonStr
	}
	if len(workspaces) == 0 {
		return "# 0 workspaces"
	}
	var sb strings.Builder
	sb.WriteString("```csv\ngid,name\n")
	for _, w := range workspaces {
		sb.WriteString(fmt.Sprintf("%s,%s\n",
			csvEscape(str(w, "gid")),
			csvEscape(str(w, "name")),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// workspaceToCompact: single workspace
func workspaceToCompact(jsonStr string) string {
	var w map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &w); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(w, "name")))
	sb.WriteString(fmt.Sprintf("- **GID**: %s\n", str(w, "gid")))
	if org, ok := w["is_organization"].(bool); ok {
		sb.WriteString(fmt.Sprintf("- **Organization**: %v\n", org))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// projectsToCSV: gid,name,archived,color
func projectsToCSV(jsonStr string) string {
	var projects []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &projects); err != nil {
		return jsonStr
	}
	if len(projects) == 0 {
		return "# 0 projects"
	}
	var sb strings.Builder
	sb.WriteString("```csv\ngid,name,archived,color\n")
	for _, p := range projects {
		sb.WriteString(fmt.Sprintf("%s,%s,%v,%s\n",
			csvEscape(str(p, "gid")),
			csvEscape(str(p, "name")),
			boolVal(p, "archived"),
			str(p, "color"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// projectToCompact: single project
func projectToCompact(jsonStr string) string {
	var p map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(p, "name")))
	sb.WriteString(fmt.Sprintf("- **GID**: %s\n", str(p, "gid")))
	if color := str(p, "color"); color != "" {
		sb.WriteString(fmt.Sprintf("- **Color**: %s\n", color))
	}
	if view := str(p, "default_view"); view != "" {
		sb.WriteString(fmt.Sprintf("- **View**: %s\n", view))
	}
	if boolVal(p, "archived") {
		sb.WriteString("- **Archived**: true\n")
	}
	if dueOn := str(p, "due_on"); dueOn != "" {
		sb.WriteString(fmt.Sprintf("- **Due**: %s\n", dueOn))
	}
	if notes := str(p, "notes"); notes != "" {
		if len(notes) > 2000 {
			notes = notes[:2000] + "...(truncated)"
		}
		sb.WriteString(fmt.Sprintf("\n## Notes\n%s\n", notes))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// sectionsToCSV: gid,name
func sectionsToCSV(jsonStr string) string {
	var sections []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &sections); err != nil {
		return jsonStr
	}
	if len(sections) == 0 {
		return "# 0 sections"
	}
	var sb strings.Builder
	sb.WriteString("```csv\ngid,name\n")
	for _, s := range sections {
		sb.WriteString(fmt.Sprintf("%s,%s\n",
			csvEscape(str(s, "gid")),
			csvEscape(str(s, "name")),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// tasksToCSV: gid,name,completed,due_on,assignee
func tasksToCSV(jsonStr string) string {
	var tasks []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &tasks); err != nil {
		return jsonStr
	}
	if len(tasks) == 0 {
		return "# 0 tasks"
	}
	var sb strings.Builder
	sb.WriteString("```csv\ngid,name,completed,due_on,assignee\n")
	for _, t := range tasks {
		assignee := ""
		if a, ok := t["assignee"].(map[string]any); ok {
			assignee = str(a, "name")
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%v,%s,%s\n",
			csvEscape(str(t, "gid")),
			csvEscape(str(t, "name")),
			boolVal(t, "completed"),
			str(t, "due_on"),
			csvEscape(assignee),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// subtasksToCSV: gid,name,completed,due_on,assignee
func subtasksToCSV(jsonStr string) string {
	return tasksToCSV(jsonStr)
}

// taskToCompact: single task detail
func taskToCompact(jsonStr string) string {
	var t map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &t); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(t, "name")))
	sb.WriteString(fmt.Sprintf("- **GID**: %s\n", str(t, "gid")))
	if boolVal(t, "completed") {
		sb.WriteString("- **Completed**: true\n")
	}
	if a, ok := t["assignee"].(map[string]any); ok {
		sb.WriteString(fmt.Sprintf("- **Assignee**: %s\n", str(a, "name")))
	}
	if dueOn := str(t, "due_on"); dueOn != "" {
		sb.WriteString(fmt.Sprintf("- **Due**: %s\n", dueOn))
	}
	if dueAt := str(t, "due_at"); dueAt != "" {
		sb.WriteString(fmt.Sprintf("- **Due At**: %s\n", dueAt))
	}
	if startOn := str(t, "start_on"); startOn != "" {
		sb.WriteString(fmt.Sprintf("- **Start**: %s\n", startOn))
	}
	// Projects
	if projects, ok := t["memberships"].([]any); ok && len(projects) > 0 {
		names := make([]string, 0)
		for _, raw := range projects {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if proj, ok := m["project"].(map[string]any); ok {
				if name := str(proj, "name"); name != "" {
					names = append(names, name)
				}
			}
		}
		if len(names) > 0 {
			sb.WriteString(fmt.Sprintf("- **Projects**: %s\n", strings.Join(names, ", ")))
		}
	}
	// Tags
	if tags, ok := t["tags"].([]any); ok && len(tags) > 0 {
		names := make([]string, 0)
		for _, raw := range tags {
			tag, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if name := str(tag, "name"); name != "" {
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			sb.WriteString(fmt.Sprintf("- **Tags**: %s\n", strings.Join(names, ", ")))
		}
	}
	if notes := str(t, "notes"); notes != "" {
		if len(notes) > 2000 {
			notes = notes[:2000] + "...(truncated)"
		}
		sb.WriteString(fmt.Sprintf("\n## Notes\n%s\n", notes))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// storiesToCompact: stories in MD
func storiesToCompact(jsonStr string) string {
	var stories []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &stories); err != nil {
		return jsonStr
	}
	if len(stories) == 0 {
		return "# 0 stories"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %d stories\n\n", len(stories)))
	for _, s := range stories {
		sType := str(s, "type")
		createdBy := ""
		if cb, ok := s["created_by"].(map[string]any); ok {
			createdBy = str(cb, "name")
		}
		created := str(s, "created_at")
		if len(created) > 16 {
			created = created[:16]
		}
		text := str(s, "text")
		if text == "" {
			continue
		}
		if len(text) > 500 {
			text = text[:500] + "..."
		}
		sb.WriteString(fmt.Sprintf("**%s** [%s] (%s):\n%s\n\n", createdBy, sType, created, text))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// tagsToCSV: gid,name,color
func tagsToCSV(jsonStr string) string {
	var tags []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &tags); err != nil {
		return jsonStr
	}
	if len(tags) == 0 {
		return "# 0 tags"
	}
	var sb strings.Builder
	sb.WriteString("```csv\ngid,name,color\n")
	for _, t := range tags {
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			csvEscape(str(t, "gid")),
			csvEscape(str(t, "name")),
			str(t, "color"),
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

func boolVal(obj map[string]any, key string) bool {
	if v, ok := obj[key].(bool); ok {
		return v
	}
	return false
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
