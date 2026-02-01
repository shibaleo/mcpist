package google_tasks

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
	googleTasksAPIBase = "https://tasks.googleapis.com/tasks/v1"
	googleTasksVersion = "v1"
	googleTokenURL     = "https://oauth2.googleapis.com/token"
	tokenRefreshBuffer = 5 * 60 // Refresh 5 minutes before expiry
)

var client = httpclient.New()

// GoogleTasksModule implements the Module interface for Google Tasks API
type GoogleTasksModule struct{}

// New creates a new GoogleTasksModule instance
func New() *GoogleTasksModule {
	return &GoogleTasksModule{}
}

// Module descriptions in multiple languages
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Google Tasks API - List, create, update, and delete tasks",
	"ja-JP": "Google Tasks API - タスクの一覧表示、作成、更新、削除",
}

// Name returns the module name
func (m *GoogleTasksModule) Name() string {
	return "google_tasks"
}

// Descriptions returns the module descriptions in all languages
func (m *GoogleTasksModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description for a specific language
func (m *GoogleTasksModule) Description(lang string) string {
	return modules.GetLocalizedText(moduleDescriptions, lang)
}

// APIVersion returns the Google Tasks API version
func (m *GoogleTasksModule) APIVersion() string {
	return googleTasksVersion
}

// Tools returns all available tools
func (m *GoogleTasksModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *GoogleTasksModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for Google Tasks)
func (m *GoogleTasksModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *GoogleTasksModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// Prompts returns all available prompts (none for Google Tasks)
func (m *GoogleTasksModule) Prompts() []modules.Prompt {
	return nil
}

// GetPrompt generates a prompt with arguments (not implemented)
func (m *GoogleTasksModule) GetPrompt(ctx context.Context, name string, args map[string]any) (string, error) {
	return "", fmt.Errorf("prompts not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *store.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[google_tasks] No auth context")
		return nil
	}
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "google_tasks")
	if err != nil {
		log.Printf("[google_tasks] GetModuleToken error: %v", err)
		return nil
	}
	log.Printf("[google_tasks] Got credentials: auth_type=%s, has_access_token=%v", credentials.AuthType, credentials.AccessToken != "")

	// Check if token needs refresh (OAuth2 only)
	if credentials.AuthType == store.AuthTypeOAuth2 && credentials.RefreshToken != "" {
		if needsRefresh(credentials) {
			log.Printf("[google_tasks] Token expired or expiring soon, refreshing...")
			refreshed, err := refreshToken(ctx, authCtx.UserID, credentials)
			if err != nil {
				log.Printf("[google_tasks] Token refresh failed: %v", err)
				// Return original credentials and let the API call fail
				return credentials
			}
			log.Printf("[google_tasks] Token refreshed successfully")
			return refreshed
		}
	}

	return credentials
}

// needsRefresh checks if the token is expired or will expire soon
func needsRefresh(creds *store.Credentials) bool {
	if creds.ExpiresAt == 0 {
		// No expiry information, assume token is valid
		return false
	}
	now := time.Now().Unix()
	// Refresh if expired or expiring within buffer period
	return now >= (creds.ExpiresAt - tokenRefreshBuffer)
}

// refreshToken exchanges the refresh token for a new access token
func refreshToken(ctx context.Context, userID string, creds *store.Credentials) (*store.Credentials, error) {
	// Get OAuth app credentials (client_id, client_secret) - shared with google_calendar
	oauthApp, err := store.GetTokenStore().GetOAuthAppCredentials(ctx, "google")
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth app credentials: %w", err)
	}

	// Exchange refresh token for new access token
	data := url.Values{}
	data.Set("client_id", oauthApp.ClientID)
	data.Set("client_secret", oauthApp.ClientSecret)
	data.Set("refresh_token", creds.RefreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", googleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Update credentials with new access token
	newCreds := &store.Credentials{
		AuthType:     store.AuthTypeOAuth2,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: creds.RefreshToken, // Keep the same refresh token
		ExpiresAt:    time.Now().Unix() + tokenResp.ExpiresIn,
	}

	// Save updated credentials to Vault - use "google_tasks" as module name
	err = store.GetTokenStore().UpdateModuleToken(ctx, userID, "google_tasks", newCreds)
	if err != nil {
		log.Printf("[google_tasks] Failed to save refreshed token: %v", err)
		// Continue anyway, the token is still valid for this request
	}

	return newCreds, nil
}

