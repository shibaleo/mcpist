package google_drive

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
	case "list_files", "search_files":
		return filesCSV(jsonStr)
	case "get_file":
		return pickKeys(jsonStr, "id", "name", "mimeType", "size", "webViewLink")
	case "create_folder", "copy_file":
		return pickKeys(jsonStr, "id", "name", "webViewLink")
	case "list_permissions":
		return permissionsCSV(jsonStr)
	case "list_comments":
		return commentsCSV(jsonStr)
	case "list_revisions":
		return revisionsCSV(jsonStr)
	case "list_shared_drives":
		return drivesCSV(jsonStr)
	default:
		return jsonStr
	}
}

// filesCSV formats file list response → CSV: id, name, mimeType, size, modifiedTime.
func filesCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	files, ok := data["files"].([]any)
	if !ok || len(files) == 0 {
		return "# 0 files"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,mimeType,size,modifiedTime\n")
	for _, f := range files {
		m, ok := f.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
			csvEscape(str(m, "id")),
			csvEscape(str(m, "name")),
			str(m, "mimeType"),
			str(m, "size"),
			str(m, "modifiedTime"),
		))
	}
	sb.WriteString("```")

	if token := str(data, "nextPageToken"); token != "" {
		sb.WriteString(fmt.Sprintf("\nnextPageToken=%s", token))
	}
	return sb.String()
}

// permissionsCSV formats permission list → CSV: id, type, role, emailAddress.
func permissionsCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	perms, ok := data["permissions"].([]any)
	if !ok || len(perms) == 0 {
		return "# 0 permissions"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,type,role,emailAddress\n")
	for _, p := range perms {
		m, ok := p.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			str(m, "id"),
			str(m, "type"),
			str(m, "role"),
			str(m, "emailAddress"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// commentsCSV formats comment list → CSV: id, content, author, createdTime.
func commentsCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	comments, ok := data["comments"].([]any)
	if !ok || len(comments) == 0 {
		return "# 0 comments"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,content,author,createdTime\n")
	for _, c := range comments {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		author := ""
		if a, ok := m["author"].(map[string]any); ok {
			author = str(a, "displayName")
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			str(m, "id"),
			csvEscape(str(m, "content")),
			csvEscape(author),
			str(m, "createdTime"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// revisionsCSV formats revision list → CSV: id, modifiedTime, size.
func revisionsCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	revisions, ok := data["revisions"].([]any)
	if !ok || len(revisions) == 0 {
		return "# 0 revisions"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,modifiedTime,size\n")
	for _, r := range revisions {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			str(m, "id"),
			str(m, "modifiedTime"),
			str(m, "size"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// drivesCSV formats shared drive list → CSV: id, name.
func drivesCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	drives, ok := data["drives"].([]any)
	if !ok || len(drives) == 0 {
		return "# 0 shared drives"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name\n")
	for _, d := range drives {
		m, ok := d.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s\n",
			str(m, "id"),
			csvEscape(str(m, "name")),
		))
	}
	sb.WriteString("```")
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
