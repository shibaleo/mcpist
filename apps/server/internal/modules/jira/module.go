package jira

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/store"
)

const (
	jiraAPIPath    = "/rest/api/3"
	jiraAPIVersion = "3"
)

var client = httpclient.New()

// JiraModule implements the Module interface for Jira API
type JiraModule struct{}

// New creates a new JiraModule instance
func New() *JiraModule {
	return &JiraModule{}
}

// Name returns the module name
func (m *JiraModule) Name() string {
	return "jira"
}

// moduleDescriptions holds multilingual module descriptions
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Jira API - Issue/Project operations (search, create, update, comment, transition)",
	"ja-JP": "Jira API - Issue/Project操作（検索、作成、更新、コメント、遷移）",
}

// Descriptions returns multilingual module descriptions
func (m *JiraModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description for a specific language
func (m *JiraModule) Description(lang string) string {
	return modules.GetLocalizedText(moduleDescriptions, lang)
}

// APIVersion returns the Jira API version
func (m *JiraModule) APIVersion() string {
	return jiraAPIVersion
}

// Tools returns all available tools
func (m *JiraModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *JiraModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for Jira)
func (m *JiraModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *JiraModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// Prompts returns all available prompts (none for Jira)
func (m *JiraModule) Prompts() []modules.Prompt {
	return nil
}

// GetPrompt generates a prompt with arguments (not implemented)
func (m *JiraModule) GetPrompt(ctx context.Context, name string, args map[string]any) (string, error) {
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
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "jira")
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
		"Accept": "application/json",
	}

	switch creds.AuthType {
	case store.AuthTypeBasic:
		// Basic auth: username:password
		auth := base64.StdEncoding.EncodeToString([]byte(creds.Username + ":" + creds.Password))
		h["Authorization"] = "Basic " + auth
	case store.AuthTypeOAuth2:
		// Bearer token (OAuth 2.0)
		h["Authorization"] = "Bearer " + creds.AccessToken
	}

	return h
}

