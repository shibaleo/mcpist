package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-faster/jx"

	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/broker"
	"mcpist/server/pkg/jiraapi"
	gen "mcpist/server/pkg/jiraapi/gen"
)

const (
	jiraAPIPath    = "/rest/api/3"
	jiraAPIVersion = "3"
)

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

// moduleDescriptions holds module descriptions
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Jira API - Issue/Project operations (search, create, update, comment, transition)",
	"ja-JP": "Jira API - Issue/Project操作（検索、作成、更新、コメント、遷移）",
}

// Descriptions returns multilingual module descriptions
func (m *JiraModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description (English)
func (m *JiraModule) Description() string {
	return moduleDescriptions["en-US"]
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

// ToCompact converts JSON result to compact format (MD or CSV)
// Implements modules.CompactConverter interface
func (m *JiraModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// Resources returns all available resources (none for Jira)
func (m *JiraModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *JiraModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// ogen client helper
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "jira")
	if err != nil {
		return nil
	}
	return credentials
}

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}

	switch creds.AuthType {
	case broker.AuthTypeBasic:
		domain, _ := creds.Metadata["domain"].(string)
		if domain == "" {
			return nil, fmt.Errorf("jira domain not configured")
		}
		serverURL := fmt.Sprintf("https://%s%s", domain, jiraAPIPath)
		return jiraapi.NewBasicClient(serverURL, creds.Username, creds.Password)
	default:
		// OAuth 2.0
		cloudID, _ := creds.Metadata["cloud_id"].(string)
		if cloudID == "" {
			return nil, fmt.Errorf("jira cloud_id not configured")
		}
		serverURL := fmt.Sprintf("https://api.atlassian.com/ex/jira/%s%s", cloudID, jiraAPIPath)
		return jiraapi.NewBearerClient(serverURL, creds.AccessToken)
	}
}

var toJSON = modules.ToJSON

