package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/store"
)

const (
	githubAPIBase    = "https://api.github.com"
	githubAPIVersion = "2022-11-28"
)

var client = httpclient.New()

// GitHubModule implements the Module interface for GitHub API
type GitHubModule struct{}

// New creates a new GitHubModule instance
func New() *GitHubModule {
	return &GitHubModule{}
}

// Name returns the module name
func (m *GitHubModule) Name() string {
	return "github"
}

// Description returns the module description
func (m *GitHubModule) Description() string {
	return "GitHub API - リポジトリ、Issue、PR、Actions、検索"
}

// APIVersion returns the GitHub API version
func (m *GitHubModule) APIVersion() string {
	return githubAPIVersion
}

// Tools returns all available tools
func (m *GitHubModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *GitHubModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for GitHub)
func (m *GitHubModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *GitHubModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// Prompts returns all available prompts (none for GitHub)
func (m *GitHubModule) Prompts() []modules.Prompt {
	return nil
}

// GetPrompt generates a prompt with arguments (not implemented)
func (m *GitHubModule) GetPrompt(ctx context.Context, name string, args map[string]any) (string, error) {
	return "", fmt.Errorf("prompts not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *store.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil
	}
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "github")
	if err != nil {
		return nil
	}
	return credentials
}

