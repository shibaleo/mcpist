package notion

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// =============================================================================
// Format dispatcher — called by handlers when format != "json"
// =============================================================================

// formatResult applies the requested format to a JSON result string.
// toolName determines which formatter to use; format is "md" or "csv".
func formatResult(toolName, format, jsonStr string) string {
	switch format {
	case "md":
		return formatMDResult(toolName, jsonStr)
	case "csv":
		return formatCSVResult(toolName, jsonStr)
	default:
		return jsonStr
	}
}

// =============================================================================
// Markdown formatters (for pages, blocks, comments)
// =============================================================================

func formatMDResult(toolName, jsonStr string) string {
	switch toolName {
	case "get_page":
		return pageToMD(jsonStr)
	case "get_page_content":
		return BlocksToMarkdown(jsonStr)
	case "list_comments":
		return commentsToMD(jsonStr)
	default:
		return jsonStr
	}
}

// pageToMD converts a Page JSON to Markdown with properties table
func pageToMD(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	var sb strings.Builder

	// Title
	title := extractTitle(data)
	if title != "" {
		sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	}

	// Metadata
	id := getString(data, "id")
	url := getString(data, "url")
	if id != "" {
		sb.WriteString(fmt.Sprintf("- **ID**: %s\n", id))
	}
	if url != "" {
		sb.WriteString(fmt.Sprintf("- **URL**: %s\n", url))
	}

	// Parent
	if parent, ok := data["parent"].(map[string]any); ok {
		if pid, ok := parent["page_id"].(string); ok {
			sb.WriteString(fmt.Sprintf("- **Parent page**: %s\n", pid))
		} else if did, ok := parent["database_id"].(string); ok {
			sb.WriteString(fmt.Sprintf("- **Parent database**: %s\n", did))
		}
	}

	// Properties as table
	props, ok := data["properties"].(map[string]any)
	if ok && len(props) > 0 {
		sb.WriteString("\n## Properties\n\n")
		sb.WriteString("| Property | Value |\n|---|---|\n")

		// Sort property names for stable output
		names := make([]string, 0, len(props))
		for name := range props {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			prop := props[name]
			val := extractPropertyValue(prop)
			sb.WriteString(fmt.Sprintf("| %s | %s |\n", name, val))
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// commentsToMD converts comment list JSON to Markdown
func commentsToMD(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	results, ok := data["results"].([]any)
	if !ok || len(results) == 0 {
		return "*No comments*"
	}

	var sb strings.Builder
	for i, item := range results {
		comment, ok := item.(map[string]any)
		if !ok {
			continue
		}
		text := extractRichText(comment, "rich_text")
		author := ""
		if by, ok := comment["created_by"].(map[string]any); ok {
			author = getString(by, "name")
			if author == "" {
				author = getString(by, "id")
			}
		}
		time := getString(comment, "created_time")
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}
		if author != "" {
			sb.WriteString(fmt.Sprintf("**%s**", author))
			if time != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", time))
			}
			sb.WriteString("\n\n")
		}
		sb.WriteString(text + "\n")
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// =============================================================================
// CSV formatters (for databases, search, users)
// =============================================================================

func formatCSVResult(toolName, jsonStr string) string {
	switch toolName {
	case "query_database":
		return queryDatabaseToCSV(jsonStr)
	case "search":
		return searchToCSV(jsonStr)
	case "get_database":
		return databaseSchemaToCSV(jsonStr)
	case "list_users":
		return usersToCSV(jsonStr)
	case "get_user", "get_bot_user":
		return userToCSV(jsonStr)
	default:
		return jsonStr
	}
}

// queryDatabaseToCSV converts query_database results to CSV with property values
func queryDatabaseToCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	results, ok := data["results"].([]any)
	if !ok || len(results) == 0 {
		return "# 0 rows"
	}

	// Collect all property names from first row to determine columns
	var columns []string
	if first, ok := results[0].(map[string]any); ok {
		if props, ok := first["properties"].(map[string]any); ok {
			columns = make([]string, 0, len(props))
			for name := range props {
				columns = append(columns, name)
			}
			sort.Strings(columns)
		}
	}

	if len(columns) == 0 {
		return "# 0 columns"
	}

	var sb strings.Builder
	// Header
	sb.WriteString(strings.Join(columns, ","))
	sb.WriteString("\n")

	// Rows
	for _, item := range results {
		page, ok := item.(map[string]any)
		if !ok {
			continue
		}
		props, _ := page["properties"].(map[string]any)

		vals := make([]string, 0, len(columns))
		for _, col := range columns {
			val := ""
			if prop, ok := props[col]; ok {
				val = extractPropertyValue(prop)
			}
			vals = append(vals, csvEscape(val))
		}
		sb.WriteString(strings.Join(vals, ","))
		sb.WriteString("\n")
	}

	// Pagination
	if hasMore, ok := data["has_more"].(bool); ok && hasMore {
		if cursor, ok := data["next_cursor"].(string); ok {
			sb.WriteString(fmt.Sprintf("# next_cursor=%s\n", cursor))
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// searchToCSV converts search results to CSV
func searchToCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	results, ok := data["results"].([]any)
	if !ok || len(results) == 0 {
		return "# 0 results"
	}

	var sb strings.Builder
	sb.WriteString("id,type,title\n")

	for _, item := range results {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := getString(obj, "id")
		objType := getString(obj, "object")
		title := extractTitle(obj)
		if title == "" && objType == "database" {
			title = extractDatabaseTitle(obj)
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n", csvEscape(id), objType, csvEscape(title)))
	}

	if hasMore, ok := data["has_more"].(bool); ok && hasMore {
		if cursor, ok := data["next_cursor"].(string); ok {
			sb.WriteString(fmt.Sprintf("# next_cursor=%s\n", cursor))
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// databaseSchemaToCSV converts database schema to CSV of property names and types
func databaseSchemaToCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	id := getString(data, "id")
	title := extractDatabaseTitle(data)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s (id=%s)\n", title, id))

	props, ok := data["properties"].(map[string]any)
	if !ok || len(props) == 0 {
		sb.WriteString("# 0 properties")
		return sb.String()
	}

	sb.WriteString("name,type,details\n")

	names := make([]string, 0, len(props))
	for name := range props {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		prop, ok := props[name].(map[string]any)
		if !ok {
			continue
		}
		propType := getString(prop, "type")
		details := extractSchemaDetails(prop, propType)
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n", csvEscape(name), propType, csvEscape(details)))
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// extractSchemaDetails extracts type-specific details from a property schema
func extractSchemaDetails(prop map[string]any, propType string) string {
	switch propType {
	case "select", "multi_select", "status":
		if inner, ok := prop[propType].(map[string]any); ok {
			if options, ok := inner["options"].([]any); ok {
				names := make([]string, 0, len(options))
				for _, opt := range options {
					if m, ok := opt.(map[string]any); ok {
						names = append(names, getString(m, "name"))
					}
				}
				return strings.Join(names, "|")
			}
		}
	case "formula":
		if inner, ok := prop["formula"].(map[string]any); ok {
			return getString(inner, "expression")
		}
	case "relation":
		if inner, ok := prop["relation"].(map[string]any); ok {
			return getString(inner, "database_id")
		}
	case "rollup":
		if inner, ok := prop["rollup"].(map[string]any); ok {
			rel := getString(inner, "relation_property_name")
			fn := getString(inner, "function")
			rp := getString(inner, "rollup_property_name")
			if rel != "" {
				return fmt.Sprintf("%s.%s(%s)", rel, rp, fn)
			}
		}
	}
	return ""
}

// usersToCSV converts user list to CSV
func usersToCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	results, ok := data["results"].([]any)
	if !ok || len(results) == 0 {
		return "# 0 users"
	}

	var sb strings.Builder
	sb.WriteString("id,name,type,email\n")

	for _, item := range results {
		user, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := getString(user, "id")
		name := getString(user, "name")
		userType := getString(user, "type")
		email := ""
		if person, ok := user["person"].(map[string]any); ok {
			email = getString(person, "email")
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n", csvEscape(id), csvEscape(name), userType, csvEscape(email)))
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// userToCSV converts single user to CSV
func userToCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	id := getString(data, "id")
	name := getString(data, "name")
	userType := getString(data, "type")
	email := ""
	if person, ok := data["person"].(map[string]any); ok {
		email = getString(person, "email")
	}

	var sb strings.Builder
	sb.WriteString("id,name,type,email\n")
	sb.WriteString(fmt.Sprintf("%s,%s,%s,%s", csvEscape(id), csvEscape(name), userType, csvEscape(email)))
	return sb.String()
}

// =============================================================================
// Property value extraction (shared by MD and CSV)
// =============================================================================

// extractPropertyValue extracts a human-readable value from a Notion property
func extractPropertyValue(prop any) string {
	propMap, ok := prop.(map[string]any)
	if !ok {
		return ""
	}

	propType := getString(propMap, "type")
	switch propType {
	case "title":
		if arr, ok := propMap["title"].([]any); ok {
			return extractPlainText(arr)
		}
	case "rich_text":
		if arr, ok := propMap["rich_text"].([]any); ok {
			return extractPlainText(arr)
		}
	case "number":
		if v, ok := propMap["number"].(float64); ok {
			if v == float64(int64(v)) {
				return fmt.Sprintf("%d", int64(v))
			}
			return fmt.Sprintf("%g", v)
		}
	case "select":
		if sel, ok := propMap["select"].(map[string]any); ok {
			return getString(sel, "name")
		}
	case "multi_select":
		if arr, ok := propMap["multi_select"].([]any); ok {
			var names []string
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					names = append(names, getString(m, "name"))
				}
			}
			return strings.Join(names, ";")
		}
	case "status":
		if status, ok := propMap["status"].(map[string]any); ok {
			return getString(status, "name")
		}
	case "date":
		if date, ok := propMap["date"].(map[string]any); ok {
			start := getString(date, "start")
			end := getString(date, "end")
			if end != "" {
				return start + " → " + end
			}
			return start
		}
	case "checkbox":
		if v, ok := propMap["checkbox"].(bool); ok {
			if v {
				return "true"
			}
			return "false"
		}
	case "url":
		if v, ok := propMap["url"].(string); ok {
			return v
		}
	case "email":
		if v, ok := propMap["email"].(string); ok {
			return v
		}
	case "phone_number":
		if v, ok := propMap["phone_number"].(string); ok {
			return v
		}
	case "formula":
		if f, ok := propMap["formula"].(map[string]any); ok {
			fType := getString(f, "type")
			switch fType {
			case "string":
				if v, ok := f["string"].(string); ok {
					return v
				}
			case "number":
				if v, ok := f["number"].(float64); ok {
					return fmt.Sprintf("%g", v)
				}
			case "boolean":
				if v, ok := f["boolean"].(bool); ok {
					return fmt.Sprintf("%t", v)
				}
			case "date":
				if d, ok := f["date"].(map[string]any); ok {
					return getString(d, "start")
				}
			}
		}
	case "relation":
		if arr, ok := propMap["relation"].([]any); ok {
			var ids []string
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					ids = append(ids, getString(m, "id"))
				}
			}
			return strings.Join(ids, ";")
		}
	case "rollup":
		if r, ok := propMap["rollup"].(map[string]any); ok {
			rType := getString(r, "type")
			switch rType {
			case "number":
				if v, ok := r["number"].(float64); ok {
					return fmt.Sprintf("%g", v)
				}
			case "array":
				if arr, ok := r["array"].([]any); ok {
					var vals []string
					for _, item := range arr {
						vals = append(vals, extractPropertyValue(item))
					}
					return strings.Join(vals, ";")
				}
			}
		}
	case "people":
		if arr, ok := propMap["people"].([]any); ok {
			var names []string
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					name := getString(m, "name")
					if name == "" {
						name = getString(m, "id")
					}
					names = append(names, name)
				}
			}
			return strings.Join(names, ";")
		}
	case "files":
		if arr, ok := propMap["files"].([]any); ok {
			var urls []string
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					if ext, ok := m["external"].(map[string]any); ok {
						urls = append(urls, getString(ext, "url"))
					} else if file, ok := m["file"].(map[string]any); ok {
						urls = append(urls, getString(file, "url"))
					} else {
						urls = append(urls, getString(m, "name"))
					}
				}
			}
			return strings.Join(urls, ";")
		}
	case "created_time":
		if v, ok := propMap["created_time"].(string); ok {
			return v
		}
	case "last_edited_time":
		if v, ok := propMap["last_edited_time"].(string); ok {
			return v
		}
	case "created_by":
		if by, ok := propMap["created_by"].(map[string]any); ok {
			return getString(by, "name")
		}
	case "last_edited_by":
		if by, ok := propMap["last_edited_by"].(map[string]any); ok {
			return getString(by, "name")
		}
	case "unique_id":
		if uid, ok := propMap["unique_id"].(map[string]any); ok {
			prefix := getString(uid, "prefix")
			if num, ok := uid["number"].(float64); ok {
				if prefix != "" {
					return fmt.Sprintf("%s-%d", prefix, int64(num))
				}
				return fmt.Sprintf("%d", int64(num))
			}
		}
	}

	return ""
}

// =============================================================================
// Common response formatters — compact by default, format=json for raw
// =============================================================================

// compactReadResult returns compact format by default, raw JSON only when format=json.
func compactReadResult(params map[string]any, toolName, defaultFmt, jsonStr string) string {
	if f, _ := params["format"].(string); f == "json" {
		return jsonStr
	}
	return formatResult(toolName, defaultFmt, jsonStr)
}

// compactWriteResult extracts only the specified keys from a JSON response.
// Returns raw JSON when params["format"] == "json".
func compactWriteResult(params map[string]any, jsonStr string, keys ...string) (string, error) {
	if f, _ := params["format"].(string); f == "json" {
		return jsonStr, nil
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr, nil
	}
	result := make(map[string]any, len(keys))
	for _, k := range keys {
		if v, ok := data[k]; ok && v != nil {
			result[k] = v
		}
	}
	out, err := json.Marshal(result)
	if err != nil {
		return jsonStr, nil
	}
	return string(out), nil
}

// compactBlockListResult returns {"block_count": N} from an append_blocks response.
// Returns raw JSON when params["format"] == "json".
func compactBlockListResult(params map[string]any, jsonStr string) (string, error) {
	if f, _ := params["format"].(string); f == "json" {
		return jsonStr, nil
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr, nil
	}
	count := 0
	if results, ok := data["results"].([]any); ok {
		count = len(results)
	}
	out, _ := json.Marshal(map[string]any{"block_count": count})
	return string(out), nil
}

// =============================================================================
// Helpers
// =============================================================================

func getString(obj map[string]any, key string) string {
	if v, ok := obj[key].(string); ok {
		return v
	}
	return ""
}

// csvEscape escapes a value for CSV (RFC 4180)
func csvEscape(s string) string {
	if s == "" {
		return ""
	}
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
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