// toRaw converts any value to jx.Raw (JSON bytes).
func toRaw(v any) (jx.Raw, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return jx.Raw(b), nil
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
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetMyself(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Projects
// =============================================================================

func listProjects(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	p := gen.SearchProjectsParams{}
	if sa, ok := params["start_at"].(float64); ok {
		p.StartAt.SetTo(int(sa))
	}
	if mr, ok := params["max_results"].(float64); ok {
		p.MaxResults.SetTo(int(mr))
	}
	res, err := c.SearchProjects(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getProject(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectKey, _ := params["project_key"].(string)
	res, err := c.GetProject(ctx, gen.GetProjectParams{ProjectIdOrKey: projectKey})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Issues
// =============================================================================

func search(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	jql, _ := params["jql"].(string)
	p := gen.SearchIssuesUsingJqlParams{Jql: jql}
	if sa, ok := params["start_at"].(float64); ok {
		p.StartAt.SetTo(int(sa))
	}
	if mr, ok := params["max_results"].(float64); ok {
		p.MaxResults.SetTo(int(mr))
	}
	if fields, ok := params["fields"].([]interface{}); ok && len(fields) > 0 {
		fieldStrs := make([]string, 0, len(fields))
		for _, f := range fields {
			if fs, ok := f.(string); ok {
				fieldStrs = append(fieldStrs, fs)
			}
		}
		if len(fieldStrs) > 0 {
			p.Fields.SetTo(strings.Join(fieldStrs, ","))
		}
	} else {
		p.Fields.SetTo("summary,status,priority,assignee,created,updated")
	}
	res, err := c.SearchIssuesUsingJql(ctx, p)
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
	issueKey, _ := params["issue_key"].(string)
	p := gen.GetIssueParams{IssueIdOrKey: issueKey}
	if fields, ok := params["fields"].([]interface{}); ok && len(fields) > 0 {
		fieldStrs := make([]string, 0, len(fields))
		for _, f := range fields {
			if fs, ok := f.(string); ok {
				fieldStrs = append(fieldStrs, fs)
			}
		}
		if len(fieldStrs) > 0 {
			p.Fields.SetTo(strings.Join(fieldStrs, ","))
		}
	}
	res, err := c.GetIssue(ctx, p)
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
	projectKey, _ := params["project_key"].(string)
	issueType, _ := params["issue_type"].(string)
	summary, _ := params["summary"].(string)

	fields := gen.IssueFields{}
	fields.Project, _ = toRaw(map[string]string{"key": projectKey})
	fields.Issuetype, _ = toRaw(map[string]string{"name": issueType})
	fields.Summary.SetTo(summary)

	if description, ok := params["description"].(string); ok && description != "" {
		fields.Description, _ = toRaw(adfDocument(description))
	}
	if assigneeID, ok := params["assignee_account_id"].(string); ok && assigneeID != "" {
		fields.Assignee, _ = toRaw(map[string]string{"accountId": assigneeID})
	}
	if priority, ok := params["priority"].(string); ok && priority != "" {
		fields.Priority, _ = toRaw(map[string]string{"name": priority})
	}
	if labels, ok := params["labels"].([]interface{}); ok && len(labels) > 0 {
		labelStrs := make([]string, 0, len(labels))
		for _, l := range labels {
			if ls, ok := l.(string); ok {
				labelStrs = append(labelStrs, ls)
			}
		}
		fields.Labels = labelStrs
	}
	if parentKey, ok := params["parent_key"].(string); ok && parentKey != "" {
		fields.Parent, _ = toRaw(map[string]string{"key": parentKey})
	}

	res, err := c.CreateIssue(ctx, &gen.CreateIssueRequest{Fields: fields})
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
	issueKey, _ := params["issue_key"].(string)

	fields := gen.IssueFields{}
	if summary, ok := params["summary"].(string); ok {
		fields.Summary.SetTo(summary)
	}
	if description, ok := params["description"].(string); ok {
		fields.Description, _ = toRaw(adfDocument(description))
	}
	if assigneeID, ok := params["assignee_account_id"].(string); ok {
		fields.Assignee, _ = toRaw(map[string]string{"accountId": assigneeID})
	}
	if priority, ok := params["priority"].(string); ok {
		fields.Priority, _ = toRaw(map[string]string{"name": priority})
	}
	if labels, ok := params["labels"].([]interface{}); ok {
		labelStrs := make([]string, 0, len(labels))
		for _, l := range labels {
			if ls, ok := l.(string); ok {
				labelStrs = append(labelStrs, ls)
			}
		}
		fields.Labels = labelStrs
	}

	err = c.UpdateIssue(ctx, &gen.UpdateIssueRequest{Fields: gen.OptIssueFields{Value: fields, Set: true}}, gen.UpdateIssueParams{IssueIdOrKey: issueKey})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"updated":true,"issue_key":"%s"}`, issueKey), nil
}

// =============================================================================
// Transitions
// =============================================================================

func getTransitions(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	issueKey, _ := params["issue_key"].(string)
	res, err := c.GetTransitions(ctx, gen.GetTransitionsParams{IssueIdOrKey: issueKey})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func transitionIssue(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	issueKey, _ := params["issue_key"].(string)
	transitionID, _ := params["transition_id"].(string)

	req := gen.DoTransitionRequest{
		Transition: gen.TransitionRef{ID: transitionID},
	}

	if comment, ok := params["comment"].(string); ok && comment != "" {
		update := map[string]interface{}{
			"comment": []map[string]interface{}{
				{"add": map[string]interface{}{"body": adfDocument(comment)}},
			},
		}
		req.Update, _ = toRaw(update)
	}

	err = c.DoTransition(ctx, &req, gen.DoTransitionParams{IssueIdOrKey: issueKey})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"transitioned":true,"issue_key":"%s","transition_id":"%s"}`, issueKey, transitionID), nil
}

// =============================================================================
// Comments
// =============================================================================

func getComments(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	issueKey, _ := params["issue_key"].(string)
	p := gen.GetCommentsParams{IssueIdOrKey: issueKey}
	if sa, ok := params["start_at"].(float64); ok {
		p.StartAt.SetTo(int(sa))
	}
	if mr, ok := params["max_results"].(float64); ok {
		p.MaxResults.SetTo(int(mr))
	}
	res, err := c.GetComments(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func addComment(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	issueKey, _ := params["issue_key"].(string)
	body, _ := params["body"].(string)

	adfBody, _ := toRaw(adfDocument(body))
	res, err := c.AddComment(ctx, &gen.AddCommentRequest{Body: adfBody}, gen.AddCommentParams{IssueIdOrKey: issueKey})
	if err != nil {
		return "", err
	}
	return toJSON(res)
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
