package google_docs

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
	case "list_comments":
		return commentsCSV(jsonStr)
	case "create_document":
		return pickKeys(jsonStr, "documentId", "title")
	case "append_text", "insert_text", "delete_range",
		"apply_text_style", "apply_paragraph_style",
		"insert_table", "insert_page_break", "insert_image":
		return pickKeys(jsonStr, "documentId")
	case "add_comment":
		return pickKeys(jsonStr, "id", "content")
	case "delete_comment", "resolve_comment":
		return jsonStr
	default:
		return jsonStr
	}
}

// commentsCSV formats list_comments → CSV: id, content, author, resolved, createdTime.
func commentsCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	items, ok := data["comments"].([]any)
	if !ok || len(items) == 0 {
		return "# 0 comments"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,content,author,resolved,createdTime\n")
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		author := ""
		if a, ok := m["author"].(map[string]any); ok {
			author = str(a, "displayName")
		}
		resolved := ""
		if r, ok := m["resolved"].(bool); ok && r {
			resolved = "true"
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
			str(m, "id"),
			csvEscape(str(m, "content")),
			csvEscape(author),
			resolved,
			str(m, "createdTime"),
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
