package google_apps_script

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
	case "list_projects":
		return projectsCSV(jsonStr)
	case "list_versions":
		return versionsCSV(jsonStr)
	case "list_deployments":
		return deploymentsCSV(jsonStr)
	case "list_executions", "list_processes":
		return processesCSV(jsonStr)
	case "create_project", "get_project":
		return pickKeys(jsonStr, "scriptId", "title")
	case "create_version":
		return pickKeys(jsonStr, "versionNumber", "description", "createTime")
	case "create_deployment":
		return pickKeys(jsonStr, "deploymentId")
	case "delete_deployment":
		return jsonStr
	default:
		return jsonStr
	}
}

// projectsCSV formats list_projects (Drive files) → CSV: id, name, modifiedTime.
func projectsCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	files, ok := data["files"].([]any)
	if !ok || len(files) == 0 {
		return "# 0 projects"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,modifiedTime\n")
	for _, item := range files {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			csvEscape(str(m, "id")),
			csvEscape(str(m, "name")),
			str(m, "modifiedTime"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// versionsCSV formats list_versions → CSV: versionNumber, description, createTime.
func versionsCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	items, ok := data["versions"].([]any)
	if !ok || len(items) == 0 {
		return "# 0 versions"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nversionNumber,description,createTime\n")
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		vn := ""
		if v, ok := m["versionNumber"].(float64); ok {
			vn = fmt.Sprintf("%d", int(v))
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			vn,
			csvEscape(str(m, "description")),
			str(m, "createTime"),
		))
	}
	sb.WriteString("```")

	if token := str(data, "nextPageToken"); token != "" {
		sb.WriteString(fmt.Sprintf("\nnextPageToken=%s", token))
	}
	return sb.String()
}

// deploymentsCSV formats list_deployments → CSV: deploymentId, versionNumber, description.
func deploymentsCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	items, ok := data["deployments"].([]any)
	if !ok || len(items) == 0 {
		return "# 0 deployments"
	}
	var sb strings.Builder
	sb.WriteString("```csv\ndeploymentId,versionNumber,description\n")
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		did := str(m, "deploymentId")
		vn := ""
		desc := ""
		if config, ok := m["deploymentConfig"].(map[string]any); ok {
			if v, ok := config["versionNumber"].(float64); ok {
				vn = fmt.Sprintf("%d", int(v))
			}
			desc = str(config, "description")
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			did,
			vn,
			csvEscape(desc),
		))
	}
	sb.WriteString("```")

	if token := str(data, "nextPageToken"); token != "" {
		sb.WriteString(fmt.Sprintf("\nnextPageToken=%s", token))
	}
	return sb.String()
}

// processesCSV formats list_executions / list_processes → CSV.
func processesCSV(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	items, ok := data["processes"].([]any)
	if !ok || len(items) == 0 {
		return "# 0 processes"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nprocessId,functionName,processStatus,startTime\n")
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			str(m, "processId"),
			csvEscape(str(m, "functionName")),
			str(m, "processStatus"),
			str(m, "startTime"),
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
