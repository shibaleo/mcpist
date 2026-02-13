package confluence

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
	case "list_spaces":
		return spacesToCSV(jsonStr)
	case "get_pages":
		return pagesToCSV(jsonStr)
	case "search":
		return searchToCSV(jsonStr)
	case "get_page_labels":
		return labelsToCSV(jsonStr)
	case "get_page_comments":
		return commentsToCompact(jsonStr)
	// Read: single items → MD
	case "get_space":
		return spaceToCompact(jsonStr)
	case "get_page":
		return pageToCompact(jsonStr)
	// Write
	case "create_page", "update_page":
		return pickKeys(jsonStr, "id", "title", "status", "spaceId")
	case "delete_page":
		return pickKeys(jsonStr, "deleted")
	case "add_page_comment":
		return pickKeys(jsonStr, "id")
	case "add_page_label":
		return jsonStr // label response is already minimal
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

// spacesToCSV: id,key,name,type
func spacesToCSV(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	results, ok := wrapper["results"].([]any)
	if !ok {
		return jsonStr
	}
	if len(results) == 0 {
		return "# 0 spaces"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,key,name,type\n")
	for _, raw := range results {
		s, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			csvEscape(str(s, "id")),
			csvEscape(str(s, "key")),
			csvEscape(str(s, "name")),
			str(s, "type"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// spaceToCompact: single space detail
func spaceToCompact(jsonStr string) string {
	var s map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &s); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s (%s)\n", str(s, "name"), str(s, "key")))
	sb.WriteString(fmt.Sprintf("- **ID**: %s\n", str(s, "id")))
	if t := str(s, "type"); t != "" {
		sb.WriteString(fmt.Sprintf("- **Type**: %s\n", t))
	}
	if st := str(s, "status"); st != "" {
		sb.WriteString(fmt.Sprintf("- **Status**: %s\n", st))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// pagesToCSV: id,title,status,spaceId
func pagesToCSV(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	results, ok := wrapper["results"].([]any)
	if !ok {
		return jsonStr
	}
	if len(results) == 0 {
		return "# 0 pages"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,title,status,spaceId\n")
	for _, raw := range results {
		p, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			csvEscape(str(p, "id")),
			csvEscape(str(p, "title")),
			str(p, "status"),
			str(p, "spaceId"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// pageToCompact: single page detail with body
func pageToCompact(jsonStr string) string {
	var p map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(p, "title")))
	sb.WriteString(fmt.Sprintf("- **ID**: %s\n", str(p, "id")))
	if st := str(p, "status"); st != "" {
		sb.WriteString(fmt.Sprintf("- **Status**: %s\n", st))
	}
	if spaceID := str(p, "spaceId"); spaceID != "" {
		sb.WriteString(fmt.Sprintf("- **Space**: %s\n", spaceID))
	}
	// Version
	if ver, ok := p["version"].(map[string]any); ok {
		if num, ok := ver["number"].(float64); ok {
			sb.WriteString(fmt.Sprintf("- **Version**: %d\n", int(num)))
		}
	}
	// Body content
	if body, ok := p["body"].(map[string]any); ok {
		if storage, ok := body["storage"].(map[string]any); ok {
			if val := str(storage, "value"); val != "" {
				if len(val) > 3000 {
					val = val[:3000] + "...(truncated)"
				}
				sb.WriteString(fmt.Sprintf("\n## Content\n%s\n", val))
			}
		}
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// searchToCSV: id,title,type,spaceId
func searchToCSV(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	results, ok := wrapper["results"].([]any)
	if !ok {
		return jsonStr
	}
	if len(results) == 0 {
		return "# 0 results"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("```csv  # %d results\nid,title,type,spaceId\n", len(results)))
	for _, raw := range results {
		r, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		// Search results have content.id, content.title etc.
		content, _ := r["content"].(map[string]any)
		if content == nil {
			content = r
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			csvEscape(str(content, "id")),
			csvEscape(str(content, "title")),
			str(content, "type"),
			str(content, "spaceId"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// labelsToCSV: id,name,prefix
func labelsToCSV(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	results, ok := wrapper["results"].([]any)
	if !ok {
		return jsonStr
	}
	if len(results) == 0 {
		return "# 0 labels"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,prefix\n")
	for _, raw := range results {
		l, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			csvEscape(str(l, "id")),
			csvEscape(str(l, "name")),
			str(l, "prefix"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// commentsToCompact: comments list in MD
func commentsToCompact(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	results, ok := wrapper["results"].([]any)
	if !ok {
		return jsonStr
	}
	if len(results) == 0 {
		return "# 0 comments"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %d comments\n\n", len(results)))
	for _, raw := range results {
		c, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := str(c, "id")
		created := str(c, "createdAt")
		if len(created) > 16 {
			created = created[:16]
		}
		// Body
		bodyText := ""
		if body, ok := c["body"].(map[string]any); ok {
			if storage, ok := body["storage"].(map[string]any); ok {
				bodyText = str(storage, "value")
			}
		}
		sb.WriteString(fmt.Sprintf("**#%s** (%s):\n%s\n\n", id, created, bodyText))
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

func csvEscape(s string) string {
	if s == "" {
		return ""
	}
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}
