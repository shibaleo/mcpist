package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/broker"
	"mcpist/server/pkg/githubapi"
	gen "mcpist/server/pkg/githubapi/gen"
)

const githubAPIVersion = "2022-11-28"

// GitHubModule implements the Module interface for GitHub API
type GitHubModule struct{}

// New creates a new GitHubModule instance
func New() *GitHubModule {
	return &GitHubModule{}
}

// Module descriptions
var moduleDescriptions = modules.LocalizedText{
	"en-US": "GitHub API - Repository, Issue, PR, Actions, and Search operations",
	"ja-JP": "GitHub API - リポジトリ、Issue、PR、Actions、検索操作",
}

// Name returns the module name
func (m *GitHubModule) Name() string {
	return "github"
}

// Descriptions returns the module descriptions in all languages
func (m *GitHubModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description (English)
func (m *GitHubModule) Description() string {
	return moduleDescriptions["en-US"]
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

// ToCompact converts JSON result to compact format (MD or CSV)
// Implements modules.CompactConverter interface
func (m *GitHubModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// Resources returns all available resources (none for GitHub)
func (m *GitHubModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *GitHubModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "github")
	if err != nil {
		return nil
	}
	return credentials
}

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	// User
	{
		ID:   "github:get_user",
		Name: "get_user",
		Descriptions: modules.LocalizedText{
			"en-US": "Get a GitHub user's profile by username.",
			"ja-JP": "GitHubユーザーのプロフィールをユーザー名で取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"username": {Type: "string", Description: "GitHub username"},
			},
			Required: []string{"username"},
		},
	},
	// Repositories
	{
		ID:   "github:list_repos",
		Name: "list_repos",
		Descriptions: modules.LocalizedText{
			"en-US": "List repositories for a user.",
			"ja-JP": "ユーザーのリポジトリを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"username": {Type: "string", Description: "GitHub username"},
				"type":     {Type: "string", Description: "Type of repositories (all, owner, member). Default: owner"},
				"sort":     {Type: "string", Description: "Sort by (created, updated, pushed, full_name). Default: updated"},
				"per_page": {Type: "number", Description: "Results per page. Default: 30"},
				"page":     {Type: "number", Description: "Page number. Default: 1"},
			},
			Required: []string{"username"},
		},
	},
	{
		ID:   "github:list_starred_repos",
		Name: "list_starred_repos",
		Descriptions: modules.LocalizedText{
			"en-US": "List repositories starred by a user.",
			"ja-JP": "ユーザーがスターしたリポジトリを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"username":  {Type: "string", Description: "GitHub username"},
				"sort":      {Type: "string", Description: "Sort by (created, updated). Default: created"},
				"direction": {Type: "string", Description: "Sort direction (asc, desc). Default: desc"},
				"per_page":  {Type: "number", Description: "Results per page. Default: 30"},
				"page":      {Type: "number", Description: "Page number. Default: 1"},
			},
			Required: []string{"username"},
		},
	},
	{
		ID:   "github:get_repo",
		Name: "get_repo",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific repository.",
			"ja-JP": "特定のリポジトリの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:list_branches",
		Name: "list_branches",
		Descriptions: modules.LocalizedText{
			"en-US": "List branches in a repository.",
			"ja-JP": "リポジトリ内のブランチを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:list_commits",
		Name: "list_commits",
		Descriptions: modules.LocalizedText{
			"en-US": "List commits in a repository.",
			"ja-JP": "リポジトリ内のコミットを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:get_file_content",
		Name: "get_file_content",
		Descriptions: modules.LocalizedText{
			"en-US": "Get the content of a file in a repository.",
			"ja-JP": "リポジトリ内のファイルの内容を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:list_issues",
		Name: "list_issues",
		Descriptions: modules.LocalizedText{
			"en-US": "List issues in a repository.",
			"ja-JP": "リポジトリ内のIssueを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:get_issue",
		Name: "get_issue",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific issue.",
			"ja-JP": "特定のIssueの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:create_issue",
		Name: "create_issue",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new issue in a repository.",
			"ja-JP": "リポジトリに新しいIssueを作成します。",
		},
		Annotations: modules.AnnotateCreate,
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
		ID:   "github:update_issue",
		Name: "update_issue",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing issue.",
			"ja-JP": "既存のIssueを更新します。",
		},
		Annotations: modules.AnnotateUpdate,
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
		ID:   "github:add_issue_comment",
		Name: "add_issue_comment",
		Descriptions: modules.LocalizedText{
			"en-US": "Add a comment to an issue.",
			"ja-JP": "Issueにコメントを追加します。",
		},
		Annotations: modules.AnnotateCreate,
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
		ID:   "github:list_prs",
		Name: "list_prs",
		Descriptions: modules.LocalizedText{
			"en-US": "List pull requests in a repository.",
			"ja-JP": "リポジトリ内のプルリクエストを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:get_pr",
		Name: "get_pr",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific pull request.",
			"ja-JP": "特定のプルリクエストの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:create_pr",
		Name: "create_pr",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new pull request.",
			"ja-JP": "新しいプルリクエストを作成します。",
		},
		Annotations: modules.AnnotateCreate,
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
		ID:   "github:list_pr_files",
		Name: "list_pr_files",
		Descriptions: modules.LocalizedText{
			"en-US": "List files changed in a pull request.",
			"ja-JP": "プルリクエストで変更されたファイルを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:search_repos",
		Name: "search_repos",
		Descriptions: modules.LocalizedText{
			"en-US": "Search for repositories.",
			"ja-JP": "リポジトリを検索します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:search_code",
		Name: "search_code",
		Descriptions: modules.LocalizedText{
			"en-US": "Search for code across repositories.",
			"ja-JP": "リポジトリ全体でコードを検索します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:search_issues",
		Name: "search_issues",
		Descriptions: modules.LocalizedText{
			"en-US": "Search for issues and pull requests.",
			"ja-JP": "Issueとプルリクエストを検索します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:list_workflows",
		Name: "list_workflows",
		Descriptions: modules.LocalizedText{
			"en-US": "List workflows in a repository.",
			"ja-JP": "リポジトリ内のワークフローを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:list_workflow_runs",
		Name: "list_workflow_runs",
		Descriptions: modules.LocalizedText{
			"en-US": "List workflow runs in a repository.",
			"ja-JP": "リポジトリ内のワークフロー実行を一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
	// User (continued)
	{
		ID:   "github:list_orgs",
		Name: "list_orgs",
		Descriptions: modules.LocalizedText{
			"en-US": "List organizations for a user.",
			"ja-JP": "ユーザーの所属組織を一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"username": {Type: "string", Description: "GitHub username"},
				"per_page": {Type: "number", Description: "Results per page. Default: 30"},
			},
			Required: []string{"username"},
		},
	},
	{
		ID:   "github:list_public_events",
		Name: "list_public_events",
		Descriptions: modules.LocalizedText{
			"en-US": "List recent public events for a user.",
			"ja-JP": "ユーザーの最近の公開イベントを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"username": {Type: "string", Description: "GitHub username"},
				"per_page": {Type: "number", Description: "Results per page. Default: 30"},
				"page":     {Type: "number", Description: "Page number. Default: 1"},
			},
			Required: []string{"username"},
		},
	},
	// Composite
	{
		ID:   "github:describe_user",
		Name: "describe_user",
		Descriptions: modules.LocalizedText{
			"en-US": "Comprehensive GitHub user analysis. Fetches profile, repositories, starred repos, organizations, and recent activity in parallel.",
			"ja-JP": "GitHubユーザーの総合分析。プロフィール、リポジトリ、スター、所属組織、最近のアクティビティを並列取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"username": {Type: "string", Description: "GitHub username to analyze"},
			},
			Required: []string{"username"},
		},
	},
	{
		ID:   "github:describe_repo",
		Name: "describe_repo",
		Descriptions: modules.LocalizedText{
			"en-US": "Comprehensive repository analysis. Fetches repo info, topics, README, branches, open issues, and open PRs in parallel.",
			"ja-JP": "リポジトリの総合分析。リポジトリ情報、トピック、README、ブランチ、オープンIssue、オープンPRを並列取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
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
		ID:   "github:describe_pr",
		Name: "describe_pr",
		Descriptions: modules.LocalizedText{
			"en-US": "Comprehensive pull request analysis. Fetches PR details, changed files, reviews, and comments in parallel.",
			"ja-JP": "プルリクエストの総合分析。PR詳細、変更ファイル、レビュー、コメントを並列取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"owner":       {Type: "string", Description: "Repository owner"},
				"repo":        {Type: "string", Description: "Repository name"},
				"pr_number": {Type: "number", Description: "Pull request number"},
			},
			Required: []string{"owner", "repo", "pr_number"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"get_user":            getUser,
	"list_repos":          listRepos,
	"list_starred_repos":  listStarredRepos,
	"get_repo":            getRepo,
	"list_branches":       listBranches,
	"list_commits":        listCommits,
	"get_file_content":    getFileContent,
	"list_issues":         listIssues,
	"get_issue":           getIssue,
	"create_issue":        createIssue,
	"update_issue":        updateIssue,
	"add_issue_comment":   addIssueComment,
	"list_prs":            listPRs,
	"get_pr":              getPR,
	"create_pr":           createPR,
	"list_pr_files":       listPRFiles,
	"search_repos":        searchRepos,
	"search_code":         searchCode,
	"search_issues":       searchIssues,
	"list_workflows":      listWorkflows,
	"list_workflow_runs":  listWorkflowRuns,
	"list_orgs":           listOrgs,
	"list_public_events":  listPublicEvents,
	"describe_user":       describeUser,
	"describe_repo":       describeRepo,
	"describe_pr":         describePR,
}

// =============================================================================
// ogen client helper
// =============================================================================

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return githubapi.NewClient(creds.AccessToken)
}

var toJSON = modules.ToJSON

var toStringSlice = modules.ToStringSlice

// =============================================================================
// User
// =============================================================================

func getUser(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	username, _ := params["username"].(string)
	res, err := c.UsersGetByName(ctx, gen.UsersGetByNameParams{Username: username})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func listOrgs(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	username, _ := params["username"].(string)
	p := gen.OrgsListForUserParams{Username: username}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	res, err := c.OrgsListForUser(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func listPublicEvents(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	username, _ := params["username"].(string)
	p := gen.ActivityListPublicEventsForUserParams{Username: username}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	if pg, ok := params["page"].(float64); ok {
		p.Page.SetTo(int(pg))
	}
	res, err := c.ActivityListPublicEventsForUser(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Repositories
// =============================================================================

func listRepos(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	username, _ := params["username"].(string)
	p := gen.ReposListForUserParams{Username: username}
	if t, ok := params["type"].(string); ok && t != "" {
		p.Type.SetTo(gen.ReposListForUserType(t))
	}
	if sort, ok := params["sort"].(string); ok && sort != "" {
		p.Sort.SetTo(gen.ReposListForUserSort(sort))
	}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	if pg, ok := params["page"].(float64); ok {
		p.Page.SetTo(int(pg))
	}
	res, err := c.ReposListForUser(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func listStarredRepos(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	username, _ := params["username"].(string)
	p := gen.ActivityListReposStarredByUserParams{Username: username}
	if sort, ok := params["sort"].(string); ok && sort != "" {
		p.Sort.SetTo(gen.ActivityListReposStarredByUserSort(sort))
	}
	if dir, ok := params["direction"].(string); ok && dir != "" {
		p.Direction.SetTo(gen.ActivityListReposStarredByUserDirection(dir))
	}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	if pg, ok := params["page"].(float64); ok {
		p.Page.SetTo(int(pg))
	}
	res, err := c.ActivityListReposStarredByUser(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getRepo(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)

	res, err := c.ReposGet(ctx, gen.ReposGetParams{Owner: owner, Repo: repo})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func listBranches(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	p := gen.ReposListBranchesParams{Owner: owner, Repo: repo}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	res, err := c.ReposListBranches(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func listCommits(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	p := gen.ReposListCommitsParams{Owner: owner, Repo: repo}
	if sha, ok := params["sha"].(string); ok && sha != "" {
		p.Sha.SetTo(sha)
	}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	if pg, ok := params["page"].(float64); ok {
		p.Page.SetTo(int(pg))
	}
	res, err := c.ReposListCommits(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getFileContent(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	path, _ := params["path"].(string)
	p := gen.ReposGetContentParams{Owner: owner, Repo: repo, Path: path}
	if ref, ok := params["ref"].(string); ok && ref != "" {
		p.Ref.SetTo(ref)
	}
	res, err := c.ReposGetContent(ctx, p)
	if err != nil {
		return "", err
	}
	// Decode base64 content inline
	if enc, ok := res.Encoding.Get(); ok && enc == "base64" {
		if content, ok := res.Content.Get(); ok {
			decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
			if err == nil {
				res.Content.SetTo(string(decoded))
				res.Encoding.SetTo("utf-8")
			}
		}
	}
	return toJSON(res)
}

// =============================================================================
// Issues
// =============================================================================

func listIssues(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	p := gen.IssuesListForRepoParams{Owner: owner, Repo: repo}
	if state, ok := params["state"].(string); ok && state != "" {
		p.State.SetTo(gen.IssuesListForRepoState(state))
	}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	if pg, ok := params["page"].(float64); ok {
		p.Page.SetTo(int(pg))
	}
	res, err := c.IssuesListForRepo(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getIssue(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	issueNumber, _ := params["issue_number"].(float64)
	res, err := c.IssuesGet(ctx, gen.IssuesGetParams{Owner: owner, Repo: repo, IssueNumber: int(issueNumber)})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func createIssue(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	title, _ := params["title"].(string)
	req := &gen.CreateIssueRequest{Title: title}
	if b, ok := params["body"].(string); ok {
		req.Body.SetTo(b)
	}
	if labels, ok := params["labels"].([]interface{}); ok {
		req.Labels = toStringSlice(labels)
	}
	if assignees, ok := params["assignees"].([]interface{}); ok {
		req.Assignees = toStringSlice(assignees)
	}
	res, err := c.IssuesCreate(ctx, req, gen.IssuesCreateParams{Owner: owner, Repo: repo})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func updateIssue(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	issueNumber, _ := params["issue_number"].(float64)
	req := &gen.UpdateIssueRequest{}
	if title, ok := params["title"].(string); ok {
		req.Title.SetTo(title)
	}
	if b, ok := params["body"].(string); ok {
		req.Body.SetTo(b)
	}
	if state, ok := params["state"].(string); ok {
		req.State.SetTo(state)
	}
	if labels, ok := params["labels"].([]interface{}); ok {
		req.Labels = toStringSlice(labels)
	}
	if assignees, ok := params["assignees"].([]interface{}); ok {
		req.Assignees = toStringSlice(assignees)
	}
	res, err := c.IssuesUpdate(ctx, req, gen.IssuesUpdateParams{Owner: owner, Repo: repo, IssueNumber: int(issueNumber)})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func addIssueComment(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	issueNumber, _ := params["issue_number"].(float64)
	body, _ := params["body"].(string)
	res, err := c.IssuesCreateComment(ctx, &gen.CreateCommentRequest{Body: body}, gen.IssuesCreateCommentParams{Owner: owner, Repo: repo, IssueNumber: int(issueNumber)})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Pull Requests
// =============================================================================

func listPRs(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	p := gen.PullsListForRepoParams{Owner: owner, Repo: repo}
	if state, ok := params["state"].(string); ok && state != "" {
		p.State.SetTo(gen.PullsListForRepoState(state))
	}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	if pg, ok := params["page"].(float64); ok {
		p.Page.SetTo(int(pg))
	}
	res, err := c.PullsListForRepo(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getPR(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	prNumber, _ := params["pr_number"].(float64)
	res, err := c.PullsGet(ctx, gen.PullsGetParams{Owner: owner, Repo: repo, PullNumber: int(prNumber)})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func createPR(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	title, _ := params["title"].(string)
	head, _ := params["head"].(string)
	baseBranch, _ := params["base"].(string)
	req := &gen.CreatePRRequest{Title: title, Head: head, Base: baseBranch}
	if b, ok := params["body"].(string); ok {
		req.Body.SetTo(b)
	}
	if draft, ok := params["draft"].(bool); ok {
		req.Draft.SetTo(draft)
	}
	res, err := c.PullsCreate(ctx, req, gen.PullsCreateParams{Owner: owner, Repo: repo})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func listPRFiles(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	prNumber, _ := params["pr_number"].(float64)
	p := gen.PullsListFilesParams{Owner: owner, Repo: repo, PullNumber: int(prNumber)}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	res, err := c.PullsListFiles(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Search
// =============================================================================

func searchRepos(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	query, _ := params["query"].(string)
	p := gen.SearchReposParams{Q: query}
	if sort, ok := params["sort"].(string); ok && sort != "" {
		p.Sort.SetTo(sort)
	}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	if pg, ok := params["page"].(float64); ok {
		p.Page.SetTo(int(pg))
	}
	res, err := c.SearchRepos(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func searchCode(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	query, _ := params["query"].(string)
	p := gen.SearchCodeParams{Q: query}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	if pg, ok := params["page"].(float64); ok {
		p.Page.SetTo(int(pg))
	}
	res, err := c.SearchCode(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func searchIssues(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	query, _ := params["query"].(string)
	p := gen.SearchIssuesParams{Q: query}
	if sort, ok := params["sort"].(string); ok && sort != "" {
		p.Sort.SetTo(sort)
	}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	if pg, ok := params["page"].(float64); ok {
		p.Page.SetTo(int(pg))
	}
	res, err := c.SearchIssues(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Actions
// =============================================================================

func listWorkflows(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)
	p := gen.ActionsListWorkflowsParams{Owner: owner, Repo: repo}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	res, err := c.ActionsListWorkflows(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func listWorkflowRuns(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)

	// Use filtered endpoint if workflow_id is provided
	if workflowID, ok := params["workflow_id"].(string); ok && workflowID != "" {
		p := gen.ActionsListWorkflowRunsByIdParams{Owner: owner, Repo: repo, WorkflowID: workflowID}
		if status, ok := params["status"].(string); ok && status != "" {
			p.Status.SetTo(status)
		}
		if pp, ok := params["per_page"].(float64); ok {
			p.PerPage.SetTo(int(pp))
		}
		res, err := c.ActionsListWorkflowRunsById(ctx, p)
		if err != nil {
			return "", err
		}
		return toJSON(res)
	}

	p := gen.ActionsListWorkflowRunsParams{Owner: owner, Repo: repo}
	if status, ok := params["status"].(string); ok && status != "" {
		p.Status.SetTo(status)
	}
	if pp, ok := params["per_page"].(float64); ok {
		p.PerPage.SetTo(int(pp))
	}
	res, err := c.ActionsListWorkflowRuns(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Composite: describe_user
// =============================================================================

func describeUser(ctx context.Context, params map[string]any) (string, error) {
	username, _ := params["username"].(string)

	type result struct {
		key string
		val string
		err error
	}

	ch := make(chan result, 5)
	var wg sync.WaitGroup

	calls := []struct {
		key    string
		params map[string]any
		fn     toolHandler
	}{
		{"profile", map[string]any{"username": username}, getUser},
		{"repos", map[string]any{"username": username, "sort": "updated", "per_page": float64(10)}, listRepos},
		{"starred", map[string]any{"username": username, "per_page": float64(10)}, listStarredRepos},
		{"orgs", map[string]any{"username": username}, listOrgs},
		{"events", map[string]any{"username": username, "per_page": float64(10)}, listPublicEvents},
	}

	for _, c := range calls {
		wg.Add(1)
		go func(key string, fn toolHandler, p map[string]any) {
			defer wg.Done()
			v, err := fn(ctx, p)
			ch <- result{key: key, val: v, err: err}
		}(c.key, c.fn, c.params)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	raw := make(map[string]string, 5)
	for r := range ch {
		if r.err != nil {
			raw[r.key] = ""
		} else {
			raw[r.key] = r.val
		}
	}

	out := map[string]any{}

	// Profile: extract only essential fields
	if s := raw["profile"]; s != "" {
		var u map[string]any
		if json.Unmarshal([]byte(s), &u) == nil {
			p := map[string]any{}
			for _, k := range []string{"name", "html_url", "bio", "public_repos", "followers", "following"} {
				if v, ok := u[k]; ok && v != nil {
					p[k] = v
				}
			}
			out["profile"] = p
		}
	}

	// Repos: name, description, language, stars, fork
	if s := raw["repos"]; s != "" {
		var repos []map[string]any
		if json.Unmarshal([]byte(s), &repos) == nil {
			compact := make([]map[string]any, 0, len(repos))
			for _, r := range repos {
				c := map[string]any{"name": r["name"]}
				if v := r["description"]; v != nil && v != "" {
					c["desc"] = v
				}
				if v := r["language"]; v != nil && v != "" {
					c["lang"] = v
				}
				if v, ok := r["stargazers_count"].(float64); ok && v > 0 {
					c["stars"] = int(v)
				}
				if v, ok := r["fork"].(bool); ok && v {
					c["fork"] = true
				}
				compact = append(compact, c)
			}
			out["repos"] = compact
		}
	}

	// Starred: full_name, description, language, topics
	if s := raw["starred"]; s != "" {
		var starred []map[string]any
		if json.Unmarshal([]byte(s), &starred) == nil {
			compact := make([]map[string]any, 0, len(starred))
			for _, r := range starred {
				c := map[string]any{"repo": r["full_name"]}
				if v := r["description"]; v != nil && v != "" {
					c["desc"] = v
				}
				if v := r["language"]; v != nil && v != "" {
					c["lang"] = v
				}
				if arr, ok := r["topics"].([]any); ok && len(arr) > 0 {
					c["topics"] = arr
				}
				compact = append(compact, c)
			}
			out["starred"] = compact
		}
	}

	// Orgs: login, description
	if s := raw["orgs"]; s != "" {
		var orgs []map[string]any
		if json.Unmarshal([]byte(s), &orgs) == nil && len(orgs) > 0 {
			compact := make([]map[string]any, 0, len(orgs))
			for _, o := range orgs {
				c := map[string]any{"login": o["login"]}
				if v := o["description"]; v != nil && v != "" {
					c["desc"] = v
				}
				compact = append(compact, c)
			}
			out["orgs"] = compact
		}
	}

	// Events: type, repo, date
	if s := raw["events"]; s != "" {
		var events []map[string]any
		if json.Unmarshal([]byte(s), &events) == nil && len(events) > 0 {
			compact := make([]map[string]any, 0, len(events))
			for _, e := range events {
				c := map[string]any{"type": e["type"]}
				if r, ok := e["repo"].(map[string]any); ok {
					if name, ok := r["name"].(string); ok {
						c["repo"] = name
					}
				}
				if v, ok := e["created_at"].(string); ok && len(v) >= 16 {
					c["date"] = v[:10] + " " + v[11:16]
				}
				compact = append(compact, c)
			}
			out["events"] = compact
		}
	}

	return toJSON(out)
}

// =============================================================================
// Composite: describe_repo
// =============================================================================

func describeRepo(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repoName, _ := params["repo"].(string)

	type result struct {
		key string
		val string
		err error
	}

	ch := make(chan result, 5)
	var wg sync.WaitGroup

	calls := []struct {
		key    string
		params map[string]any
		fn     toolHandler
	}{
		{"repo", map[string]any{"owner": owner, "repo": repoName}, getRepo},
		{"readme", map[string]any{"owner": owner, "repo": repoName, "path": "README.md"}, getFileContent},
		{"branches", map[string]any{"owner": owner, "repo": repoName, "per_page": float64(10)}, listBranches},
		{"issues", map[string]any{"owner": owner, "repo": repoName, "state": "open", "per_page": float64(10)}, listIssues},
		{"prs", map[string]any{"owner": owner, "repo": repoName, "state": "open", "per_page": float64(10)}, listPRs},
	}

	for _, c := range calls {
		wg.Add(1)
		go func(key string, fn toolHandler, p map[string]any) {
			defer wg.Done()
			v, err := fn(ctx, p)
			ch <- result{key: key, val: v, err: err}
		}(c.key, c.fn, c.params)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	raw := make(map[string]string, 5)
	for r := range ch {
		if r.err != nil {
			raw[r.key] = ""
		} else {
			raw[r.key] = r.val
		}
	}

	out := map[string]any{
		"_note": "Partial data. Use get_repo, get_issue, get_pr, get_file_content for full details.",
	}

	// Repo: essential fields only (no html_url, owner)
	if s := raw["repo"]; s != "" {
		var r map[string]any
		if json.Unmarshal([]byte(s), &r) == nil {
			p := map[string]any{}
			for _, k := range []string{"full_name", "description", "language", "visibility", "stargazers_count", "forks_count", "open_issues_count", "default_branch", "archived", "fork"} {
				if v, ok := r[k]; ok && v != nil {
					p[k] = v
				}
			}
			if arr, ok := r["topics"].([]any); ok && len(arr) > 0 {
				p["topics"] = arr
			}
			if v, ok := r["created_at"].(string); ok && len(v) >= 10 {
				p["created"] = v[:10]
			}
			if v, ok := r["pushed_at"].(string); ok && len(v) >= 10 {
				p["last_push"] = v[:10]
			}
			out["repo"] = p
		}
	}

	// README: extract content field, truncate to 2000 chars
	if s := raw["readme"]; s != "" {
		var f map[string]any
		if json.Unmarshal([]byte(s), &f) == nil {
			if content, ok := f["content"].(string); ok {
				if len(content) > 2000 {
					out["readme"] = content[:2000] + "...(truncated)"
				} else {
					out["readme"] = content
				}
			}
		}
	}

	// Branches: already compact (name, protected)
	if s := raw["branches"]; s != "" {
		var branches []map[string]any
		if json.Unmarshal([]byte(s), &branches) == nil {
			names := make([]string, 0, len(branches))
			for _, b := range branches {
				if name, ok := b["name"].(string); ok {
					names = append(names, name)
				}
			}
			out["branches"] = names
		}
	}

	// Issues: number, title, author, labels, created_at
	if s := raw["issues"]; s != "" {
		var issues []map[string]any
		if json.Unmarshal([]byte(s), &issues) == nil {
			compact := make([]map[string]any, 0, len(issues))
			for _, i := range issues {
				c := map[string]any{"number": i["number"], "title": i["title"]}
				if u, ok := i["user"].(map[string]any); ok {
					if login, ok := u["login"].(string); ok {
						c["author"] = login
					}
				}
				if v, ok := i["created_at"].(string); ok && len(v) >= 10 {
					c["date"] = v[:10]
				}
				if labels, ok := i["labels"].([]any); ok && len(labels) > 0 {
					names := make([]string, 0, len(labels))
					for _, l := range labels {
						if lm, ok := l.(map[string]any); ok {
							if name, ok := lm["name"].(string); ok {
								names = append(names, name)
							}
						}
					}
					if len(names) > 0 {
						c["labels"] = names
					}
				}
				compact = append(compact, c)
			}
			out["issues"] = compact
		}
	}

	// PRs: number, title, author, draft, created_at
	if s := raw["prs"]; s != "" {
		var prs []map[string]any
		if json.Unmarshal([]byte(s), &prs) == nil {
			compact := make([]map[string]any, 0, len(prs))
			for _, p := range prs {
				c := map[string]any{"number": p["number"], "title": p["title"]}
				if u, ok := p["user"].(map[string]any); ok {
					if login, ok := u["login"].(string); ok {
						c["author"] = login
					}
				}
				if v, ok := p["draft"].(bool); ok && v {
					c["draft"] = true
				}
				if v, ok := p["created_at"].(string); ok && len(v) >= 10 {
					c["date"] = v[:10]
				}
				compact = append(compact, c)
			}
			out["prs"] = compact
		}
	}

	return toJSON(out)
}

// =============================================================================
// Composite: describe_pr
// =============================================================================

func describePR(ctx context.Context, params map[string]any) (string, error) {
	owner, _ := params["owner"].(string)
	repoName, _ := params["repo"].(string)
	prNumber, _ := params["pr_number"].(float64)

	type result struct {
		key string
		val string
		err error
	}

	ch := make(chan result, 2)
	var wg sync.WaitGroup

	calls := []struct {
		key    string
		params map[string]any
		fn     toolHandler
	}{
		{"pr", map[string]any{"owner": owner, "repo": repoName, "pr_number": prNumber}, getPR},
		{"files", map[string]any{"owner": owner, "repo": repoName, "pr_number": prNumber, "per_page": float64(30)}, listPRFiles},
	}

	for _, c := range calls {
		wg.Add(1)
		go func(key string, fn toolHandler, p map[string]any) {
			defer wg.Done()
			v, err := fn(ctx, p)
			ch <- result{key: key, val: v, err: err}
		}(c.key, c.fn, c.params)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	raw := make(map[string]string, 2)
	for r := range ch {
		if r.err != nil {
			raw[r.key] = ""
		} else {
			raw[r.key] = r.val
		}
	}

	out := map[string]any{
		"_note": "Partial data. Use get_pr and list_pr_files for full details including patch diffs.",
	}

	// PR: essential fields (no html_url, truncate body)
	if s := raw["pr"]; s != "" {
		var pr map[string]any
		if json.Unmarshal([]byte(s), &pr) == nil {
			p := map[string]any{}
			for _, k := range []string{"number", "title", "state", "draft", "merged"} {
				if v, ok := pr[k]; ok && v != nil {
					p[k] = v
				}
			}
			if v, ok := pr["created_at"].(string); ok && len(v) >= 10 {
				p["created"] = v[:10]
			}
			if v, ok := pr["updated_at"].(string); ok && len(v) >= 10 {
				p["updated"] = v[:10]
			}
			if v, ok := pr["merged_at"].(string); ok && len(v) >= 10 {
				p["merged_at"] = v[:10]
			}
			if u, ok := pr["user"].(map[string]any); ok {
				if login, ok := u["login"].(string); ok {
					p["author"] = login
				}
			}
			if head, ok := pr["head"].(map[string]any); ok {
				if ref, ok := head["ref"].(string); ok {
					p["head"] = ref
				}
			}
			if base, ok := pr["base"].(map[string]any); ok {
				if ref, ok := base["ref"].(string); ok {
					p["base"] = ref
				}
			}
			// body: truncate to 3000 chars
			if body, ok := pr["body"].(string); ok && body != "" {
				if len(body) > 3000 {
					p["body"] = body[:3000] + "...(truncated)"
				} else {
					p["body"] = body
				}
			}
			out["pr"] = p
		}
	}

	// Files: filename, status, additions, deletions (no patch)
	if s := raw["files"]; s != "" {
		var files []map[string]any
		if json.Unmarshal([]byte(s), &files) == nil {
			compact := make([]map[string]any, 0, len(files))
			for _, f := range files {
				c := map[string]any{"file": f["filename"]}
				if v, ok := f["status"].(string); ok {
					c["status"] = v
				}
				if v, ok := f["additions"].(float64); ok {
					c["add"] = int(v)
				}
				if v, ok := f["deletions"].(float64); ok {
					c["del"] = int(v)
				}
				compact = append(compact, c)
			}
			out["files"] = compact
		}
	}

	return toJSON(out)
}
