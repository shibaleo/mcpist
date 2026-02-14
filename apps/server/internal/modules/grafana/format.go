package grafana

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
	case "search":
		return searchToCSV(jsonStr)
	case "list_datasources":
		return datasourcesToCSV(jsonStr)
	case "list_alerts":
		return alertsToCSV(jsonStr)
	case "query_annotations":
		return annotationsToCSV(jsonStr)
	case "list_folders":
		return foldersToCSV(jsonStr)
	// Read: single items → MD
	case "get_dashboard":
		return dashboardToCompact(jsonStr)
	case "get_datasource":
		return datasourceToCompact(jsonStr)
	case "get_alert":
		return alertToCompact(jsonStr)
	// Write
	case "create_update_dashboard":
		return pickKeys(jsonStr, "id", "uid", "url", "status", "version")
	case "delete_dashboard":
		return pickKeys(jsonStr, "title", "message")
	case "create_annotation":
		return pickKeys(jsonStr, "id", "message")
	case "delete_annotation":
		return pickKeys(jsonStr, "message")
	case "create_folder":
		return pickKeys(jsonStr, "id", "uid", "title")
	case "delete_folder":
		return pickKeys(jsonStr, "title", "message")
	case "create_alert_rule":
		return pickKeys(jsonStr, "uid", "title", "folderUID", "ruleGroup")
	// Contact Points
	case "list_contact_points":
		return contactPointsToCSV(jsonStr)
	case "create_contact_point", "update_contact_point":
		return pickKeys(jsonStr, "uid", "name", "type")
	case "delete_contact_point":
		return pickKeys(jsonStr, "message")
	// Notification Policies — tree structure, keep as-is
	case "get_notification_policy", "update_notification_policy":
		return jsonStr
	// query_datasource returns free-form data, keep as-is
	case "query_datasource":
		return jsonStr
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

