package todoist

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/store"
)

const (
	todoistAPIBase     = "https://api.todoist.com/rest/v2"
	todoistSyncAPIBase = "https://api.todoist.com/sync/v9"
	todoistVersion     = "v2"
	todoistTokenURL    = "https://todoist.com/oauth/access_token"
	tokenRefreshBuffer = 5 * 60 // Refresh 5 minutes before expiry (if applicable)
)

var client = httpclient.New()

// TodoistModule implements the Module interface for Todoist API
type TodoistModule struct{}

// New creates a new TodoistModule instance
func New() *TodoistModule {
	return &TodoistModule{}
}

// Module descriptions in multiple languages
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

// Description returns the module description for a specific language
func (m *TodoistModule) Description(lang string) string {
	return modules.GetLocalizedText(moduleDescriptions, lang)
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

// Resources returns all available resources (none for Todoist)
func (m *TodoistModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *TodoistModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *store.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[todoist] No auth context")
		return nil
	}
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "todoist")
	if err != nil {
		log.Printf("[todoist] GetModuleToken error: %v", err)
		return nil
	}
	log.Printf("[todoist] Got credentials: auth_type=%s, has_access_token=%v", credentials.AuthType, credentials.AccessToken != "")

	// Note: Todoist OAuth2 does NOT provide refresh tokens by default
	// Tokens are long-lived but may need re-authorization if revoked

	return credentials
}

func headers(ctx context.Context) map[string]string {
	creds := getCredentials(ctx)
	if creds == nil {
		return map[string]string{}
	}

	h := map[string]string{
		"Content-Type": "application/json",
	}

	// OAuth2 uses Bearer token
	if creds.AuthType == store.AuthTypeOAuth2 {
		h["Authorization"] = "Bearer " + creds.AccessToken
	}

	return h
}

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
			Type:       "object",
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
				"filter":     {Type: "string", Description: "Filter string (e.g., 'today', 'overdue', 'priority 1')"},
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
				"content":     {Type: "string", Description: "Task content (required)"},
				"description": {Type: "string", Description: "Task description"},
				"project_id":  {Type: "string", Description: "Project ID (default: Inbox)"},
				"section_id":  {Type: "string", Description: "Section ID"},
				"parent_id":   {Type: "string", Description: "Parent task ID for subtasks"},
				"priority":    {Type: "number", Description: "Priority: 1 (normal) to 4 (urgent)"},
				"due_string":  {Type: "string", Description: "Due date in natural language (e.g., 'tomorrow', 'next Monday')"},
				"due_date":    {Type: "string", Description: "Due date (YYYY-MM-DD format)"},
				"due_datetime": {Type: "string", Description: "Due datetime (RFC3339 format)"},
				"labels":      {Type: "array", Description: "Array of label names"},
				"assignee_id": {Type: "string", Description: "Assignee user ID (for shared projects)"},
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
				"task_id":     {Type: "string", Description: "Task ID (required)"},
				"content":     {Type: "string", Description: "New task content"},
				"description": {Type: "string", Description: "New task description"},
				"priority":    {Type: "number", Description: "Priority: 1 (normal) to 4 (urgent)"},
				"due_string":  {Type: "string", Description: "Due date in natural language"},
				"due_date":    {Type: "string", Description: "Due date (YYYY-MM-DD format)"},
				"due_datetime": {Type: "string", Description: "Due datetime (RFC3339 format)"},
				"labels":      {Type: "array", Description: "Array of label names"},
				"assignee_id": {Type: "string", Description: "Assignee user ID"},
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
	// Quick Add
	{
		ID:   "todoist:quick_add",
		Name: "quick_add",
		Descriptions: modules.LocalizedText{
			"en-US": "Add a task using natural language (Todoist Quick Add syntax). Supports #project, @label, p1-p4 priority, dates.",
			"ja-JP": "自然言語でタスクを追加します（Todoistクイック追加構文）。#プロジェクト、@ラベル、p1-p4優先度、日付をサポート。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"text": {Type: "string", Description: "Quick add text (e.g., 'Buy milk #Shopping @errands tomorrow p2')"},
			},
			Required: []string{"text"},
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
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"list_projects":  listProjects,
	"get_project":    getProject,
	"list_tasks":     listTasks,
	"get_task":       getTask,
	"create_task":    createTask,
	"update_task":    updateTask,
	"complete_task":  completeTask,
	"reopen_task":    reopenTask,
	"delete_task":    deleteTask,
	"quick_add":      quickAdd,
	"list_sections":  listSections,
	"list_labels":    listLabels,
}

