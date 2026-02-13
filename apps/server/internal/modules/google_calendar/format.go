package google_calendar

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
	case "list_calendars":
		return calendarsCSV(jsonStr)
	case "list_events":
		return eventsCSV(jsonStr)
	case "create_event", "update_event", "quick_add":
		return pickKeys(jsonStr, "id", "summary", "htmlLink")
	case "get_calendar":
		return pickKeys(jsonStr, "id", "summary", "timeZone")
	default:
		return jsonStr
	}
}

// calendarsCSV formats list_calendars response → CSV: id, summary, primary, accessRole.
func calendarsCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	items, ok := data["items"].([]any)
	if !ok || len(items) == 0 {
		return "# 0 calendars"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,summary,primary,accessRole\n")
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		primary := ""
		if p, ok := m["primary"].(bool); ok && p {
			primary = "true"
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			csvEscape(str(m, "id")),
			csvEscape(str(m, "summary")),
			primary,
			str(m, "accessRole"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// eventsCSV formats list_events response → CSV: id, summary, start, end, status.
func eventsCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	items, ok := data["items"].([]any)
	if !ok || len(items) == 0 {
		return "# 0 events"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,summary,start,end,status\n")
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
			csvEscape(str(m, "id")),
			csvEscape(str(m, "summary")),
			csvEscape(eventTime(m, "start")),
			csvEscape(eventTime(m, "end")),
			str(m, "status"),
		))
	}
	sb.WriteString("```")

	// Append pagination token if present
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

// eventTime extracts dateTime or date from an event's start/end object.
func eventTime(event map[string]any, key string) string {
	obj, ok := event[key].(map[string]any)
	if !ok {
		return ""
	}
	if dt := str(obj, "dateTime"); dt != "" {
		return dt
	}
	return str(obj, "date")
}

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
