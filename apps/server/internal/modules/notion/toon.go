package notion

import (
	"encoding/json"
	"fmt"
	"strings"
)

// =============================================================================
// TOON Format Converter for Notion API Responses
// Converts verbose JSON to compact 2D TOON format
// =============================================================================

// ToTOON converts Notion API JSON response to TOON format based on tool name
func ToTOON(toolName string, jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr // Return original if parse fails
	}

	switch toolName {
	case "search", "query_database":
		return listPagesTOON(data)
	case "get_page", "create_page", "update_page":
		return pageTOON(data)
	case "get_page_content":
		// Use Markdown format for page content (more readable for LLMs)
		return BlocksToMarkdown(jsonStr)
	case "get_database":
		return databaseTOON(data)
	case "list_users":
		return usersTOON(data)
	case "get_user", "get_bot_user":
		return userTOON(data)
	case "list_comments":
		return commentsTOON(data)
	case "add_comment":
		return commentTOON(data)
	case "append_blocks":
		return appendBlocksResultTOON(data)
	case "delete_block":
		return deleteBlockResultTOON(data)
	default:
		return jsonStr
	}
}

// =============================================================================
// Page conversions
// =============================================================================

// listPagesTOON converts search/query results to TOON
// Format: pages[N]{id,title,type}:
func listPagesTOON(data map[string]any) string {
	results, ok := data["results"].([]any)
	if !ok {
		return "pages[0]{id,title,type}:"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("pages[%d]{id,title,type}:\n", len(results)))

	for _, item := range results {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}

		id := getString(obj, "id")
		objType := getString(obj, "object") // "page" or "database"
		title := extractTitle(obj)

		sb.WriteString(fmt.Sprintf("  %s,%s,%s\n", id, toonEscape(title), objType))
	}

	// Add pagination info if has_more
	if hasMore, ok := data["has_more"].(bool); ok && hasMore {
		if cursor, ok := data["next_cursor"].(string); ok {
			sb.WriteString(fmt.Sprintf("next_cursor:%s\n", cursor))
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// pageTOON converts single page to TOON
// Format: page{id,title,parent_id,url}:
func pageTOON(data map[string]any) string {
	id := getString(data, "id")
	title := extractTitle(data)
	parentID := extractParentID(data)
	url := getString(data, "url")

	return fmt.Sprintf("page{id,title,parent_id,url}:\n  %s,%s,%s,%s",
		id, toonEscape(title), parentID, url)
}

// =============================================================================
// Block conversions
// =============================================================================

// appendBlocksResultTOON converts append_blocks response
func appendBlocksResultTOON(data map[string]any) string {
	results, ok := data["results"].([]any)
	if !ok {
		return "appended[0]{id,type}:"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("appended[%d]{id,type}:\n", len(results)))

	for _, item := range results {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := getString(block, "id")
		blockType := getString(block, "type")
		sb.WriteString(fmt.Sprintf("  %s,%s\n", id, blockType))
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// deleteBlockResultTOON converts delete response
func deleteBlockResultTOON(data map[string]any) string {
	id := getString(data, "id")
	archived := false
	if a, ok := data["archived"].(bool); ok {
		archived = a
	}
	return fmt.Sprintf("deleted{id,archived}:\n  %s,%t", id, archived)
}

// =============================================================================
// Database conversions
// =============================================================================

// databaseTOON converts database schema to TOON
// Format: database{id,title}: followed by properties[N]{name,type}:
func databaseTOON(data map[string]any) string {
	id := getString(data, "id")
	title := extractDatabaseTitle(data)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("database{id,title}:\n  %s,%s\n", id, toonEscape(title)))

	// Extract properties schema
	props, ok := data["properties"].(map[string]any)
	if ok && len(props) > 0 {
		sb.WriteString(fmt.Sprintf("properties[%d]{name,type}:\n", len(props)))
		for name, prop := range props {
			if propMap, ok := prop.(map[string]any); ok {
				propType := getString(propMap, "type")
				sb.WriteString(fmt.Sprintf("  %s,%s\n", toonEscape(name), propType))
			}
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// =============================================================================
// User conversions
// =============================================================================

// usersTOON converts user list to TOON
func usersTOON(data map[string]any) string {
	results, ok := data["results"].([]any)
	if !ok {
		return "users[0]{id,name,type}:"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("users[%d]{id,name,type}:\n", len(results)))

	for _, item := range results {
		user, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := getString(user, "id")
		name := getString(user, "name")
		userType := getString(user, "type")
		sb.WriteString(fmt.Sprintf("  %s,%s,%s\n", id, toonEscape(name), userType))
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// userTOON converts single user to TOON
func userTOON(data map[string]any) string {
	id := getString(data, "id")
	name := getString(data, "name")
	userType := getString(data, "type")
	return fmt.Sprintf("user{id,name,type}:\n  %s,%s,%s", id, toonEscape(name), userType)
}

// =============================================================================
// Comment conversions
// =============================================================================

// commentsTOON converts comment list to TOON
func commentsTOON(data map[string]any) string {
	results, ok := data["results"].([]any)
	if !ok {
		return "comments[0]{id,text}:"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("comments[%d]{id,text}:\n", len(results)))

	for _, item := range results {
		comment, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := getString(comment, "id")
		text := extractRichText(comment, "rich_text")
		sb.WriteString(fmt.Sprintf("  %s,%s\n", id, toonEscape(text)))
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// commentTOON converts single comment to TOON
func commentTOON(data map[string]any) string {
	id := getString(data, "id")
	text := extractRichText(data, "rich_text")
	return fmt.Sprintf("comment{id,text}:\n  %s,%s", id, toonEscape(text))
}

// =============================================================================
// Helper functions
// =============================================================================

func getString(obj map[string]any, key string) string {
	if v, ok := obj[key].(string); ok {
		return v
	}
	return ""
}

// extractTitle extracts title from page/database object
func extractTitle(obj map[string]any) string {
	props, ok := obj["properties"].(map[string]any)
	if !ok {
		return ""
	}

	// Try common title property names
	for _, name := range []string{"title", "Title", "Name", "name"} {
		if prop, ok := props[name].(map[string]any); ok {
			if titleArr, ok := prop["title"].([]any); ok {
				return extractPlainText(titleArr)
			}
		}
	}

	// Fallback: search for any title type property
	for _, prop := range props {
		if propMap, ok := prop.(map[string]any); ok {
			if titleArr, ok := propMap["title"].([]any); ok {
				return extractPlainText(titleArr)
			}
		}
	}

	return ""
}

// extractDatabaseTitle extracts title from database object
func extractDatabaseTitle(obj map[string]any) string {
	if titleArr, ok := obj["title"].([]any); ok {
		return extractPlainText(titleArr)
	}
	return ""
}

// extractPlainText extracts plain_text from rich_text array
func extractPlainText(arr []any) string {
	var texts []string
	for _, item := range arr {
		if textObj, ok := item.(map[string]any); ok {
			if pt, ok := textObj["plain_text"].(string); ok {
				texts = append(texts, pt)
			}
		}
	}
	return strings.Join(texts, "")
}

// extractRichText extracts text from rich_text field
func extractRichText(obj map[string]any, key string) string {
	if arr, ok := obj[key].([]any); ok {
		return extractPlainText(arr)
	}
	return ""
}

// extractParentID extracts parent ID from page object
func extractParentID(obj map[string]any) string {
	parent, ok := obj["parent"].(map[string]any)
	if !ok {
		return ""
	}

	if id, ok := parent["page_id"].(string); ok {
		return id
	}
	if id, ok := parent["database_id"].(string); ok {
		return id
	}
	if _, ok := parent["workspace"].(bool); ok {
		return "workspace"
	}
	return ""
}

// toonEscape escapes a value for TOON format (CSV-like)
func toonEscape(s string) string {
	if s == "" {
		return ""
	}
	needsQuote := strings.ContainsAny(s, ",\"\n\r")
	if !needsQuote {
		return s
	}
	escaped := strings.ReplaceAll(s, "\"", "\"\"")
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	return "\"" + escaped + "\""
}
