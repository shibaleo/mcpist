package supabase

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
	case "list_organizations":
		return organizationsToCSV(jsonStr)
	case "list_projects":
		return projectsToCSV(jsonStr)
	case "list_tables":
		return tablesToCSV(jsonStr)
	case "list_migrations":
		return migrationsToCSV(jsonStr)
	case "get_api_keys":
		return apiKeysToCSV(jsonStr)
	case "list_edge_functions":
		return edgeFunctionsToCSV(jsonStr)
	case "list_storage_buckets":
		return storageBucketsToCSV(jsonStr)
	// Read: single items → MD
	case "get_project":
		return projectToCompact(jsonStr)
	case "get_edge_function":
		return edgeFunctionToCompact(jsonStr)
	case "get_storage_config":
		return storageConfigToCompact(jsonStr)
	// Read: advisors → MD
	case "get_security_advisors", "get_performance_advisors":
		return advisorsToCompact(jsonStr)
	// Composite: already compacted in handler
	case "describe_project", "inspect_health":
		return jsonStr
	// Query results
	case "run_query":
		return queryResultToCSV(jsonStr)
	// Write
	case "apply_migration":
		return pickKeys(jsonStr, "success", "migration")
	case "get_project_url":
		return pickKeys(jsonStr, "url")
	case "get_logs":
		return logsToCompact(jsonStr)
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

// organizationsToCSV: id,name
func organizationsToCSV(jsonStr string) string {
	var orgs []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &orgs); err != nil {
		return jsonStr
	}
	if len(orgs) == 0 {
		return "# 0 organizations"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name\n")
	for _, o := range orgs {
		sb.WriteString(fmt.Sprintf("%s,%s\n",
			csvEscape(str(o, "id")),
			csvEscape(str(o, "name")),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// projectsToCSV: id,name,region,status
func projectsToCSV(jsonStr string) string {
	var projects []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &projects); err != nil {
		return jsonStr
	}
	if len(projects) == 0 {
		return "# 0 projects"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,region,status\n")
	for _, p := range projects {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			csvEscape(str(p, "id")),
			csvEscape(str(p, "name")),
			str(p, "region"),
			str(p, "status"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// projectToCompact: single project
func projectToCompact(jsonStr string) string {
	var p map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(p, "name")))
	sb.WriteString(fmt.Sprintf("- **ID**: %s\n", str(p, "id")))
	if ref := str(p, "ref"); ref != "" {
		sb.WriteString(fmt.Sprintf("- **Ref**: %s\n", ref))
	}
	if region := str(p, "region"); region != "" {
		sb.WriteString(fmt.Sprintf("- **Region**: %s\n", region))
	}
	if status := str(p, "status"); status != "" {
		sb.WriteString(fmt.Sprintf("- **Status**: %s\n", status))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// tablesToCSV: schema,name,column_count
func tablesToCSV(jsonStr string) string {
	var tables []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &tables); err != nil {
		return jsonStr
	}
	if len(tables) == 0 {
		return "# 0 tables"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nschema,name,column_count\n")
	for _, t := range tables {
		sb.WriteString(fmt.Sprintf("%s,%s,%d\n",
			csvEscape(str(t, "schema")),
			csvEscape(str(t, "name")),
			intVal(t, "column_count"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// migrationsToCSV: version,name
func migrationsToCSV(jsonStr string) string {
	var migrations []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &migrations); err != nil {
		return jsonStr
	}
	if len(migrations) == 0 {
		return "# 0 migrations"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nversion,name\n")
	for _, m := range migrations {
		sb.WriteString(fmt.Sprintf("%s,%s\n",
			csvEscape(str(m, "version")),
			csvEscape(str(m, "name")),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// queryResultToCSV: dynamic columns from query result
func queryResultToCSV(jsonStr string) string {
	var rows []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &rows); err != nil {
		return jsonStr
	}
	if len(rows) == 0 {
		return "# 0 rows"
	}
	// Collect column names from first row
	cols := make([]string, 0)
	for k := range rows[0] {
		cols = append(cols, k)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("```csv  # %d rows\n%s\n", len(rows), strings.Join(cols, ",")))
	for _, row := range rows {
		vals := make([]string, 0, len(cols))
		for _, c := range cols {
			vals = append(vals, csvEscape(fmt.Sprintf("%v", row[c])))
		}
		sb.WriteString(strings.Join(vals, ","))
		sb.WriteString("\n")
	}
	sb.WriteString("```")
	return sb.String()
}

// apiKeysToCSV: name,type
func apiKeysToCSV(jsonStr string) string {
	var keys []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &keys); err != nil {
		return jsonStr
	}
	if len(keys) == 0 {
		return "# 0 API keys"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nname,type\n")
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("%s,%s\n",
			csvEscape(str(k, "name")),
			str(k, "type"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// edgeFunctionsToCSV: slug,name,status,version
func edgeFunctionsToCSV(jsonStr string) string {
	var funcs []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &funcs); err != nil {
		return jsonStr
	}
	if len(funcs) == 0 {
		return "# 0 edge functions"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nslug,name,status,version\n")
	for _, f := range funcs {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%d\n",
			csvEscape(str(f, "slug")),
			csvEscape(str(f, "name")),
			str(f, "status"),
			intVal(f, "version"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// edgeFunctionToCompact: single function detail
func edgeFunctionToCompact(jsonStr string) string {
	var f map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &f); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(f, "name")))
	sb.WriteString(fmt.Sprintf("- **Slug**: %s\n", str(f, "slug")))
	if status := str(f, "status"); status != "" {
		sb.WriteString(fmt.Sprintf("- **Status**: %s\n", status))
	}
	if ver := intVal(f, "version"); ver > 0 {
		sb.WriteString(fmt.Sprintf("- **Version**: %d\n", ver))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// storageBucketsToCSV: name,public
func storageBucketsToCSV(jsonStr string) string {
	var buckets []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &buckets); err != nil {
		return jsonStr
	}
	if len(buckets) == 0 {
		return "# 0 storage buckets"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nname,public\n")
	for _, b := range buckets {
		sb.WriteString(fmt.Sprintf("%s,%v\n",
			csvEscape(str(b, "name")),
			boolVal(b, "public"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// storageConfigToCompact: storage config
func storageConfigToCompact(jsonStr string) string {
	var c map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &c); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString("# Storage Config\n")
	if limit := intVal(c, "fileSizeLimit"); limit > 0 {
		sb.WriteString(fmt.Sprintf("- **File Size Limit**: %d bytes\n", limit))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// advisorsToCompact: lints list
func advisorsToCompact(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	lints, ok := wrapper["lints"].([]any)
	if !ok || len(lints) == 0 {
		return "# 0 recommendations"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %d recommendations\n\n", len(lints)))
	for _, raw := range lints {
		l, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		level := str(l, "level")
		title := str(l, "title")
		desc := str(l, "description")
		sb.WriteString(fmt.Sprintf("### [%s] %s\n%s\n\n", level, title, desc))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// logsToCompact: logs as compact text
func logsToCompact(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	result, ok := wrapper["result"].([]any)
	if !ok || len(result) == 0 {
		return "# 0 log entries"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %d log entries\n\n```\n", len(result)))
	for _, raw := range result {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ts := str(entry, "timestamp")
		if len(ts) > 19 {
			ts = ts[:19]
		}
		msg := str(entry, "event_message")
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("[%s] %s\n", ts, msg))
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

func boolVal(obj map[string]any, key string) bool {
	if v, ok := obj[key].(bool); ok {
		return v
	}
	return false
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