func headers(ctx context.Context) map[string]string {
	creds := getCredentials(ctx)
	if creds == nil {
		return map[string]string{}
	}

	h := map[string]string{
		"Accept": "application/json",
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
	// Task Lists
	{
		ID:   "google_tasks:list_task_lists",
		Name: "list_task_lists",
		Descriptions: modules.LocalizedText{
			"en-US": "List all task lists for the user.",
			"ja-JP": "ユーザーのすべてのタスクリストを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "google_tasks:get_task_list",
		Name: "get_task_list",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific task list.",
			"ja-JP": "特定のタスクリストの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_list_id": {Type: "string", Description: "Task list ID"},
			},
			Required: []string{"task_list_id"},
		},
	},
	// Tasks
	{
		ID:   "google_tasks:list_tasks",
		Name: "list_tasks",
		Descriptions: modules.LocalizedText{
			"en-US": "List tasks from a task list.",
			"ja-JP": "タスクリストからタスクを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_list_id":   {Type: "string", Description: "Task list ID. Use '@default' for the default task list."},
				"max_results":    {Type: "number", Description: "Maximum number of tasks to return. Default: 100"},
				"show_completed": {Type: "boolean", Description: "Include completed tasks. Default: true"},
				"show_hidden":    {Type: "boolean", Description: "Include hidden tasks. Default: false"},
				"due_min":        {Type: "string", Description: "Minimum due date (RFC3339 format)"},
				"due_max":        {Type: "string", Description: "Maximum due date (RFC3339 format)"},
			},
			Required: []string{"task_list_id"},
		},
	},
	{
		ID:   "google_tasks:get_task",
		Name: "get_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific task.",
			"ja-JP": "特定のタスクの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_list_id": {Type: "string", Description: "Task list ID"},
				"task_id":      {Type: "string", Description: "Task ID"},
			},
			Required: []string{"task_list_id", "task_id"},
		},
	},
	{
		ID:   "google_tasks:create_task",
		Name: "create_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new task in a task list.",
			"ja-JP": "タスクリストに新しいタスクを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_list_id": {Type: "string", Description: "Task list ID. Use '@default' for the default task list."},
				"title":        {Type: "string", Description: "Task title"},
				"notes":        {Type: "string", Description: "Task notes/description"},
				"due":          {Type: "string", Description: "Due date (RFC3339 format, e.g., '2024-01-15T00:00:00Z')"},
				"parent":       {Type: "string", Description: "Parent task ID for creating subtasks"},
			},
			Required: []string{"task_list_id", "title"},
		},
	},
	{
		ID:   "google_tasks:update_task",
		Name: "update_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing task.",
			"ja-JP": "既存のタスクを更新します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_list_id": {Type: "string", Description: "Task list ID"},
				"task_id":      {Type: "string", Description: "Task ID"},
				"title":        {Type: "string", Description: "New task title"},
				"notes":        {Type: "string", Description: "New task notes"},
				"due":          {Type: "string", Description: "New due date (RFC3339 format)"},
				"status":       {Type: "string", Description: "Task status: 'needsAction' or 'completed'"},
			},
			Required: []string{"task_list_id", "task_id"},
		},
	},
	{
		ID:   "google_tasks:delete_task",
		Name: "delete_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a task from a task list.",
			"ja-JP": "タスクリストからタスクを削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_list_id": {Type: "string", Description: "Task list ID"},
				"task_id":      {Type: "string", Description: "Task ID"},
			},
			Required: []string{"task_list_id", "task_id"},
		},
	},
	{
		ID:   "google_tasks:complete_task",
		Name: "complete_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Mark a task as completed or uncompleted.",
			"ja-JP": "タスクを完了または未完了としてマークします。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_list_id": {Type: "string", Description: "Task list ID"},
				"task_id":      {Type: "string", Description: "Task ID"},
				"completed":    {Type: "boolean", Description: "Set to true to mark as completed, false to mark as needs action"},
			},
			Required: []string{"task_list_id", "task_id", "completed"},
		},
	},
	{
		ID:   "google_tasks:clear_completed",
		Name: "clear_completed",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete all completed tasks from a task list permanently.",
			"ja-JP": "タスクリストの完了済みタスクをすべて完全に削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_list_id": {Type: "string", Description: "Task list ID"},
			},
			Required: []string{"task_list_id"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"list_task_lists": listTaskLists,
	"get_task_list":   getTaskList,
	"list_tasks":      listTasks,
	"get_task":        getTask,
	"create_task":     createTask,
	"update_task":     updateTask,
	"delete_task":     deleteTask,
	"complete_task":   completeTask,
	"clear_completed": clearCompleted,
}