// =============================================================================
// Projects
// =============================================================================

func listProjects(ctx context.Context, params map[string]any) (string, error) {
	endpoint := todoistAPIBase + "/projects"
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getProject(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	endpoint := fmt.Sprintf("%s/projects/%s", todoistAPIBase, url.PathEscape(projectID))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Tasks
// =============================================================================

func listTasks(ctx context.Context, params map[string]any) (string, error) {
	query := url.Values{}

	if projectID, ok := params["project_id"].(string); ok && projectID != "" {
		query.Set("project_id", projectID)
	}
	if sectionID, ok := params["section_id"].(string); ok && sectionID != "" {
		query.Set("section_id", sectionID)
	}
	if label, ok := params["label"].(string); ok && label != "" {
		query.Set("label", label)
	}
	if filter, ok := params["filter"].(string); ok && filter != "" {
		query.Set("filter", filter)
	}

	endpoint := todoistAPIBase + "/tasks"
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getTask(ctx context.Context, params map[string]any) (string, error) {
	taskID, _ := params["task_id"].(string)
	endpoint := fmt.Sprintf("%s/tasks/%s", todoistAPIBase, url.PathEscape(taskID))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createTask(ctx context.Context, params map[string]any) (string, error) {
	content, _ := params["content"].(string)

	task := map[string]interface{}{
		"content": content,
	}

	if description, ok := params["description"].(string); ok && description != "" {
		task["description"] = description
	}
	if projectID, ok := params["project_id"].(string); ok && projectID != "" {
		task["project_id"] = projectID
	}
	if sectionID, ok := params["section_id"].(string); ok && sectionID != "" {
		task["section_id"] = sectionID
	}
	if parentID, ok := params["parent_id"].(string); ok && parentID != "" {
		task["parent_id"] = parentID
	}
	if priority, ok := params["priority"].(float64); ok {
		task["priority"] = int(priority)
	}
	if dueString, ok := params["due_string"].(string); ok && dueString != "" {
		task["due_string"] = dueString
	}
	if dueDate, ok := params["due_date"].(string); ok && dueDate != "" {
		task["due_date"] = dueDate
	}
	if dueDatetime, ok := params["due_datetime"].(string); ok && dueDatetime != "" {
		task["due_datetime"] = dueDatetime
	}
	if labels, ok := params["labels"].([]interface{}); ok && len(labels) > 0 {
		labelStrings := make([]string, len(labels))
		for i, l := range labels {
			labelStrings[i], _ = l.(string)
		}
		task["labels"] = labelStrings
	}
	if assigneeID, ok := params["assignee_id"].(string); ok && assigneeID != "" {
		task["assignee_id"] = assigneeID
	}

	endpoint := todoistAPIBase + "/tasks"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), task)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updateTask(ctx context.Context, params map[string]any) (string, error) {
	taskID, _ := params["task_id"].(string)

	task := map[string]interface{}{}

	if content, ok := params["content"].(string); ok && content != "" {
		task["content"] = content
	}
	if description, ok := params["description"].(string); ok {
		task["description"] = description
	}
	if priority, ok := params["priority"].(float64); ok {
		task["priority"] = int(priority)
	}
	if dueString, ok := params["due_string"].(string); ok && dueString != "" {
		task["due_string"] = dueString
	}
	if dueDate, ok := params["due_date"].(string); ok && dueDate != "" {
		task["due_date"] = dueDate
	}
	if dueDatetime, ok := params["due_datetime"].(string); ok && dueDatetime != "" {
		task["due_datetime"] = dueDatetime
	}
	if labels, ok := params["labels"].([]interface{}); ok {
		labelStrings := make([]string, len(labels))
		for i, l := range labels {
			labelStrings[i], _ = l.(string)
		}
		task["labels"] = labelStrings
	}
	if assigneeID, ok := params["assignee_id"].(string); ok {
		task["assignee_id"] = assigneeID
	}

	endpoint := fmt.Sprintf("%s/tasks/%s", todoistAPIBase, url.PathEscape(taskID))
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), task)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func completeTask(ctx context.Context, params map[string]any) (string, error) {
	taskID, _ := params["task_id"].(string)
	endpoint := fmt.Sprintf("%s/tasks/%s/close", todoistAPIBase, url.PathEscape(taskID))

	_, err := client.DoJSON("POST", endpoint, headers(ctx), nil)
	if err != nil {
		// Check if it's a 204 No Content (success)
		if apiErr, ok := err.(*httpclient.APIError); ok && apiErr.StatusCode == 204 {
			return `{"success": true, "message": "Task completed"}`, nil
		}
		return "", err
	}
	return `{"success": true, "message": "Task completed"}`, nil
}

func reopenTask(ctx context.Context, params map[string]any) (string, error) {
	taskID, _ := params["task_id"].(string)
	endpoint := fmt.Sprintf("%s/tasks/%s/reopen", todoistAPIBase, url.PathEscape(taskID))

	_, err := client.DoJSON("POST", endpoint, headers(ctx), nil)
	if err != nil {
		// Check if it's a 204 No Content (success)
		if apiErr, ok := err.(*httpclient.APIError); ok && apiErr.StatusCode == 204 {
			return `{"success": true, "message": "Task reopened"}`, nil
		}
		return "", err
	}
	return `{"success": true, "message": "Task reopened"}`, nil
}

func deleteTask(ctx context.Context, params map[string]any) (string, error) {
	taskID, _ := params["task_id"].(string)
	endpoint := fmt.Sprintf("%s/tasks/%s", todoistAPIBase, url.PathEscape(taskID))

	_, err := client.DoJSON("DELETE", endpoint, headers(ctx), nil)
	if err != nil {
		// Check if it's a 204 No Content (success)
		if apiErr, ok := err.(*httpclient.APIError); ok && apiErr.StatusCode == 204 {
			return `{"success": true, "message": "Task deleted"}`, nil
		}
		return "", err
	}
	return `{"success": true, "message": "Task deleted"}`, nil
}

// =============================================================================
// Quick Add (Sync API)
// =============================================================================

func quickAdd(ctx context.Context, params map[string]any) (string, error) {
	text, _ := params["text"].(string)

	// Quick Add uses the Sync API
	endpoint := todoistSyncAPIBase + "/quick/add"

	// Quick add uses form data, not JSON
	data := url.Values{}
	data.Set("text", text)

	creds := getCredentials(ctx)
	if creds == nil {
		return "", fmt.Errorf("no credentials available")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("quick add request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("quick add failed: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	resultBytes, _ := json.Marshal(result)
	return httpclient.PrettyJSON(resultBytes), nil
}

// =============================================================================
// Sections
// =============================================================================

func listSections(ctx context.Context, params map[string]any) (string, error) {
	endpoint := todoistAPIBase + "/sections"

	if projectID, ok := params["project_id"].(string); ok && projectID != "" {
		endpoint += "?project_id=" + url.QueryEscape(projectID)
	}

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Labels
// =============================================================================

func listLabels(ctx context.Context, params map[string]any) (string, error) {
	endpoint := todoistAPIBase + "/labels"
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Token Exchange (for OAuth callback)
// =============================================================================

// ExchangeCodeForToken exchanges an authorization code for an access token
// Note: This is typically called from the Console OAuth callback, not directly from MCP
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

// Unused import workaround
var _ = time.Now
