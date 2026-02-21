package todoist

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/broker"
	"mcpist/server/pkg/todoistapi"
	gen "mcpist/server/pkg/todoistapi/gen"
)

const (
	todoistVersion  = "v1"
	todoistTokenURL = "https://todoist.com/oauth/access_token"
)

// TodoistModule implements the Module interface for Todoist API
type TodoistModule struct{}

// New creates a new TodoistModule instance
func New() *TodoistModule {
	return &TodoistModule{}
}

// Module descriptions
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Todoist API - List, create, update, and delete tasks and projects",
	"ja-JP": "Todoist API - タスクとプロジェクトの一覧表示、作成、更新、削除",
}

// Name returns the module name
func (m *TodoistModule) Name() string {
	return "todoist"
}

// Descriptions returns the module descriptions in all languages
func (m *TodoistModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description (English)
func (m *TodoistModule) Description() string {
	return moduleDescriptions["en-US"]
}

// APIVersion returns the Todoist API version
func (m *TodoistModule) APIVersion() string {
	return todoistVersion
}

// Tools returns all available tools
func (m *TodoistModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *TodoistModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format.
// Implements modules.CompactConverter interface.
func (m *TodoistModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// Resources returns all available resources (none for Todoist)
func (m *TodoistModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *TodoistModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// ogen client helpers
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[todoist] No auth context")
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "todoist")
	if err != nil {
		log.Printf("[todoist] GetModuleToken error: %v", err)
		return nil
	}
	return credentials
}

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return todoistapi.NewClient(creds.AccessToken)
}

var toJSON = modules.ToJSON
var toStringSlice = modules.ToStringSlice

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	// Projects
	{
		ID:   "todoist:list_projects",
		Name: "list_projects",
		Descriptions: modules.LocalizedText{
			"en-US": "List all projects for the user.",
			"ja-JP": "ユーザーのすべてのプロジェクトを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "todoist:get_project",
		Name: "get_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific project.",
			"ja-JP": "特定のプロジェクトの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_id": {Type: "string", Description: "Project ID"},
			},
			Required: []string{"project_id"},
		},
	},
	// Tasks
	{
		ID:   "todoist:list_tasks",
		Name: "list_tasks",
		Descriptions: modules.LocalizedText{
			"en-US": "List active tasks. Can filter by project, label, or filter string.",
			"ja-JP": "アクティブなタスクを一覧表示します。プロジェクト、ラベル、フィルター文字列でフィルタリングできます。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_id": {Type: "string", Description: "Filter tasks by project ID"},
				"section_id": {Type: "string", Description: "Filter tasks by section ID"},
				"label":      {Type: "string", Description: "Filter tasks by label name"},
			},
		},
	},
	{
		ID:   "todoist:get_task",
		Name: "get_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific task.",
			"ja-JP": "特定のタスクの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_id": {Type: "string", Description: "Task ID"},
			},
			Required: []string{"task_id"},
		},
	},
	{
		ID:   "todoist:create_task",
		Name: "create_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new task.",
			"ja-JP": "新しいタスクを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"content":      {Type: "string", Description: "Task content (required)"},
				"description":  {Type: "string", Description: "Task description"},
				"project_id":   {Type: "string", Description: "Project ID (default: Inbox)"},
				"section_id":   {Type: "string", Description: "Section ID"},
				"parent_id":    {Type: "string", Description: "Parent task ID for subtasks"},
				"priority":     {Type: "number", Description: "Priority: 1 (normal) to 4 (urgent)"},
				"due_string":   {Type: "string", Description: "Due date in natural language (e.g., 'tomorrow', 'next Monday')"},
				"due_date":     {Type: "string", Description: "Due date (YYYY-MM-DD format)"},
				"due_datetime":  {Type: "string", Description: "Due datetime (RFC3339 format)"},
				"labels":       {Type: "array", Description: "Array of label names"},
				"assignee_id":  {Type: "string", Description: "Assignee user ID (for shared projects)"},
			},
			Required: []string{"content"},
		},
	},
	{
		ID:   "todoist:update_task",
		Name: "update_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing task.",
			"ja-JP": "既存のタスクを更新します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_id":      {Type: "string", Description: "Task ID (required)"},
				"content":      {Type: "string", Description: "New task content"},
				"description":  {Type: "string", Description: "New task description"},
				"priority":     {Type: "number", Description: "Priority: 1 (normal) to 4 (urgent)"},
				"due_string":   {Type: "string", Description: "Due date in natural language"},
				"due_date":     {Type: "string", Description: "Due date (YYYY-MM-DD format)"},
				"due_datetime":  {Type: "string", Description: "Due datetime (RFC3339 format)"},
				"labels":       {Type: "array", Description: "Array of label names"},
				"assignee_id":  {Type: "string", Description: "Assignee user ID"},
			},
			Required: []string{"task_id"},
		},
	},
	{
		ID:   "todoist:complete_task",
		Name: "complete_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Mark a task as completed.",
			"ja-JP": "タスクを完了としてマークします。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_id": {Type: "string", Description: "Task ID"},
			},
			Required: []string{"task_id"},
		},
	},
	{
		ID:   "todoist:reopen_task",
		Name: "reopen_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Reopen a completed task.",
			"ja-JP": "完了したタスクを再度開きます。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_id": {Type: "string", Description: "Task ID"},
			},
			Required: []string{"task_id"},
		},
	},
	{
		ID:   "todoist:delete_task",
		Name: "delete_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a task.",
			"ja-JP": "タスクを削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_id": {Type: "string", Description: "Task ID"},
			},
			Required: []string{"task_id"},
		},
	},
	// Sections
	{
		ID:   "todoist:list_sections",
		Name: "list_sections",
		Descriptions: modules.LocalizedText{
			"en-US": "List all sections in a project.",
			"ja-JP": "プロジェクト内のすべてのセクションを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_id": {Type: "string", Description: "Project ID (optional, lists all sections if omitted)"},
			},
		},
	},
	// Labels
	{
		ID:   "todoist:list_labels",
		Name: "list_labels",
		Descriptions: modules.LocalizedText{
			"en-US": "List all labels for the user.",
			"ja-JP": "ユーザーのすべてのラベルを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"list_projects": listProjects,
	"get_project":   getProject,
	"list_tasks":    listTasks,
	"get_task":      getTask,
	"create_task":   createTask,
	"update_task":   updateTask,
	"complete_task": completeTask,
	"reopen_task":   reopenTask,
	"delete_task":   deleteTask,
	"list_sections": listSections,
	"list_labels":   listLabels,
}

// =============================================================================
// Projects
// =============================================================================

func listProjects(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListProjects(ctx, gen.ListProjectsParams{})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res.Results)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func getProject(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetProject(ctx, gen.GetProjectParams{ProjectId: projectID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

// =============================================================================
// Tasks
// =============================================================================

func listTasks(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	p := gen.ListTasksParams{}
	if v, ok := params["project_id"].(string); ok && v != "" {
		p.ProjectId.SetTo(v)
	}
	if v, ok := params["section_id"].(string); ok && v != "" {
		p.SectionId.SetTo(v)
	}
	if v, ok := params["label"].(string); ok && v != "" {
		p.Label.SetTo(v)
	}
	res, err := c.ListTasks(ctx, p)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res.Results)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func getTask(ctx context.Context, params map[string]any) (string, error) {
	taskID, _ := params["task_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetTask(ctx, gen.GetTaskParams{TaskId: taskID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func createTask(ctx context.Context, params map[string]any) (string, error) {
	content, _ := params["content"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	req := gen.CreateTaskReq{Content: content}
	if v, ok := params["description"].(string); ok && v != "" {
		req.Description.SetTo(v)
	}
	if v, ok := params["project_id"].(string); ok && v != "" {
		req.ProjectId.SetTo(v)
	}
	if v, ok := params["section_id"].(string); ok && v != "" {
		req.SectionId.SetTo(v)
	}
	if v, ok := params["parent_id"].(string); ok && v != "" {
		req.ParentId.SetTo(v)
	}
	if v, ok := params["priority"].(float64); ok {
		req.Priority.SetTo(int(v))
	}
	if v, ok := params["due_string"].(string); ok && v != "" {
		req.DueString.SetTo(v)
	}
	if v, ok := params["due_date"].(string); ok && v != "" {
		req.DueDate.SetTo(v)
	}
	if v, ok := params["due_datetime"].(string); ok && v != "" {
		req.DueDatetime.SetTo(v)
	}
	if v, ok := params["labels"].([]interface{}); ok && len(v) > 0 {
		req.Labels = toStringSlice(v)
	}
	if v, ok := params["assignee_id"].(string); ok && v != "" {
		req.AssigneeId.SetTo(v)
	}
	res, err := c.CreateTask(ctx, &req)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func updateTask(ctx context.Context, params map[string]any) (string, error) {
	taskID, _ := params["task_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	req := gen.UpdateTaskReq{}
	if v, ok := params["content"].(string); ok && v != "" {
		req.Content.SetTo(v)
	}
	if v, ok := params["description"].(string); ok {
		req.Description.SetTo(v)
	}
	if v, ok := params["priority"].(float64); ok {
		req.Priority.SetTo(int(v))
	}
	if v, ok := params["due_string"].(string); ok && v != "" {
		req.DueString.SetTo(v)
	}
	if v, ok := params["due_date"].(string); ok && v != "" {
		req.DueDate.SetTo(v)
	}
	if v, ok := params["due_datetime"].(string); ok && v != "" {
		req.DueDatetime.SetTo(v)
	}
	if v, ok := params["labels"].([]interface{}); ok {
		req.Labels = toStringSlice(v)
	}
	if v, ok := params["assignee_id"].(string); ok {
		req.AssigneeId.SetTo(v)
	}
	res, err := c.UpdateTask(ctx, &req, gen.UpdateTaskParams{TaskId: taskID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func completeTask(ctx context.Context, params map[string]any) (string, error) {
	taskID, _ := params["task_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	err = c.CloseTask(ctx, gen.CloseTaskParams{TaskId: taskID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Task completed"}`, nil
}

func reopenTask(ctx context.Context, params map[string]any) (string, error) {
	taskID, _ := params["task_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	err = c.ReopenTask(ctx, gen.ReopenTaskParams{TaskId: taskID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Task reopened"}`, nil
}

func deleteTask(ctx context.Context, params map[string]any) (string, error) {
	taskID, _ := params["task_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	err = c.DeleteTask(ctx, gen.DeleteTaskParams{TaskId: taskID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Task deleted"}`, nil
}

// =============================================================================
// Sections
// =============================================================================

func listSections(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	p := gen.ListSectionsParams{}
	if v, ok := params["project_id"].(string); ok && v != "" {
		p.ProjectId.SetTo(v)
	}
	res, err := c.ListSections(ctx, p)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res.Results)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

// =============================================================================
// Labels
// =============================================================================

func listLabels(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListLabels(ctx, gen.ListLabelsParams{})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res.Results)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

// =============================================================================
// Token Exchange (for OAuth callback)
// =============================================================================

// ExchangeCodeForToken exchanges an authorization code for an access token
func ExchangeCodeForToken(ctx context.Context, code, clientID, clientSecret string) (string, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, "POST", todoistTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token exchange failed: status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResp.AccessToken, nil
}
