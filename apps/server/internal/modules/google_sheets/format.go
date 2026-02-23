package google_sheets

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
	case "search_spreadsheets":
		return spreadsheetsCSV(jsonStr)
	case "list_sheets":
		return sheetsCSV(jsonStr)
	case "create_spreadsheet":
		return pickKeys(jsonStr, "spreadsheetId", "properties")
	case "update_values":
		return pickKeys(jsonStr, "updatedRange", "updatedRows", "updatedCells")
	case "batch_update_values":
		return pickKeys(jsonStr, "totalUpdatedRows", "totalUpdatedCells", "totalUpdatedSheets")
	case "find_replace":
		return findReplaceCompact(jsonStr)
	case "copy_sheet_to":
		return pickKeys(jsonStr, "sheetId", "title")
	case "clear_values":
		return pickKeys(jsonStr, "clearedRange")
	default:
		return jsonStr
	}
}

// spreadsheetsCSV formats search_spreadsheets → CSV: id, name, modifiedTime.
func spreadsheetsCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	files, ok := data["files"].([]any)
	if !ok || len(files) == 0 {
		return "# 0 spreadsheets"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,modifiedTime\n")
	for _, item := range files {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			str(m, "id"),
			csvEscape(str(m, "name")),
			str(m, "modifiedTime"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// sheetsCSV formats list_sheets → CSV: sheetId, title, index, sheetType.
func sheetsCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	sheets, ok := data["sheets"].([]any)
	if !ok || len(sheets) == 0 {
		return "# 0 sheets"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nsheetId,title,index,sheetType\n")
	for _, item := range sheets {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		props, ok := m["properties"].(map[string]any)
		if !ok {
			continue
		}
		sheetID := ""
		if v, ok := props["sheetId"].(float64); ok {
			sheetID = fmt.Sprintf("%d", int(v))
		}
		idx := ""
		if v, ok := props["index"].(float64); ok {
			idx = fmt.Sprintf("%d", int(v))
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			sheetID,
			csvEscape(str(props, "title")),
			idx,
			str(props, "sheetType"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// findReplaceCompact extracts occurrencesChanged and sheetsChanged from batchUpdate reply.
func findReplaceCompact(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	replies, ok := data["replies"].([]any)
	if !ok || len(replies) == 0 {
		return jsonStr
	}
	for _, reply := range replies {
		rm, ok := reply.(map[string]any)
		if !ok {
			continue
		}
		if fr, ok := rm["findReplace"].(map[string]any); ok {
			result := map[string]any{}
			if v, ok := fr["occurrencesChanged"]; ok {
				result["occurrencesChanged"] = v
			}
			if v, ok := fr["sheetsChanged"]; ok {
				result["sheetsChanged"] = v
			}
			out, err := json.Marshal(result)
			if err != nil {
				return jsonStr
			}
			return string(out)
		}
	}
	return jsonStr
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
