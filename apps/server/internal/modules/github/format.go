package github

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
	case "list_repos":
		return reposToCSV(jsonStr)
	case "list_starred_repos":
		return starredReposToCSV(jsonStr)
	case "list_branches":
		return branchesToCSV(jsonStr)
	case "list_commits":
		return commitsToCSV(jsonStr)
	case "list_issues":
		return issuesToCSV(jsonStr)
	case "list_prs":
		return prsToCSV(jsonStr)
	case "list_pr_files":
		return prFilesToCSV(jsonStr)
	case "list_workflows":
		return workflowsToCSV(jsonStr)
	case "list_workflow_runs":
		return workflowRunsToCSV(jsonStr)
	case "list_orgs":
		return orgsToCSV(jsonStr)
	case "list_public_events":
		return eventsToCSV(jsonStr)
	// Search → CSV
	case "search_repos":
		return searchReposToCSV(jsonStr)
	case "search_code":
		return searchCodeToCSV(jsonStr)
	case "search_issues":
		return searchIssuesToCSV(jsonStr)
	// Read: single item → MD
	case "get_user":
		return userToCompact(jsonStr)
	case "get_repo":
		return repoToCompact(jsonStr)
	case "get_issue":
		return issueToCompact(jsonStr)
	case "get_pr":
		return prToCompact(jsonStr)
	case "get_file_content":
		return fileContentToCompact(jsonStr)
	// Composite: already compacted in handler
	case "describe_user", "describe_repo", "describe_pr":
		return jsonStr
	// Write
	case "create_issue", "update_issue":
		return pickKeys(jsonStr, "number", "html_url", "state")
	case "add_issue_comment":
		return pickKeys(jsonStr, "id", "html_url")
	case "create_pr":
		return pickKeys(jsonStr, "number", "html_url", "state", "draft")
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