// =============================================================================
// Task Lists
// =============================================================================

func listTaskLists(ctx context.Context, params map[string]any) (string, error) {
	endpoint := googleTasksAPIBase + "/users/@me/lists"
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getTaskList(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)
	endpoint := fmt.Sprintf("%s/users/@me/lists/%s", googleTasksAPIBase, url.PathEscape(taskListID))
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
	taskListID, _ := params["task_list_id"].(string)

	query := url.Values{}

	// Max results
	maxResults := 100
	if mr, ok := params["max_results"].(float64); ok {
		maxResults = int(mr)
	}
	query.Set("maxResults", fmt.Sprintf("%d", maxResults))

	// Show completed
	showCompleted := true
	if sc, ok := params["show_completed"].(bool); ok {
		showCompleted = sc
	}
	query.Set("showCompleted", fmt.Sprintf("%t", showCompleted))

	// Show hidden
	showHidden := false
	if sh, ok := params["show_hidden"].(bool); ok {
		showHidden = sh
	}
	query.Set("showHidden", fmt.Sprintf("%t", showHidden))

	// Due date filters
	if dueMin, ok := params["due_min"].(string); ok && dueMin != "" {
		query.Set("dueMin", dueMin)
	}
	if dueMax, ok := params["due_max"].(string); ok && dueMax != "" {
		query.Set("dueMax", dueMax)
	}

	endpoint := fmt.Sprintf("%s/lists/%s/tasks?%s", googleTasksAPIBase, url.PathEscape(taskListID), query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getTask(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)
	taskID, _ := params["task_id"].(string)

	endpoint := fmt.Sprintf("%s/lists/%s/tasks/%s", googleTasksAPIBase, url.PathEscape(taskListID), url.PathEscape(taskID))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createTask(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)
	title, _ := params["title"].(string)

	// Build task body
	task := map[string]interface{}{
		"title": title,
	}

	if notes, ok := params["notes"].(string); ok && notes != "" {
		task["notes"] = notes
	}
	if due, ok := params["due"].(string); ok && due != "" {
		task["due"] = due
	}

	// Build endpoint with optional parent parameter
	endpoint := fmt.Sprintf("%s/lists/%s/tasks", googleTasksAPIBase, url.PathEscape(taskListID))
	if parent, ok := params["parent"].(string); ok && parent != "" {
		endpoint += "?parent=" + url.QueryEscape(parent)
	}

	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), task)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updateTask(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)
	taskID, _ := params["task_id"].(string)

	// First, get the existing task
	getEndpoint := fmt.Sprintf("%s/lists/%s/tasks/%s", googleTasksAPIBase, url.PathEscape(taskListID), url.PathEscape(taskID))
	existingBody, err := client.DoJSON("GET", getEndpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	var task map[string]interface{}
	if err := json.Unmarshal(existingBody, &task); err != nil {
		return "", fmt.Errorf("failed to parse existing task: %w", err)
	}

	// Update fields
	if title, ok := params["title"].(string); ok && title != "" {
		task["title"] = title
	}
	if notes, ok := params["notes"].(string); ok {
		task["notes"] = notes
	}
	if due, ok := params["due"].(string); ok {
		if due == "" {
			delete(task, "due")
		} else {
			task["due"] = due
		}
	}
	if status, ok := params["status"].(string); ok && status != "" {
		task["status"] = status
		if status == "completed" {
			task["completed"] = time.Now().UTC().Format(time.RFC3339)
		} else {
			delete(task, "completed")
		}
	}

	// PATCH to update
	respBody, err := client.DoJSON("PATCH", getEndpoint, headers(ctx), task)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteTask(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)
	taskID, _ := params["task_id"].(string)

	endpoint := fmt.Sprintf("%s/lists/%s/tasks/%s", googleTasksAPIBase, url.PathEscape(taskListID), url.PathEscape(taskID))

	// DoJSON handles DELETE requests - Google Tasks API returns 204 No Content on success
	_, err := client.DoJSON("DELETE", endpoint, headers(ctx), nil)
	if err != nil {
		// Check if it's a 204 No Content (success for DELETE)
		if apiErr, ok := err.(*httpclient.APIError); ok && apiErr.StatusCode == 204 {
			return `{"success": true, "message": "Task deleted"}`, nil
		}
		return "", err
	}

	return `{"success": true, "message": "Task deleted"}`, nil
}