func headers(ctx context.Context) map[string]string {
	creds := getCredentials(ctx)
	if creds == nil {
		return map[string]string{}
	}

	h := map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": githubAPIVersion,
	}

	// Both oauth2 and api_key use Bearer token
	switch creds.AuthType {
	case store.AuthTypeOAuth2, store.AuthTypeAPIKey:
		h["Authorization"] = "Bearer " + creds.AccessToken
	}

	return h
}

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	// User
	{
		Name:        "get_user",
		Description: "Get information about the authenticated GitHub user.",
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	// Repositories
	{
		Name:        "list_repos",
		Description: "List repositories for the authenticated user.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"type":     {Type: "string", Description: "Type of repositories (all, owner, public, private). Default: owner"},
				"sort":     {Type: "string", Description: "Sort by (created, updated, pushed, full_name). Default: updated"},
				"per_page": {Type: "number", Description: "Results per page. Default: 30"},
				"page":     {Type: "number", Description: "Page number. Default: 1"},
			},
		},
	},
	{
		Name:        "get_repo",
		Description: "Get details of a specific repository.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner": {Type: "string", Description: "Repository owner"},
				"repo":  {Type: "string", Description: "Repository name"},
			},
			Required: []string{"owner", "repo"},
		},
	},
	{
		Name:        "list_branches",
		Description: "List branches in a repository.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner":    {Type: "string", Description: "Repository owner"},
				"repo":     {Type: "string", Description: "Repository name"},
				"per_page": {Type: "number", Description: "Results per page. Default: 30"},
			},
			Required: []string{"owner", "repo"},
		},
	},
	{
		Name:        "list_commits",
		Description: "List commits in a repository.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner":    {Type: "string", Description: "Repository owner"},
				"repo":     {Type: "string", Description: "Repository name"},
				"sha":      {Type: "string", Description: "Branch name or commit SHA to filter by"},
				"per_page": {Type: "number", Description: "Results per page. Default: 30"},
				"page":     {Type: "number", Description: "Page number. Default: 1"},
			},
			Required: []string{"owner", "repo"},
		},
	},
	{
		Name:        "get_file_content",
		Description: "Get the content of a file in a repository.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner": {Type: "string", Description: "Repository owner"},
				"repo":  {Type: "string", Description: "Repository name"},
				"path":  {Type: "string", Description: "File path"},
				"ref":   {Type: "string", Description: "Branch name or commit SHA"},
			},
			Required: []string{"owner", "repo", "path"},
		},
	},
	// Issues
	{
		Name:        "list_issues",
		Description: "List issues in a repository.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner":    {Type: "string", Description: "Repository owner"},
				"repo":     {Type: "string", Description: "Repository name"},
				"state":    {Type: "string", Description: "Issue state (open, closed, all). Default: open"},
				"per_page": {Type: "number", Description: "Results per page. Default: 30"},
				"page":     {Type: "number", Description: "Page number. Default: 1"},
			},
			Required: []string{"owner", "repo"},
		},
	},
	{
		Name:        "get_issue",
		Description: "Get details of a specific issue.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner":        {Type: "string", Description: "Repository owner"},
				"repo":         {Type: "string", Description: "Repository name"},
				"issue_number": {Type: "number", Description: "Issue number"},
			},
			Required: []string{"owner", "repo", "issue_number"},
		},
	},
	{
		Name:        "create_issue",
		Description: "Create a new issue in a repository.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner":     {Type: "string", Description: "Repository owner"},
				"repo":      {Type: "string", Description: "Repository name"},
				"title":     {Type: "string", Description: "Issue title"},
				"body":      {Type: "string", Description: "Issue body"},
				"labels":    {Type: "array", Description: "Labels to assign"},
				"assignees": {Type: "array", Description: "Users to assign"},
			},
			Required: []string{"owner", "repo", "title"},
		},
	},
	{
		Name:        "update_issue",
		Description: "Update an existing issue.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner":        {Type: "string", Description: "Repository owner"},
				"repo":         {Type: "string", Description: "Repository name"},
				"issue_number": {Type: "number", Description: "Issue number"},
				"title":        {Type: "string", Description: "New title"},
				"body":         {Type: "string", Description: "New body"},
				"state":        {Type: "string", Description: "New state (open, closed)"},
				"labels":       {Type: "array", Description: "Labels to set"},
				"assignees":    {Type: "array", Description: "Users to assign"},
			},
			Required: []string{"owner", "repo", "issue_number"},
		},
	},
	{
		Name:        "add_issue_comment",
		Description: "Add a comment to an issue.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner":        {Type: "string", Description: "Repository owner"},
				"repo":         {Type: "string", Description: "Repository name"},
				"issue_number": {Type: "number", Description: "Issue number"},
				"body":         {Type: "string", Description: "Comment body"},
			},
			Required: []string{"owner", "repo", "issue_number", "body"},
		},
	},
	// Pull Requests
	{
		Name:        "list_prs",
		Description: "List pull requests in a repository.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner":    {Type: "string", Description: "Repository owner"},
				"repo":     {Type: "string", Description: "Repository name"},
				"state":    {Type: "string", Description: "PR state (open, closed, all). Default: open"},
				"per_page": {Type: "number", Description: "Results per page. Default: 30"},
				"page":     {Type: "number", Description: "Page number. Default: 1"},
			},
			Required: []string{"owner", "repo"},
		},
	},
	{
		Name:        "get_pr",
		Description: "Get details of a specific pull request.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner":     {Type: "string", Description: "Repository owner"},
				"repo":      {Type: "string", Description: "Repository name"},
				"pr_number": {Type: "number", Description: "PR number"},
			},
			Required: []string{"owner", "repo", "pr_number"},
		},
	},
	{
		Name:        "create_pr",
		Description: "Create a new pull request.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner": {Type: "string", Description: "Repository owner"},
				"repo":  {Type: "string", Description: "Repository name"},
				"title": {Type: "string", Description: "PR title"},
				"head":  {Type: "string", Description: "Branch with changes"},
				"base":  {Type: "string", Description: "Branch to merge into"},
				"body":  {Type: "string", Description: "PR description"},
				"draft": {Type: "boolean", Description: "Create as draft PR"},
			},
			Required: []string{"owner", "repo", "title", "head", "base"},
		},
	},
	{
		Name:        "list_pr_files",
		Description: "List files changed in a pull request.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner":     {Type: "string", Description: "Repository owner"},
				"repo":      {Type: "string", Description: "Repository name"},
				"pr_number": {Type: "number", Description: "PR number"},
				"per_page":  {Type: "number", Description: "Results per page. Default: 30"},
			},
			Required: []string{"owner", "repo", "pr_number"},
		},
	},
	// Search
	{
		Name:        "search_repos",
		Description: "Search for repositories.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"query":    {Type: "string", Description: "Search query"},
				"sort":     {Type: "string", Description: "Sort by (stars, forks, help-wanted-issues, updated)"},
				"per_page": {Type: "number", Description: "Results per page. Default: 30"},
				"page":     {Type: "number", Description: "Page number. Default: 1"},
			},
			Required: []string{"query"},
		},
	},
	{
		Name:        "search_code",
		Description: "Search for code across repositories.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"query":    {Type: "string", Description: "Search query (e.g., 'addClass in:file language:js repo:jquery/jquery')"},
				"per_page": {Type: "number", Description: "Results per page. Default: 30"},
				"page":     {Type: "number", Description: "Page number. Default: 1"},
			},
			Required: []string{"query"},
		},
	},
	{
		Name:        "search_issues",
		Description: "Search for issues and pull requests.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"query":    {Type: "string", Description: "Search query (e.g., 'repo:owner/repo is:open is:issue')"},
				"sort":     {Type: "string", Description: "Sort by (comments, reactions, created, updated)"},
				"per_page": {Type: "number", Description: "Results per page. Default: 30"},
				"page":     {Type: "number", Description: "Page number. Default: 1"},
			},
			Required: []string{"query"},
		},
	},
	// Actions
	{
		Name:        "list_workflows",
		Description: "List workflows in a repository.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner":    {Type: "string", Description: "Repository owner"},
				"repo":     {Type: "string", Description: "Repository name"},
				"per_page": {Type: "number", Description: "Results per page. Default: 30"},
			},
			Required: []string{"owner", "repo"},
		},
	},
	{
		Name:        "list_workflow_runs",
		Description: "List workflow runs in a repository.",
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner":       {Type: "string", Description: "Repository owner"},
				"repo":        {Type: "string", Description: "Repository name"},
				"workflow_id": {Type: "string", Description: "Workflow ID or file name to filter by"},
				"status":      {Type: "string", Description: "Filter by status (queued, in_progress, completed)"},
				"per_page":    {Type: "number", Description: "Results per page. Default: 30"},
			},
			Required: []string{"owner", "repo"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"get_user":           getUser,
	"list_repos":         listRepos,
	"get_repo":           getRepo,
	"list_branches":      listBranches,
	"list_commits":       listCommits,
	"get_file_content":   getFileContent,
	"list_issues":        listIssues,
	"get_issue":          getIssue,
	"create_issue":       createIssue,
	"update_issue":       updateIssue,
	"add_issue_comment":  addIssueComment,
	"list_prs":           listPRs,
	"get_pr":             getPR,
	"create_pr":          createPR,
	"list_pr_files":      listPRFiles,
	"search_repos":       searchRepos,
	"search_code":        searchCode,
	"search_issues":      searchIssues,
	"list_workflows":     listWorkflows,
	"list_workflow_runs": listWorkflowRuns,
}