func baseURL(ctx context.Context) string {
	creds := getCredentials(ctx)
	if creds == nil {
		return ""
	}
	domain := creds.Metadata["domain"]
	if domain == "" {
		return ""
	}
	return fmt.Sprintf("https://%s%s", domain, jiraAPIPath)
}

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	{
		ID:   "jira:get_myself",
		Name: "get_myself",
		Descriptions: modules.LocalizedText{
			"en-US": "Get information about the current Jira user (myself).",
			"ja-JP": "現在のJiraユーザー（自分自身）の情報を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "jira:list_projects",
		Name: "list_projects",
		Descriptions: modules.LocalizedText{
			"en-US": "List all Jira projects accessible to the current user.",
			"ja-JP": "現在のユーザーがアクセス可能なすべてのJiraプロジェクトを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"start_at":    {Type: "number", Description: "Starting index for pagination. Default: 0"},
				"max_results": {Type: "number", Description: "Maximum results to return. Default: 50"},
			},
		},
	},
	{
		ID:   "jira:get_project",
		Name: "get_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific Jira project.",
			"ja-JP": "特定のJiraプロジェクトの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_key": {Type: "string", Description: "Project key (e.g., 'PROJ') or ID"},
			},
			Required: []string{"project_key"},
		},
	},
	{
		ID:   "jira:search",
		Name: "search",
		Descriptions: modules.LocalizedText{
			"en-US": "Search for Jira issues using JQL (Jira Query Language). Example JQL: 'project = PROJ AND status = \"In Progress\"'",
			"ja-JP": "JQL（Jira Query Language）を使用してJira課題を検索します。JQL例：'project = PROJ AND status = \"In Progress\"'",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"jql":         {Type: "string", Description: "JQL query string. Example: 'project = PROJ AND status != Done ORDER BY created DESC'"},
				"start_at":    {Type: "number", Description: "Starting index for pagination. Default: 0"},
				"max_results": {Type: "number", Description: "Maximum results to return. Default: 50"},
				"fields":      {Type: "array", Description: "Fields to return. Default: summary, status, priority, assignee, created, updated"},
			},
			Required: []string{"jql"},
		},
	},
	{
		ID:   "jira:get_issue",
		Name: "get_issue",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific Jira issue by key or ID.",
			"ja-JP": "キーまたはIDで特定のJira課題の詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"issue_key": {Type: "string", Description: "Issue key (e.g., 'PROJ-123') or ID"},
				"fields":    {Type: "array", Description: "Specific fields to return. If not specified, returns common fields."},
			},
			Required: []string{"issue_key"},
		},
	},
	{
		ID:   "jira:create_issue",
		Name: "create_issue",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new Jira issue.",
			"ja-JP": "新しいJira課題を作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_key":         {Type: "string", Description: "Project key (e.g., 'PROJ')"},
				"issue_type":          {Type: "string", Description: "Issue type (e.g., 'Task', 'Bug', 'Story', 'Epic')"},
				"summary":             {Type: "string", Description: "Issue summary/title"},
				"description":         {Type: "string", Description: "Issue description"},
				"assignee_account_id": {Type: "string", Description: "Assignee's Atlassian account ID"},
				"priority":            {Type: "string", Description: "Priority name (e.g., 'High', 'Medium', 'Low')"},
				"labels":              {Type: "array", Description: "Labels to add to the issue"},
				"parent_key":          {Type: "string", Description: "Parent issue key for subtasks"},
			},
			Required: []string{"project_key", "issue_type", "summary"},
		},
	},
	{
		ID:   "jira:update_issue",
		Name: "update_issue",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing Jira issue.",
			"ja-JP": "既存のJira課題を更新します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"issue_key":           {Type: "string", Description: "Issue key (e.g., 'PROJ-123')"},
				"summary":             {Type: "string", Description: "New summary/title"},
				"description":         {Type: "string", Description: "New description"},
				"assignee_account_id": {Type: "string", Description: "New assignee's Atlassian account ID"},
				"priority":            {Type: "string", Description: "New priority name"},
				"labels":              {Type: "array", Description: "New labels (replaces existing)"},
			},
			Required: []string{"issue_key"},
		},
	},
	{
		ID:   "jira:get_transitions",
		Name: "get_transitions",
		Descriptions: modules.LocalizedText{
			"en-US": "Get available transitions for an issue. Use this to find valid transition IDs before changing issue status.",
			"ja-JP": "課題で利用可能なトランジションを取得します。課題のステータスを変更する前に有効なトランジションIDを見つけるために使用します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"issue_key": {Type: "string", Description: "Issue key (e.g., 'PROJ-123')"},
			},
			Required: []string{"issue_key"},
		},
	},
	{
		ID:   "jira:transition_issue",
		Name: "transition_issue",
		Descriptions: modules.LocalizedText{
			"en-US": "Transition an issue to a new status. Use get_transitions first to get valid transition IDs.",
			"ja-JP": "課題を新しいステータスに遷移させます。最初にget_transitionsを使用して有効なトランジションIDを取得してください。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"issue_key":     {Type: "string", Description: "Issue key (e.g., 'PROJ-123')"},
				"transition_id": {Type: "string", Description: "Transition ID (get from get_transitions)"},
				"comment":       {Type: "string", Description: "Optional comment to add with the transition"},
			},
			Required: []string{"issue_key", "transition_id"},
		},
	},
	{
		ID:   "jira:get_comments",
		Name: "get_comments",
		Descriptions: modules.LocalizedText{
			"en-US": "Get comments on a Jira issue.",
			"ja-JP": "Jira課題のコメントを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"issue_key":   {Type: "string", Description: "Issue key (e.g., 'PROJ-123')"},
				"start_at":    {Type: "number", Description: "Starting index for pagination. Default: 0"},
				"max_results": {Type: "number", Description: "Maximum results to return. Default: 50"},
			},
			Required: []string{"issue_key"},
		},
	},
	{
		ID:   "jira:add_comment",
		Name: "add_comment",
		Descriptions: modules.LocalizedText{
			"en-US": "Add a comment to a Jira issue.",
			"ja-JP": "Jira課題にコメントを追加します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"issue_key": {Type: "string", Description: "Issue key (e.g., 'PROJ-123')"},
				"body":      {Type: "string", Description: "Comment text"},
			},
			Required: []string{"issue_key", "body"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"get_myself":       getMyself,
	"list_projects":    listProjects,
	"get_project":      getProject,
	"search":           search,
	"get_issue":        getIssue,
	"create_issue":     createIssue,
	"update_issue":     updateIssue,
	"get_transitions":  getTransitions,
	"transition_issue": transitionIssue,
	"get_comments":     getComments,
	"add_comment":      addComment,
}

// =============================================================================
// User
// =============================================================================

func getMyself(ctx context.Context, params map[string]any) (string, error) {
	endpoint := baseURL(ctx) + "/myself"
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Projects
// =============================================================================

func listProjects(ctx context.Context, params map[string]any) (string, error) {
	startAt := 0
	if sa, ok := params["start_at"].(float64); ok {
		startAt = int(sa)
	}
	maxResults := 50
	if mr, ok := params["max_results"].(float64); ok {
		maxResults = int(mr)
	}
	endpoint := fmt.Sprintf("%s/project/search?startAt=%d&maxResults=%d", baseURL(ctx), startAt, maxResults)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getProject(ctx context.Context, params map[string]any) (string, error) {
	projectKey, _ := params["project_key"].(string)
	endpoint := fmt.Sprintf("%s/project/%s", baseURL(ctx), projectKey)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Issues
// =============================================================================

func search(ctx context.Context, params map[string]any) (string, error) {
	jql, _ := params["jql"].(string)
	query := url.Values{}
	query.Set("jql", jql)

	startAt := 0
	if sa, ok := params["start_at"].(float64); ok {
		startAt = int(sa)
	}
	query.Set("startAt", fmt.Sprintf("%d", startAt))

	maxResults := 50
	if mr, ok := params["max_results"].(float64); ok {
		maxResults = int(mr)
	}
	query.Set("maxResults", fmt.Sprintf("%d", maxResults))

	if fields, ok := params["fields"].([]interface{}); ok && len(fields) > 0 {
		fieldStrs := make([]string, 0, len(fields))
		for _, f := range fields {
			if fs, ok := f.(string); ok {
				fieldStrs = append(fieldStrs, fs)
			}
		}
		if len(fieldStrs) > 0 {
			query.Set("fields", joinStrings(fieldStrs, ","))
		}
	} else {
		query.Set("fields", "summary,status,priority,assignee,created,updated")
	}

	endpoint := fmt.Sprintf("%s/search/jql?%s", baseURL(ctx), query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getIssue(ctx context.Context, params map[string]any) (string, error) {
	issueKey, _ := params["issue_key"].(string)
	query := url.Values{}
	if fields, ok := params["fields"].([]interface{}); ok && len(fields) > 0 {
		fieldStrs := make([]string, 0, len(fields))
		for _, f := range fields {
			if fs, ok := f.(string); ok {
				fieldStrs = append(fieldStrs, fs)
			}
		}
		if len(fieldStrs) > 0 {
			query.Set("fields", joinStrings(fieldStrs, ","))
		}
	}

	queryStr := ""
	if len(query) > 0 {
		queryStr = "?" + query.Encode()
	}

	endpoint := fmt.Sprintf("%s/issue/%s%s", baseURL(ctx), issueKey, queryStr)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createIssue(ctx context.Context, params map[string]any) (string, error) {
	projectKey, _ := params["project_key"].(string)
	issueType, _ := params["issue_type"].(string)
	summary, _ := params["summary"].(string)

	fields := map[string]interface{}{
		"project":   map[string]string{"key": projectKey},
		"issuetype": map[string]string{"name": issueType},
		"summary":   summary,
	}

	if description, ok := params["description"].(string); ok && description != "" {
		fields["description"] = adfDocument(description)
	}
	if assigneeID, ok := params["assignee_account_id"].(string); ok && assigneeID != "" {
		fields["assignee"] = map[string]string{"accountId": assigneeID}
	}
	if priority, ok := params["priority"].(string); ok && priority != "" {
		fields["priority"] = map[string]string{"name": priority}
	}
	if labels, ok := params["labels"].([]interface{}); ok && len(labels) > 0 {
		labelStrs := make([]string, 0, len(labels))
		for _, l := range labels {
			if ls, ok := l.(string); ok {
				labelStrs = append(labelStrs, ls)
			}
		}
		fields["labels"] = labelStrs
	}
	if parentKey, ok := params["parent_key"].(string); ok && parentKey != "" {
		fields["parent"] = map[string]string{"key": parentKey}
	}

	body := map[string]interface{}{"fields": fields}
	endpoint := baseURL(ctx) + "/issue"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updateIssue(ctx context.Context, params map[string]any) (string, error) {
	issueKey, _ := params["issue_key"].(string)
	fields := make(map[string]interface{})

	if summary, ok := params["summary"].(string); ok {
		fields["summary"] = summary
	}
	if description, ok := params["description"].(string); ok {
		fields["description"] = adfDocument(description)
	}
	if assigneeID, ok := params["assignee_account_id"].(string); ok {
		fields["assignee"] = map[string]string{"accountId": assigneeID}
	}
	if priority, ok := params["priority"].(string); ok {
		fields["priority"] = map[string]string{"name": priority}
	}
	if labels, ok := params["labels"].([]interface{}); ok {
		labelStrs := make([]string, 0, len(labels))
		for _, l := range labels {
			if ls, ok := l.(string); ok {
				labelStrs = append(labelStrs, ls)
			}
		}
		fields["labels"] = labelStrs
	}

	body := map[string]interface{}{"fields": fields}
	endpoint := fmt.Sprintf("%s/issue/%s", baseURL(ctx), issueKey)
	_, err := client.DoJSON("PUT", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"updated": true, "issue_key": "%s"}`, issueKey), nil
}

// =============================================================================
// Transitions
// =============================================================================

func getTransitions(ctx context.Context, params map[string]any) (string, error) {
	issueKey, _ := params["issue_key"].(string)
	endpoint := fmt.Sprintf("%s/issue/%s/transitions", baseURL(ctx), issueKey)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func transitionIssue(ctx context.Context, params map[string]any) (string, error) {
	issueKey, _ := params["issue_key"].(string)
	transitionID, _ := params["transition_id"].(string)

	body := map[string]interface{}{
		"transition": map[string]string{"id": transitionID},
	}

	if comment, ok := params["comment"].(string); ok && comment != "" {
		body["update"] = map[string]interface{}{
			"comment": []map[string]interface{}{
				{"add": map[string]interface{}{"body": adfDocument(comment)}},
			},
		}
	}

	endpoint := fmt.Sprintf("%s/issue/%s/transitions", baseURL(ctx), issueKey)
	_, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"transitioned": true, "issue_key": "%s", "transition_id": "%s"}`, issueKey, transitionID), nil
}

// =============================================================================
// Comments
// =============================================================================

func getComments(ctx context.Context, params map[string]any) (string, error) {
	issueKey, _ := params["issue_key"].(string)
	startAt := 0
	if sa, ok := params["start_at"].(float64); ok {
		startAt = int(sa)
	}
	maxResults := 50
	if mr, ok := params["max_results"].(float64); ok {
		maxResults = int(mr)
	}
	endpoint := fmt.Sprintf("%s/issue/%s/comment?startAt=%d&maxResults=%d", baseURL(ctx), issueKey, startAt, maxResults)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func addComment(ctx context.Context, params map[string]any) (string, error) {
	issueKey, _ := params["issue_key"].(string)
	body, _ := params["body"].(string)

	payload := map[string]interface{}{
		"body": adfDocument(body),
	}

	endpoint := fmt.Sprintf("%s/issue/%s/comment", baseURL(ctx), issueKey)
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), payload)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Helpers
// =============================================================================

// adfDocument creates an Atlassian Document Format document with a single paragraph
func adfDocument(text string) map[string]interface{} {
	return map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []map[string]interface{}{
			{
				"type": "paragraph",
				"content": []map[string]interface{}{
					{"type": "text", "text": text},
				},
			},
		},
	}
}

// joinStrings joins strings with a separator
func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