func completeTask(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)
	taskID, _ := params["task_id"].(string)
	completed, _ := params["completed"].(bool)

	// Get existing task
	getEndpoint := fmt.Sprintf("%s/lists/%s/tasks/%s", googleTasksAPIBase, url.PathEscape(taskListID), url.PathEscape(taskID))
	existingBody, err := client.DoJSON("GET", getEndpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	var task map[string]interface{}
	if err := json.Unmarshal(existingBody, &task); err != nil {
		return "", fmt.Errorf("failed to parse existing task: %w", err)
	}

	// Update status
	if completed {
		task["status"] = "completed"
		task["completed"] = time.Now().UTC().Format(time.RFC3339)
	} else {
		task["status"] = "needsAction"
		delete(task, "completed")
	}

	// PATCH to update
	respBody, err := client.DoJSON("PATCH", getEndpoint, headers(ctx), task)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func clearCompleted(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)

	// First, list all completed tasks
	listEndpoint := fmt.Sprintf("%s/lists/%s/tasks?showCompleted=true&showHidden=false", googleTasksAPIBase, url.PathEscape(taskListID))
	respBody, err := client.DoJSON("GET", listEndpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	var taskList struct {
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(respBody, &taskList); err != nil {
		return "", fmt.Errorf("failed to parse task list: %w", err)
	}

	// Delete each completed task
	deletedCount := 0
	for _, task := range taskList.Items {
		if task.Status == "completed" {
			deleteEndpoint := fmt.Sprintf("%s/lists/%s/tasks/%s", googleTasksAPIBase, url.PathEscape(taskListID), url.PathEscape(task.ID))
			_, err := client.DoJSON("DELETE", deleteEndpoint, headers(ctx), nil)
			if err != nil {
				// Check if it's a 204 No Content (success)
				if apiErr, ok := err.(*httpclient.APIError); ok && apiErr.StatusCode == 204 {
					deletedCount++
					continue
				}
				// Log error but continue deleting other tasks
				log.Printf("[google_tasks] Failed to delete task %s: %v", task.ID, err)
				continue
			}
			deletedCount++
		}
	}

	result := map[string]interface{}{
		"success":       true,
		"deleted_count": deletedCount,
		"message":       fmt.Sprintf("Deleted %d completed tasks", deletedCount),
	}
	resultBytes, _ := json.Marshal(result)
	return httpclient.PrettyJSON(resultBytes), nil
}