// searchToCSV: uid,title,type,tags
func searchToCSV(jsonStr string) string {
	var items []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
		return jsonStr
	}
	if len(items) == 0 {
		return "# 0 results"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nuid,title,type,tags\n")
	for _, item := range items {
		tags := ""
		if t, ok := item["tags"].([]any); ok && len(t) > 0 {
			strs := make([]string, 0, len(t))
			for _, tag := range t {
				if s, ok := tag.(string); ok {
					strs = append(strs, s)
				}
			}
			tags = strings.Join(strs, ";")
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			csvEscape(str(item, "uid")),
			csvEscape(str(item, "title")),
			str(item, "type"),
			csvEscape(tags),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// dashboardToCompact: dashboard summary
func dashboardToCompact(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	// Meta info
	if meta, ok := wrapper["meta"].(map[string]any); ok {
		sb.WriteString(fmt.Sprintf("# %s\n", str(meta, "slug")))
		if folder := str(meta, "folderTitle"); folder != "" {
			sb.WriteString(fmt.Sprintf("- **Folder**: %s\n", folder))
		}
		if ver := intVal(meta, "version"); ver > 0 {
			sb.WriteString(fmt.Sprintf("- **Version**: %d\n", ver))
		}
	}
	// Dashboard info
	if dash, ok := wrapper["dashboard"].(map[string]any); ok {
		if title := str(dash, "title"); title != "" {
			if sb.Len() == 0 {
				sb.WriteString(fmt.Sprintf("# %s\n", title))
			}
		}
		sb.WriteString(fmt.Sprintf("- **UID**: %s\n", str(dash, "uid")))
		// Panel count
		if panels, ok := dash["panels"].([]any); ok {
			sb.WriteString(fmt.Sprintf("- **Panels**: %d\n", len(panels)))
			// List panel titles
			for _, raw := range panels {
				p, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				title := str(p, "title")
				pType := str(p, "type")
				if title != "" {
					sb.WriteString(fmt.Sprintf("  - %s (%s)\n", title, pType))
				}
			}
		}
		if tags, ok := dash["tags"].([]any); ok && len(tags) > 0 {
			strs := make([]string, 0, len(tags))
			for _, t := range tags {
				if s, ok := t.(string); ok {
					strs = append(strs, s)
				}
			}
			sb.WriteString(fmt.Sprintf("- **Tags**: %s\n", strings.Join(strs, ", ")))
		}
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// datasourcesToCSV: uid,name,type,access
func datasourcesToCSV(jsonStr string) string {
	var dss []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &dss); err != nil {
		return jsonStr
	}
	if len(dss) == 0 {
		return "# 0 datasources"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nuid,name,type,access\n")
	for _, ds := range dss {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			csvEscape(str(ds, "uid")),
			csvEscape(str(ds, "name")),
			str(ds, "type"),
			str(ds, "access"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// datasourceToCompact: single datasource
func datasourceToCompact(jsonStr string) string {
	var ds map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &ds); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(ds, "name")))
	sb.WriteString(fmt.Sprintf("- **UID**: %s\n", str(ds, "uid")))
	sb.WriteString(fmt.Sprintf("- **Type**: %s\n", str(ds, "type")))
	if access := str(ds, "access"); access != "" {
		sb.WriteString(fmt.Sprintf("- **Access**: %s\n", access))
	}
	if url := str(ds, "url"); url != "" {
		sb.WriteString(fmt.Sprintf("- **URL**: %s\n", url))
	}
	if db := str(ds, "database"); db != "" {
		sb.WriteString(fmt.Sprintf("- **Database**: %s\n", db))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// alertsToCSV: uid,title,folderUID,ruleGroup
func alertsToCSV(jsonStr string) string {
	var alerts []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &alerts); err != nil {
		return jsonStr
	}
	if len(alerts) == 0 {
		return "# 0 alerts"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nuid,title,folderUID,ruleGroup\n")
	for _, a := range alerts {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			csvEscape(str(a, "uid")),
			csvEscape(str(a, "title")),
			csvEscape(str(a, "folderUID")),
			csvEscape(str(a, "ruleGroup")),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// contactPointsToCSV: uid,name,type
func contactPointsToCSV(jsonStr string) string {
	var cps []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &cps); err != nil {
		return jsonStr
	}
	if len(cps) == 0 {
		return "# 0 contact points"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nuid,name,type\n")
	for _, cp := range cps {
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			csvEscape(str(cp, "uid")),
			csvEscape(str(cp, "name")),
			str(cp, "type"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// alertToCompact: single alert rule
func alertToCompact(jsonStr string) string {
	var a map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &a); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(a, "title")))
	sb.WriteString(fmt.Sprintf("- **UID**: %s\n", str(a, "uid")))
	sb.WriteString(fmt.Sprintf("- **Rule Group**: %s\n", str(a, "ruleGroup")))
	if folder := str(a, "folderUID"); folder != "" {
		sb.WriteString(fmt.Sprintf("- **Folder**: %s\n", folder))
	}
	if cond := str(a, "condition"); cond != "" {
		sb.WriteString(fmt.Sprintf("- **Condition**: %s\n", cond))
	}
	if forDur := str(a, "for"); forDur != "" {
		sb.WriteString(fmt.Sprintf("- **For**: %s\n", forDur))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// annotationsToCSV: id,dashboardUID,text,time,tags
func annotationsToCSV(jsonStr string) string {
	var annotations []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &annotations); err != nil {
		return jsonStr
	}
	if len(annotations) == 0 {
		return "# 0 annotations"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,dashboardUID,text,time,tags\n")
	for _, a := range annotations {
		tags := ""
		if t, ok := a["tags"].([]any); ok && len(t) > 0 {
			strs := make([]string, 0, len(t))
			for _, tag := range t {
				if s, ok := tag.(string); ok {
					strs = append(strs, s)
				}
			}
			tags = strings.Join(strs, ";")
		}
		text := str(a, "text")
		if len(text) > 80 {
			text = text[:80] + "..."
		}
		sb.WriteString(fmt.Sprintf("%d,%s,%s,%d,%s\n",
			intVal(a, "id"),
			csvEscape(str(a, "dashboardUID")),
			csvEscape(text),
			intVal(a, "time"),
			csvEscape(tags),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// foldersToCSV: uid,title
func foldersToCSV(jsonStr string) string {
	var folders []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &folders); err != nil {
		return jsonStr
	}
	if len(folders) == 0 {
		return "# 0 folders"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nuid,title\n")
	for _, f := range folders {
		sb.WriteString(fmt.Sprintf("%s,%s\n",
			csvEscape(str(f, "uid")),
			csvEscape(str(f, "title")),
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

func intVal(obj map[string]any, key string) int {
	if v, ok := obj[key].(float64); ok {
		return int(v)
	}
	return 0
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
