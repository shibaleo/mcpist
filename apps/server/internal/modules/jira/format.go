package jira

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
	// Read: list/search → CSV
	case "list_projects":
		return projectsToCSV(jsonStr)
	case "search":
		return issuesToCSV(jsonStr)
	case "get_transitions":
		return transitionsToCSV(jsonStr)
	case "get_comments":
		return commentsToCompact(jsonStr)
	// Read: single item → MD
	case "get_myself":
		return myselfToCompact(jsonStr)
	case "get_project":
		return projectToCompact(jsonStr)
	case "get_issue":
		return issueToCompact(jsonStr)
	// Write: confirmation with key fields
	case "create_issue":
		return pickKeys(jsonStr, "id", "key", "self")
	case "update_issue":
		return pickKeys(jsonStr, "updated", "issue_key")
	case "transition_issue":
		return pickKeys(jsonStr, "transitioned", "issue_key", "transition_id")
	case "add_comment":
		return pickKeys(jsonStr, "id", "self")
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

// myselfToCompact: user profile summary
func myselfToCompact(jsonStr string) string {
	var u map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &u); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(u, "displayName")))
	sb.WriteString(fmt.Sprintf("- **Account ID**: %s\n", str(u, "accountId")))
	if email := str(u, "emailAddress"); email != "" {
		sb.WriteString(fmt.Sprintf("- **Email**: %s\n", email))
	}
	if tz := str(u, "timeZone"); tz != "" {
		sb.WriteString(fmt.Sprintf("- **Timezone**: %s\n", tz))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// projectsToCSV: id,key,name,projectTypeKey
func projectsToCSV(jsonStr string) string {
	// Jira SearchProjects returns {values:[...]}
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	projects, ok := wrapper["values"].([]any)
	if !ok {
		return jsonStr
	}
	if len(projects) == 0 {
		return "# 0 projects"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,key,name,projectTypeKey\n")
	for _, raw := range projects {
		p, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			csvEscape(str(p, "id")),
			csvEscape(str(p, "key")),
			csvEscape(str(p, "name")),
			str(p, "projectTypeKey"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// projectToCompact: single project detail
func projectToCompact(jsonStr string) string {
	var p map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s (%s)\n", str(p, "name"), str(p, "key")))
	sb.WriteString(fmt.Sprintf("- **ID**: %s\n", str(p, "id")))
	if pt := str(p, "projectTypeKey"); pt != "" {
		sb.WriteString(fmt.Sprintf("- **Type**: %s\n", pt))
	}
	if desc := str(p, "description"); desc != "" {
		sb.WriteString(fmt.Sprintf("- **Description**: %s\n", desc))
	}
	if lead, ok := p["lead"].(map[string]any); ok {
		sb.WriteString(fmt.Sprintf("- **Lead**: %s\n", str(lead, "displayName")))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// issuesToCSV: key,summary,status,priority,assignee,updated
func issuesToCSV(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	issues, ok := wrapper["issues"].([]any)
	if !ok {
		return jsonStr
	}
	total := intVal(wrapper, "total")
	if len(issues) == 0 {
		return fmt.Sprintf("# 0/%d issues", total)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("```csv  # %d/%d issues\nkey,summary,status,priority,assignee,updated\n", len(issues), total))
	for _, raw := range issues {
		issue, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		fields, _ := issue["fields"].(map[string]any)
		if fields == nil {
			fields = map[string]any{}
		}
		key := str(issue, "key")
		summary := str(fields, "summary")
		status := nestedName(fields, "status")
		priority := nestedName(fields, "priority")
		assignee := nestedName(fields, "assignee", "displayName")
		updated := str(fields, "updated")
		if len(updated) > 10 {
			updated = updated[:10]
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			key,
			csvEscape(summary),
			csvEscape(status),
			csvEscape(priority),
			csvEscape(assignee),
			updated,
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// issueToCompact: single issue detail
func issueToCompact(jsonStr string) string {
	var issue map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &issue); err != nil {
		return jsonStr
	}
	fields, _ := issue["fields"].(map[string]any)
	if fields == nil {
		fields = map[string]any{}
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s: %s\n", str(issue, "key"), str(fields, "summary")))
	sb.WriteString(fmt.Sprintf("- **Status**: %s\n", nestedName(fields, "status")))
	if p := nestedName(fields, "priority"); p != "" {
		sb.WriteString(fmt.Sprintf("- **Priority**: %s\n", p))
	}
	if a := nestedName(fields, "assignee", "displayName"); a != "" {
		sb.WriteString(fmt.Sprintf("- **Assignee**: %s\n", a))
	}
	if r := nestedName(fields, "reporter", "displayName"); r != "" {
		sb.WriteString(fmt.Sprintf("- **Reporter**: %s\n", r))
	}
	if it := nestedName(fields, "issuetype"); it != "" {
		sb.WriteString(fmt.Sprintf("- **Type**: %s\n", it))
	}
	if labels, ok := fields["labels"].([]any); ok && len(labels) > 0 {
		strs := make([]string, 0, len(labels))
		for _, l := range labels {
			if s, ok := l.(string); ok {
				strs = append(strs, s)
			}
		}
		if len(strs) > 0 {
			sb.WriteString(fmt.Sprintf("- **Labels**: %s\n", strings.Join(strs, ", ")))
		}
	}
	if created := str(fields, "created"); len(created) >= 10 {
		sb.WriteString(fmt.Sprintf("- **Created**: %s\n", created[:10]))
	}
	if updated := str(fields, "updated"); len(updated) >= 10 {
		sb.WriteString(fmt.Sprintf("- **Updated**: %s\n", updated[:10]))
	}
	// Description: extract text from ADF
	if desc := extractADFText(fields["description"]); desc != "" {
		sb.WriteString(fmt.Sprintf("\n## Description\n%s\n", desc))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// transitionsToCSV: id,name,to_status
func transitionsToCSV(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	transitions, ok := wrapper["transitions"].([]any)
	if !ok {
		return jsonStr
	}
	if len(transitions) == 0 {
		return "# 0 transitions"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,to_status\n")
	for _, raw := range transitions {
		t, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		toStatus := ""
		if to, ok := t["to"].(map[string]any); ok {
			toStatus = str(to, "name")
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			str(t, "id"),
			csvEscape(str(t, "name")),
			csvEscape(toStatus),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// commentsToCompact: comments list
func commentsToCompact(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	comments, ok := wrapper["comments"].([]any)
	if !ok {
		return jsonStr
	}
	if len(comments) == 0 {
		return "# 0 comments"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %d comments\n\n", len(comments)))
	for _, raw := range comments {
		c, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		author := ""
		if a, ok := c["author"].(map[string]any); ok {
			author = str(a, "displayName")
		}
		created := str(c, "created")
		if len(created) > 16 {
			created = created[:16]
		}
		body := extractADFText(c["body"])
		sb.WriteString(fmt.Sprintf("**%s** (%s):\n%s\n\n", author, created, body))
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

// nestedName extracts a nested object's "name" field (or custom field).
func nestedName(obj map[string]any, key string, field ...string) string {
	f := "name"
	if len(field) > 0 {
		f = field[0]
	}
	if nested, ok := obj[key].(map[string]any); ok {
		return str(nested, f)
	}
	return ""
}

// extractADFText extracts plain text from an ADF document.
func extractADFText(v any) string {
	doc, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	content, _ := doc["content"].([]any)
	var parts []string
	for _, block := range content {
		bm, _ := block.(map[string]any)
		innerContent, _ := bm["content"].([]any)
		for _, inline := range innerContent {
			im, _ := inline.(map[string]any)
			if text := str(im, "text"); text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
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