// =============================================================================
// User
// =============================================================================

func getUser(ctx context.Context, params map[string]any) (string, error) {
	endpoint := githubAPIBase + "/user"
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Repositories
// =============================================================================

func listRepos(ctx context.Context, params map[string]any) (string, error) {
	query := url.Values{}
	if t, ok := params["type"].(string); ok && t != "" {
		query.Set("type", t)
	} else {
		query.Set("type", "owner")
	}
	if sort, ok := params["sort"].(string); ok && sort != "" {
		query.Set("sort", sort)
	} else {
		query.Set("sort", "updated")
	}
	perPage := 30
	if pp, ok := params["per_page"].(float64); ok {
		perPage = int(pp)
	}
	query.Set("per_page", fmt.Sprintf("%d", perPage))
	page := 1
	if p, ok := params["page"].(float64); ok {
		page = int(p)
	}
	query.Set("page", fmt.Sprintf("%d", page))

	endpoint := fmt.Sprintf("%s/user/repos?%s", githubAPIBase, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getRepo(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	endpoint := fmt.Sprintf("%s/repos/%s/%s", githubAPIBase, owner, repo)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func listBranches(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	perPage := 30
	if pp, ok := params["per_page"].(float64); ok {
		perPage = int(pp)
	}
	endpoint := fmt.Sprintf("%s/repos/%s/%s/branches?per_page=%d", githubAPIBase, owner, repo, perPage)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func listCommits(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	query := url.Values{}
	if sha, ok := params["sha"].(string); ok && sha != "" {
		query.Set("sha", sha)
	}
	perPage := 30
	if pp, ok := params["per_page"].(float64); ok {
		perPage = int(pp)
	}
	query.Set("per_page", fmt.Sprintf("%d", perPage))
	page := 1
	if p, ok := params["page"].(float64); ok {
		page = int(p)
	}
	query.Set("page", fmt.Sprintf("%d", page))

	endpoint := fmt.Sprintf("%s/repos/%s/%s/commits?%s", githubAPIBase, owner, repo, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getFileContent(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	path, _ := params["path"].(string)

	endpoint := fmt.Sprintf("%s/repos/%s/%s/contents/%s", githubAPIBase, owner, repo, path)
	if ref, ok := params["ref"].(string); ok && ref != "" {
		endpoint += "?ref=" + url.QueryEscape(ref)
	}

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	// Try to decode base64 content
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err == nil {
		if content, ok := result["content"].(string); ok {
			if encoding, ok := result["encoding"].(string); ok && encoding == "base64" {
				decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
				if err == nil {
					result["content"] = string(decoded)
					result["encoding"] = "utf-8"
				}
			}
		}
		return httpclient.PrettyJSONFromInterface(result), nil
	}

	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Issues
// =============================================================================

func listIssues(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	query := url.Values{}
	if state, ok := params["state"].(string); ok && state != "" {
		query.Set("state", state)
	} else {
		query.Set("state", "open")
	}
	perPage := 30
	if pp, ok := params["per_page"].(float64); ok {
		perPage = int(pp)
	}
	query.Set("per_page", fmt.Sprintf("%d", perPage))
	page := 1
	if p, ok := params["page"].(float64); ok {
		page = int(p)
	}
	query.Set("page", fmt.Sprintf("%d", page))

	endpoint := fmt.Sprintf("%s/repos/%s/%s/issues?%s", githubAPIBase, owner, repo, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getIssue(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	issueNumber, _ := params["issue_number"].(float64)
	endpoint := fmt.Sprintf("%s/repos/%s/%s/issues/%d", githubAPIBase, owner, repo, int(issueNumber))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createIssue(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	title, _ := params["title"].(string)

	body := map[string]interface{}{"title": title}
	if b, ok := params["body"].(string); ok {
		body["body"] = b
	}
	if labels, ok := params["labels"].([]interface{}); ok {
		body["labels"] = labels
	}
	if assignees, ok := params["assignees"].([]interface{}); ok {
		body["assignees"] = assignees
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s/issues", githubAPIBase, owner, repo)
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updateIssue(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	issueNumber, _ := params["issue_number"].(float64)

	body := make(map[string]interface{})
	if title, ok := params["title"].(string); ok {
		body["title"] = title
	}
	if b, ok := params["body"].(string); ok {
		body["body"] = b
	}
	if state, ok := params["state"].(string); ok {
		body["state"] = state
	}
	if labels, ok := params["labels"].([]interface{}); ok {
		body["labels"] = labels
	}
	if assignees, ok := params["assignees"].([]interface{}); ok {
		body["assignees"] = assignees
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s/issues/%d", githubAPIBase, owner, repo, int(issueNumber))
	respBody, err := client.DoJSON("PATCH", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func addIssueComment(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	issueNumber, _ := params["issue_number"].(float64)
	body, _ := params["body"].(string)

	endpoint := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", githubAPIBase, owner, repo, int(issueNumber))
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), map[string]string{"body": body})
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Pull Requests
// =============================================================================

func listPRs(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	query := url.Values{}
	if state, ok := params["state"].(string); ok && state != "" {
		query.Set("state", state)
	} else {
		query.Set("state", "open")
	}
	perPage := 30
	if pp, ok := params["per_page"].(float64); ok {
		perPage = int(pp)
	}
	query.Set("per_page", fmt.Sprintf("%d", perPage))
	page := 1
	if p, ok := params["page"].(float64); ok {
		page = int(p)
	}
	query.Set("page", fmt.Sprintf("%d", page))

	endpoint := fmt.Sprintf("%s/repos/%s/%s/pulls?%s", githubAPIBase, owner, repo, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getPR(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	prNumber, _ := params["pr_number"].(float64)
	endpoint := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", githubAPIBase, owner, repo, int(prNumber))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createPR(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	title, _ := params["title"].(string)
	head, _ := params["head"].(string)
	base, _ := params["base"].(string)

	body := map[string]interface{}{
		"title": title,
		"head":  head,
		"base":  base,
	}
	if b, ok := params["body"].(string); ok {
		body["body"] = b
	}
	if draft, ok := params["draft"].(bool); ok {
		body["draft"] = draft
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s/pulls", githubAPIBase, owner, repo)
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func listPRFiles(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	prNumber, _ := params["pr_number"].(float64)
	perPage := 30
	if pp, ok := params["per_page"].(float64); ok {
		perPage = int(pp)
	}
	endpoint := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/files?per_page=%d", githubAPIBase, owner, repo, int(prNumber), perPage)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Search
// =============================================================================

func searchRepos(ctx context.Context, params map[string]any) (string, error) {
	query, _ := params["query"].(string)
	q := url.Values{}
	q.Set("q", query)
	if sort, ok := params["sort"].(string); ok && sort != "" {
		q.Set("sort", sort)
	}
	perPage := 30
	if pp, ok := params["per_page"].(float64); ok {
		perPage = int(pp)
	}
	q.Set("per_page", fmt.Sprintf("%d", perPage))
	page := 1
	if p, ok := params["page"].(float64); ok {
		page = int(p)
	}
	q.Set("page", fmt.Sprintf("%d", page))

	endpoint := fmt.Sprintf("%s/search/repositories?%s", githubAPIBase, q.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func searchCode(ctx context.Context, params map[string]any) (string, error) {
	query, _ := params["query"].(string)
	q := url.Values{}
	q.Set("q", query)
	perPage := 30
	if pp, ok := params["per_page"].(float64); ok {
		perPage = int(pp)
	}
	q.Set("per_page", fmt.Sprintf("%d", perPage))
	page := 1
	if p, ok := params["page"].(float64); ok {
		page = int(p)
	}
	q.Set("page", fmt.Sprintf("%d", page))

	endpoint := fmt.Sprintf("%s/search/code?%s", githubAPIBase, q.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func searchIssues(ctx context.Context, params map[string]any) (string, error) {
	query, _ := params["query"].(string)
	q := url.Values{}
	q.Set("q", query)
	if sort, ok := params["sort"].(string); ok && sort != "" {
		q.Set("sort", sort)
	}
	perPage := 30
	if pp, ok := params["per_page"].(float64); ok {
		perPage = int(pp)
	}
	q.Set("per_page", fmt.Sprintf("%d", perPage))
	page := 1
	if p, ok := params["page"].(float64); ok {
		page = int(p)
	}
	q.Set("page", fmt.Sprintf("%d", page))

	endpoint := fmt.Sprintf("%s/search/issues?%s", githubAPIBase, q.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Actions
// =============================================================================

func listWorkflows(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	perPage := 30
	if pp, ok := params["per_page"].(float64); ok {
		perPage = int(pp)
	}
	endpoint := fmt.Sprintf("%s/repos/%s/%s/actions/workflows?per_page=%d", githubAPIBase, owner, repo, perPage)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func listWorkflowRuns(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	query := url.Values{}
	if status, ok := params["status"].(string); ok && status != "" {
		query.Set("status", status)
	}
	perPage := 30
	if pp, ok := params["per_page"].(float64); ok {
		perPage = int(pp)
	}
	query.Set("per_page", fmt.Sprintf("%d", perPage))

	var endpoint string
	if workflowID, ok := params["workflow_id"].(string); ok && workflowID != "" {
		endpoint = fmt.Sprintf("%s/repos/%s/%s/actions/workflows/%s/runs?%s", githubAPIBase, owner, repo, workflowID, query.Encode())
	} else {
		endpoint = fmt.Sprintf("%s/repos/%s/%s/actions/runs?%s", githubAPIBase, owner, repo, query.Encode())
	}

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}