// userToCompact: user profile
func userToCompact(jsonStr string) string {
	var u map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &u); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s", str(u, "login")))
	if name := str(u, "name"); name != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", name))
	}
	sb.WriteString("\n")
	if bio := str(u, "bio"); bio != "" {
		sb.WriteString(fmt.Sprintf("- **Bio**: %s\n", bio))
	}
	if repos, ok := u["public_repos"].(float64); ok {
		sb.WriteString(fmt.Sprintf("- **Repos**: %d\n", int(repos)))
	}
	if followers, ok := u["followers"].(float64); ok {
		sb.WriteString(fmt.Sprintf("- **Followers**: %d\n", int(followers)))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// reposToCSV: name,language,stars,fork,updated
func reposToCSV(jsonStr string) string {
	var repos []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &repos); err != nil {
		return jsonStr
	}
	if len(repos) == 0 {
		return "# 0 repos"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nname,language,stars,fork,updated\n")
	for _, r := range repos {
		updated := str(r, "updated_at")
		if len(updated) >= 10 {
			updated = updated[:10]
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%d,%v,%s\n",
			csvEscape(str(r, "name")),
			str(r, "language"),
			intVal(r, "stargazers_count"),
			boolVal(r, "fork"),
			updated,
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// starredReposToCSV: full_name,language,stars,description
func starredReposToCSV(jsonStr string) string {
	var repos []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &repos); err != nil {
		return jsonStr
	}
	if len(repos) == 0 {
		return "# 0 starred repos"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nfull_name,language,stars,description\n")
	for _, r := range repos {
		desc := str(r, "description")
		if len(desc) > 80 {
			desc = desc[:80] + "..."
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%d,%s\n",
			csvEscape(str(r, "full_name")),
			str(r, "language"),
			intVal(r, "stargazers_count"),
			csvEscape(desc),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// repoToCompact: single repo detail
func repoToCompact(jsonStr string) string {
	var r map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &r); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(r, "full_name")))
	if desc := str(r, "description"); desc != "" {
		sb.WriteString(fmt.Sprintf("- **Description**: %s\n", desc))
	}
	if lang := str(r, "language"); lang != "" {
		sb.WriteString(fmt.Sprintf("- **Language**: %s\n", lang))
	}
	sb.WriteString(fmt.Sprintf("- **Stars**: %d\n", intVal(r, "stargazers_count")))
	sb.WriteString(fmt.Sprintf("- **Forks**: %d\n", intVal(r, "forks_count")))
	sb.WriteString(fmt.Sprintf("- **Issues**: %d\n", intVal(r, "open_issues_count")))
	sb.WriteString(fmt.Sprintf("- **Default Branch**: %s\n", str(r, "default_branch")))
	if vis := str(r, "visibility"); vis != "" {
		sb.WriteString(fmt.Sprintf("- **Visibility**: %s\n", vis))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// branchesToCSV: name,protected
func branchesToCSV(jsonStr string) string {
	var branches []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &branches); err != nil {
		return jsonStr
	}
	if len(branches) == 0 {
		return "# 0 branches"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nname,protected\n")
	for _, b := range branches {
		sb.WriteString(fmt.Sprintf("%s,%v\n",
			csvEscape(str(b, "name")),
			boolVal(b, "protected"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// commitsToCSV: sha,author,date,message
func commitsToCSV(jsonStr string) string {
	var commits []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &commits); err != nil {
		return jsonStr
	}
	if len(commits) == 0 {
		return "# 0 commits"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nsha,author,date,message\n")
	for _, c := range commits {
		sha := str(c, "sha")
		if len(sha) > 7 {
			sha = sha[:7]
		}
		author := ""
		date := ""
		message := ""
		if cm, ok := c["commit"].(map[string]any); ok {
			message = str(cm, "message")
			if nl := strings.IndexByte(message, '\n'); nl > 0 {
				message = message[:nl]
			}
			if a, ok := cm["author"].(map[string]any); ok {
				author = str(a, "name")
				d := str(a, "date")
				if len(d) >= 10 {
					date = d[:10]
				}
			}
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			sha,
			csvEscape(author),
			date,
			csvEscape(message),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// issuesToCSV: number,title,state,author,labels,created
func issuesToCSV(jsonStr string) string {
	var issues []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &issues); err != nil {
		return jsonStr
	}
	if len(issues) == 0 {
		return "# 0 issues"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nnumber,title,state,author,labels,created\n")
	for _, i := range issues {
		author := ""
		if u, ok := i["user"].(map[string]any); ok {
			author = str(u, "login")
		}
		labels := labelsStr(i)
		created := str(i, "created_at")
		if len(created) >= 10 {
			created = created[:10]
		}
		sb.WriteString(fmt.Sprintf("%d,%s,%s,%s,%s,%s\n",
			intVal(i, "number"),
			csvEscape(str(i, "title")),
			str(i, "state"),
			csvEscape(author),
			csvEscape(labels),
			created,
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// issueToCompact: single issue
func issueToCompact(jsonStr string) string {
	var i map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &i); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# #%d: %s\n", intVal(i, "number"), str(i, "title")))
	sb.WriteString(fmt.Sprintf("- **State**: %s\n", str(i, "state")))
	if u, ok := i["user"].(map[string]any); ok {
		sb.WriteString(fmt.Sprintf("- **Author**: %s\n", str(u, "login")))
	}
	if labels := labelsStr(i); labels != "" {
		sb.WriteString(fmt.Sprintf("- **Labels**: %s\n", labels))
	}
	if created := str(i, "created_at"); len(created) >= 10 {
		sb.WriteString(fmt.Sprintf("- **Created**: %s\n", created[:10]))
	}
	if body := str(i, "body"); body != "" {
		if len(body) > 3000 {
			body = body[:3000] + "...(truncated)"
		}
		sb.WriteString(fmt.Sprintf("\n## Body\n%s\n", body))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// prsToCSV: number,title,state,author,draft,created
func prsToCSV(jsonStr string) string {
	var prs []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &prs); err != nil {
		return jsonStr
	}
	if len(prs) == 0 {
		return "# 0 PRs"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nnumber,title,state,author,draft,created\n")
	for _, p := range prs {
		author := ""
		if u, ok := p["user"].(map[string]any); ok {
			author = str(u, "login")
		}
		created := str(p, "created_at")
		if len(created) >= 10 {
			created = created[:10]
		}
		sb.WriteString(fmt.Sprintf("%d,%s,%s,%s,%v,%s\n",
			intVal(p, "number"),
			csvEscape(str(p, "title")),
			str(p, "state"),
			csvEscape(author),
			boolVal(p, "draft"),
			created,
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// prToCompact: single PR detail
func prToCompact(jsonStr string) string {
	var p map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# #%d: %s\n", intVal(p, "number"), str(p, "title")))
	sb.WriteString(fmt.Sprintf("- **State**: %s\n", str(p, "state")))
	if boolVal(p, "draft") {
		sb.WriteString("- **Draft**: true\n")
	}
	if boolVal(p, "merged") {
		sb.WriteString("- **Merged**: true\n")
	}
	if u, ok := p["user"].(map[string]any); ok {
		sb.WriteString(fmt.Sprintf("- **Author**: %s\n", str(u, "login")))
	}
	if head, ok := p["head"].(map[string]any); ok {
		sb.WriteString(fmt.Sprintf("- **Head**: %s\n", str(head, "ref")))
	}
	if base, ok := p["base"].(map[string]any); ok {
		sb.WriteString(fmt.Sprintf("- **Base**: %s\n", str(base, "ref")))
	}
	if created := str(p, "created_at"); len(created) >= 10 {
		sb.WriteString(fmt.Sprintf("- **Created**: %s\n", created[:10]))
	}
	if body := str(p, "body"); body != "" {
		if len(body) > 3000 {
			body = body[:3000] + "...(truncated)"
		}
		sb.WriteString(fmt.Sprintf("\n## Body\n%s\n", body))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// prFilesToCSV: filename,status,additions,deletions
func prFilesToCSV(jsonStr string) string {
	var files []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &files); err != nil {
		return jsonStr
	}
	if len(files) == 0 {
		return "# 0 files"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nfilename,status,additions,deletions\n")
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("%s,%s,%d,%d\n",
			csvEscape(str(f, "filename")),
			str(f, "status"),
			intVal(f, "additions"),
			intVal(f, "deletions"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// fileContentToCompact: file content
func fileContentToCompact(jsonStr string) string {
	var f map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &f); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(f, "path")))
	sb.WriteString(fmt.Sprintf("- **Size**: %d bytes\n", intVal(f, "size")))
	if enc := str(f, "encoding"); enc != "" {
		sb.WriteString(fmt.Sprintf("- **Encoding**: %s\n", enc))
	}
	if content := str(f, "content"); content != "" {
		if len(content) > 5000 {
			content = content[:5000] + "...(truncated)"
		}
		sb.WriteString(fmt.Sprintf("\n```\n%s\n```\n", content))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// searchReposToCSV: full_name,language,stars,description
func searchReposToCSV(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	total := intVal(wrapper, "total_count")
	items, ok := wrapper["items"].([]any)
	if !ok {
		return jsonStr
	}
	if len(items) == 0 {
		return fmt.Sprintf("# 0/%d repos", total)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("```csv  # %d/%d repos\nfull_name,language,stars,description\n", len(items), total))
	for _, raw := range items {
		r, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		desc := str(r, "description")
		if len(desc) > 80 {
			desc = desc[:80] + "..."
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%d,%s\n",
			csvEscape(str(r, "full_name")),
			str(r, "language"),
			intVal(r, "stargazers_count"),
			csvEscape(desc),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// searchCodeToCSV: path,repo,score
func searchCodeToCSV(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	total := intVal(wrapper, "total_count")
	items, ok := wrapper["items"].([]any)
	if !ok {
		return jsonStr
	}
	if len(items) == 0 {
		return fmt.Sprintf("# 0/%d code results", total)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("```csv  # %d/%d results\npath,repo,score\n", len(items), total))
	for _, raw := range items {
		r, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		repo := ""
		if rep, ok := r["repository"].(map[string]any); ok {
			repo = str(rep, "full_name")
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%.1f\n",
			csvEscape(str(r, "path")),
			csvEscape(repo),
			floatVal(r, "score"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// searchIssuesToCSV: number,title,state,repo,created
func searchIssuesToCSV(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	total := intVal(wrapper, "total_count")
	items, ok := wrapper["items"].([]any)
	if !ok {
		return jsonStr
	}
	if len(items) == 0 {
		return fmt.Sprintf("# 0/%d issues", total)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("```csv  # %d/%d issues\nnumber,title,state,repo,created\n", len(items), total))
	for _, raw := range items {
		i, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		repo := ""
		if rep, ok := i["repository"].(map[string]any); ok {
			repo = str(rep, "full_name")
		} else if repoURL := str(i, "repository_url"); repoURL != "" {
			// Extract owner/repo from URL
			parts := strings.Split(repoURL, "/")
			if len(parts) >= 2 {
				repo = parts[len(parts)-2] + "/" + parts[len(parts)-1]
			}
		}
		created := str(i, "created_at")
		if len(created) >= 10 {
			created = created[:10]
		}
		sb.WriteString(fmt.Sprintf("%d,%s,%s,%s,%s\n",
			intVal(i, "number"),
			csvEscape(str(i, "title")),
			str(i, "state"),
			csvEscape(repo),
			created,
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// workflowsToCSV: id,name,state,path
func workflowsToCSV(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	workflows, ok := wrapper["workflows"].([]any)
	if !ok {
		return jsonStr
	}
	if len(workflows) == 0 {
		return "# 0 workflows"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,state,path\n")
	for _, raw := range workflows {
		w, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("%d,%s,%s,%s\n",
			intVal(w, "id"),
			csvEscape(str(w, "name")),
			str(w, "state"),
			csvEscape(str(w, "path")),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// workflowRunsToCSV: id,name,status,conclusion,branch,created
func workflowRunsToCSV(jsonStr string) string {
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return jsonStr
	}
	runs, ok := wrapper["workflow_runs"].([]any)
	if !ok {
		return jsonStr
	}
	if len(runs) == 0 {
		return "# 0 workflow runs"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,status,conclusion,branch,created\n")
	for _, raw := range runs {
		r, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		created := str(r, "created_at")
		if len(created) >= 10 {
			created = created[:10]
		}
		sb.WriteString(fmt.Sprintf("%d,%s,%s,%s,%s,%s\n",
			intVal(r, "id"),
			csvEscape(str(r, "name")),
			str(r, "status"),
			str(r, "conclusion"),
			csvEscape(str(r, "head_branch")),
			created,
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// orgsToCSV: login,description
func orgsToCSV(jsonStr string) string {
	var orgs []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &orgs); err != nil {
		return jsonStr
	}
	if len(orgs) == 0 {
		return "# 0 orgs"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nlogin,description\n")
	for _, o := range orgs {
		sb.WriteString(fmt.Sprintf("%s,%s\n",
			csvEscape(str(o, "login")),
			csvEscape(str(o, "description")),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// eventsToCSV: type,repo,date
func eventsToCSV(jsonStr string) string {
	var events []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &events); err != nil {
		return jsonStr
	}
	if len(events) == 0 {
		return "# 0 events"
	}
	var sb strings.Builder
	sb.WriteString("```csv\ntype,repo,date\n")
	for _, e := range events {
		repo := ""
		if r, ok := e["repo"].(map[string]any); ok {
			repo = str(r, "name")
		}
		created := str(e, "created_at")
		if len(created) >= 16 {
			created = created[:10] + " " + created[11:16]
		}
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			str(e, "type"),
			csvEscape(repo),
			created,
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

func floatVal(obj map[string]any, key string) float64 {
	if v, ok := obj[key].(float64); ok {
		return v
	}
	return 0
}

func boolVal(obj map[string]any, key string) bool {
	if v, ok := obj[key].(bool); ok {
		return v
	}
	return false
}

func labelsStr(obj map[string]any) string {
	labels, ok := obj["labels"].([]any)
	if !ok || len(labels) == 0 {
		return ""
	}
	names := make([]string, 0, len(labels))
	for _, l := range labels {
		if lm, ok := l.(map[string]any); ok {
			if name := str(lm, "name"); name != "" {
				names = append(names, name)
			}
		}
	}
	return strings.Join(names, ";")
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
