package dropbox

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
	case "list_folder", "list_folder_continue":
		return entriesCSV(jsonStr)
	case "search_files":
		return searchCSV(jsonStr)
	case "list_shared_links":
		return linksCSV(jsonStr)
	case "list_revisions":
		return revisionsCSV(jsonStr)
	case "get_current_account":
		return accountCompact(jsonStr)
	case "create_shared_link":
		return pickKeys(jsonStr, "url", "name", "path_lower")
	default:
		return jsonStr
	}
}

// entriesCSV formats list_folder / list_folder_continue responses.
// Extracts entries[] array → CSV with name, .tag, size, client_modified.
func entriesCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	entries, ok := data["entries"].([]any)
	if !ok {
		return jsonStr
	}
	if len(entries) == 0 {
		hasMore, _ := data["has_more"].(bool)
		if hasMore {
			cursor := str(data, "cursor")
			return fmt.Sprintf("# 0 entries (has_more=true, cursor=%s)", cursor)
		}
		return "# 0 entries"
	}

	var sb strings.Builder
	sb.WriteString("```csv\nname,.tag,size,client_modified\n")
	for _, e := range entries {
		em, ok := e.(map[string]any)
		if !ok {
			continue
		}
		size := ""
		if s, ok := em["size"].(float64); ok {
			size = fmt.Sprintf("%.0f", s)
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			csvEscape(str(em, "name")),
			str(em, ".tag"),
			size,
			str(em, "client_modified"),
		))
	}
	sb.WriteString("```")

	// Append pagination info
	hasMore, _ := data["has_more"].(bool)
	if hasMore {
		cursor := str(data, "cursor")
		sb.WriteString(fmt.Sprintf("\nhas_more=true cursor=%s", cursor))
	}

	return sb.String()
}

// searchCSV formats search_v2 responses.
// Extracts matches[].metadata.metadata → CSV with name, path_display, .tag.
func searchCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	matches, ok := data["matches"].([]any)
	if !ok {
		return jsonStr
	}
	if len(matches) == 0 {
		return "# 0 matches"
	}

	var sb strings.Builder
	sb.WriteString("```csv\nname,path_display,.tag\n")
	for _, m := range matches {
		mm, ok := m.(map[string]any)
		if !ok {
			continue
		}
		// matches[].metadata.metadata contains the actual file/folder metadata
		metaWrap, _ := mm["metadata"].(map[string]any)
		if metaWrap == nil {
			continue
		}
		meta, _ := metaWrap["metadata"].(map[string]any)
		if meta == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			csvEscape(str(meta, "name")),
			csvEscape(str(meta, "path_display")),
			str(meta, ".tag"),
		))
	}
	sb.WriteString("```")

	hasMore, _ := data["has_more"].(bool)
	if hasMore {
		cursor := str(data, "cursor")
		sb.WriteString(fmt.Sprintf("\nhas_more=true cursor=%s", cursor))
	}

	return sb.String()
}

// linksCSV formats list_shared_links responses.
func linksCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	links, ok := data["links"].([]any)
	if !ok {
		return jsonStr
	}
	if len(links) == 0 {
		return "# 0 shared links"
	}

	var sb strings.Builder
	sb.WriteString("```csv\nname,url,path_lower\n")
	for _, l := range links {
		lm, ok := l.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			csvEscape(str(lm, "name")),
			csvEscape(str(lm, "url")),
			csvEscape(str(lm, "path_lower")),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// revisionsCSV formats list_revisions responses.
func revisionsCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	entries, ok := data["entries"].([]any)
	if !ok {
		return jsonStr
	}
	if len(entries) == 0 {
		return "# 0 revisions"
	}

	var sb strings.Builder
	sb.WriteString("```csv\nrev,size,client_modified,server_modified\n")
	for _, e := range entries {
		em, ok := e.(map[string]any)
		if !ok {
			continue
		}
		size := ""
		if s, ok := em["size"].(float64); ok {
			size = fmt.Sprintf("%.0f", s)
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			str(em, "rev"),
			size,
			str(em, "client_modified"),
			str(em, "server_modified"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// accountCompact formats get_current_account response as key-value text.
func accountCompact(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("account_id: %s\n", str(data, "account_id")))
	if name, ok := data["name"].(map[string]any); ok {
		sb.WriteString(fmt.Sprintf("name: %s\n", str(name, "display_name")))
	}
	sb.WriteString(fmt.Sprintf("email: %s\n", str(data, "email")))
	if v := str(data, "country"); v != "" {
		sb.WriteString(fmt.Sprintf("country: %s\n", v))
	}
	if v := str(data, "account_type"); v == "" {
		if at, ok := data["account_type"].(map[string]any); ok {
			sb.WriteString(fmt.Sprintf("account_type: %s\n", str(at, ".tag")))
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
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
