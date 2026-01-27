package microsoft_todo

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
	graphAPIBase       = "https://graph.microsoft.com/v1.0"
	microsoftTokenURL  = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	apiVersion         = "v1.0"
	tokenRefreshBuffer = 5 * 60 // Refresh 5 minutes before expiry
)

var client = httpclient.New()

// MicrosoftTodoModule implements the Module interface for Microsoft To Do API
type MicrosoftTodoModule struct{}

// New creates a new MicrosoftTodoModule instance
func New() *MicrosoftTodoModule {
	return &MicrosoftTodoModule{}
}

// Name returns the module name
func (m *MicrosoftTodoModule) Name() string {
	return "microsoft_todo"
}

// Description returns the module description
func (m *MicrosoftTodoModule) Description() string {
	return "Microsoft To Do API - List, create, update, and delete tasks and task lists"
}

// APIVersion returns the Microsoft Graph API version
func (m *MicrosoftTodoModule) APIVersion() string {
	return apiVersion
}

// Tools returns all available tools
func (m *MicrosoftTodoModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *MicrosoftTodoModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for Microsoft To Do)
func (m *MicrosoftTodoModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *MicrosoftTodoModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// Prompts returns all available prompts (none for Microsoft To Do)
func (m *MicrosoftTodoModule) Prompts() []modules.Prompt {
	return nil
}

// GetPrompt generates a prompt with arguments (not implemented)
func (m *MicrosoftTodoModule) GetPrompt(ctx context.Context, name string, args map[string]any) (string, error) {
	return "", fmt.Errorf("prompts not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *store.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[microsoft_todo] No auth context")
		return nil
	}
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "microsoft_todo")
	if err != nil {
		log.Printf("[microsoft_todo] GetModuleToken error: %v", err)
		return nil
	}
	log.Printf("[microsoft_todo] Got credentials: auth_type=%s, has_access_token=%v", credentials.AuthType, credentials.AccessToken != "")

	// Check if token needs refresh (OAuth2 only)
	if credentials.AuthType == store.AuthTypeOAuth2 && credentials.RefreshToken != "" {
		if needsRefresh(credentials) {
			log.Printf("[microsoft_todo] Token expired or expiring soon, refreshing...")
			refreshed, err := refreshToken(ctx, authCtx.UserID, credentials)
			if err != nil {
				log.Printf("[microsoft_todo] Token refresh failed: %v", err)
				// Return original credentials and let the API call fail
				return credentials
			}
			log.Printf("[microsoft_todo] Token refreshed successfully")
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
	// Get OAuth app credentials (client_id, client_secret)
	oauthApp, err := store.GetTokenStore().GetOAuthAppCredentials(ctx, "microsoft")
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth app credentials: %w", err)
	}

	// Exchange refresh token for new access token
	data := url.Values{}
	data.Set("client_id", oauthApp.ClientID)
	data.Set("client_secret", oauthApp.ClientSecret)
	data.Set("refresh_token", creds.RefreshToken)
	data.Set("grant_type", "refresh_token")
	data.Set("scope", "offline_access Tasks.ReadWrite")

	req, err := http.NewRequestWithContext(ctx, "POST", microsoftTokenURL, strings.NewReader(data.Encode()))
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
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Microsoft may return a new refresh token
	refreshTokenToSave := creds.RefreshToken
	if tokenResp.RefreshToken != "" {
		refreshTokenToSave = tokenResp.RefreshToken
	}

	// Update credentials with new access token
	newCreds := &store.Credentials{
		AuthType:     store.AuthTypeOAuth2,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: refreshTokenToSave,
		ExpiresAt:    time.Now().Unix() + tokenResp.ExpiresIn,
	}

	// Save updated credentials to Vault
	err = store.GetTokenStore().UpdateModuleToken(ctx, userID, "microsoft_todo", newCreds)
	if err != nil {
		log.Printf("[microsoft_todo] Failed to save refreshed token: %v", err)
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
		"Accept":       "application/json",
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
	// Lists
	{
		Name:        "list_lists",
		Description: "Get all task lists for the user.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		Name:        "get_list",
		Description: "Get a specific task list by ID.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"list_id": {Type: "string", Description: "The ID of the task list"},
			},
			Required: []string{"list_id"},
		},
	},
	{
		Name:        "create_list",
		Description: "Create a new task list.",
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"display_name": {Type: "string", Description: "The name of the task list"},
			},
			Required: []string{"display_name"},
		},
	},
	{
		Name:        "update_list",
		Description: "Update an existing task list.",
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"list_id":      {Type: "string", Description: "The ID of the task list"},
				"display_name": {Type: "string", Description: "The new name of the task list"},
			},
			Required: []string{"list_id", "display_name"},
		},
	},
	{
		Name:        "delete_list",
		Description: "Delete a task list.",
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"list_id": {Type: "string", Description: "The ID of the task list to delete"},
			},
			Required: []string{"list_id"},
		},
	},
	// Tasks
	{
		Name:        "list_tasks",
		Description: "Get all tasks in a task list.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"list_id": {Type: "string", Description: "The ID of the task list"},
				"filter":  {Type: "string", Description: "OData filter query (e.g., 'status eq \"notStarted\"')"},
				"top":     {Type: "number", Description: "Maximum number of tasks to return (default: 100)"},
			},
			Required: []string{"list_id"},
		},
	},
	{
		Name:        "get_task",
		Description: "Get a specific task by ID.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"list_id": {Type: "string", Description: "The ID of the task list"},
				"task_id": {Type: "string", Description: "The ID of the task"},
			},
			Required: []string{"list_id", "task_id"},
		},
	},
	{
		Name:        "create_task",
		Description: "Create a new task in a task list.",
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"list_id":       {Type: "string", Description: "The ID of the task list"},
				"title":         {Type: "string", Description: "The title of the task"},
				"body":          {Type: "string", Description: "The body/description of the task (plain text)"},
				"importance":    {Type: "string", Description: "Importance level: low, normal, high"},
				"due_date":      {Type: "string", Description: "Due date in YYYY-MM-DD format"},
				"reminder_date": {Type: "string", Description: "Reminder date and time in ISO 8601 format"},
			},
			Required: []string{"list_id", "title"},
		},
	},
	{
		Name:        "update_task",
		Description: "Update an existing task.",
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"list_id":       {Type: "string", Description: "The ID of the task list"},
				"task_id":       {Type: "string", Description: "The ID of the task"},
				"title":         {Type: "string", Description: "The new title of the task"},
				"body":          {Type: "string", Description: "The new body/description of the task"},
				"importance":    {Type: "string", Description: "Importance level: low, normal, high"},
				"status":        {Type: "string", Description: "Status: notStarted, inProgress, completed, waitingOnOthers, deferred"},
				"due_date":      {Type: "string", Description: "Due date in YYYY-MM-DD format"},
				"reminder_date": {Type: "string", Description: "Reminder date and time in ISO 8601 format"},
			},
			Required: []string{"list_id", "task_id"},
		},
	},
	{
		Name:        "complete_task",
		Description: "Mark a task as completed.",
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"list_id": {Type: "string", Description: "The ID of the task list"},
				"task_id": {Type: "string", Description: "The ID of the task"},
			},
			Required: []string{"list_id", "task_id"},
		},
	},
	{
		Name:        "delete_task",
		Description: "Delete a task.",
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"list_id": {Type: "string", Description: "The ID of the task list"},
				"task_id": {Type: "string", Description: "The ID of the task to delete"},
			},
			Required: []string{"list_id", "task_id"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"list_lists":    listLists,
	"get_list":      getList,
	"create_list":   createList,
	"update_list":   updateList,
	"delete_list":   deleteList,
	"list_tasks":    listTasks,
	"get_task":      getTask,
	"create_task":   createTask,
	"update_task":   updateTask,
	"complete_task": completeTask,
	"delete_task":   deleteTask,
}

// =============================================================================
// Lists
// =============================================================================

func listLists(ctx context.Context, params map[string]any) (string, error) {
	endpoint := graphAPIBase + "/me/todo/lists"
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getList(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	endpoint := fmt.Sprintf("%s/me/todo/lists/%s", graphAPIBase, url.PathEscape(listID))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createList(ctx context.Context, params map[string]any) (string, error) {
	displayName, _ := params["display_name"].(string)

	body := map[string]interface{}{
		"displayName": displayName,
	}

	endpoint := graphAPIBase + "/me/todo/lists"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updateList(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	displayName, _ := params["display_name"].(string)

	body := map[string]interface{}{
		"displayName": displayName,
	}

	endpoint := fmt.Sprintf("%s/me/todo/lists/%s", graphAPIBase, url.PathEscape(listID))
	respBody, err := client.DoJSON("PATCH", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteList(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	endpoint := fmt.Sprintf("%s/me/todo/lists/%s", graphAPIBase, url.PathEscape(listID))

	_, err := client.DoJSON("DELETE", endpoint, headers(ctx), nil)
	if err != nil {
		// Check if it's a 204 No Content (success for DELETE)
		if apiErr, ok := err.(*httpclient.APIError); ok && apiErr.StatusCode == 204 {
			return `{"success": true, "message": "List deleted"}`, nil
		}
		return "", err
	}

	return `{"success": true, "message": "List deleted"}`, nil
}

// =============================================================================
// Tasks
// =============================================================================

func listTasks(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)

	query := url.Values{}

	// Filter
	if filter, ok := params["filter"].(string); ok && filter != "" {
		query.Set("$filter", filter)
	}

	// Top (max results)
	top := 100
	if t, ok := params["top"].(float64); ok {
		top = int(t)
	}
	query.Set("$top", fmt.Sprintf("%d", top))

	endpoint := fmt.Sprintf("%s/me/todo/lists/%s/tasks", graphAPIBase, url.PathEscape(listID))
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
	listID, _ := params["list_id"].(string)
	taskID, _ := params["task_id"].(string)

	endpoint := fmt.Sprintf("%s/me/todo/lists/%s/tasks/%s", graphAPIBase, url.PathEscape(listID), url.PathEscape(taskID))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createTask(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	title, _ := params["title"].(string)

	body := map[string]interface{}{
		"title": title,
	}

	// Optional: body content
	if bodyText, ok := params["body"].(string); ok && bodyText != "" {
		body["body"] = map[string]interface{}{
			"content":     bodyText,
			"contentType": "text",
		}
	}

	// Optional: importance
	if importance, ok := params["importance"].(string); ok && importance != "" {
		body["importance"] = importance
	}

	// Optional: due date
	if dueDate, ok := params["due_date"].(string); ok && dueDate != "" {
		body["dueDateTime"] = map[string]interface{}{
			"dateTime": dueDate + "T00:00:00",
			"timeZone": "UTC",
		}
	}

	// Optional: reminder
	if reminderDate, ok := params["reminder_date"].(string); ok && reminderDate != "" {
		body["isReminderOn"] = true
		body["reminderDateTime"] = map[string]interface{}{
			"dateTime": reminderDate,
			"timeZone": "UTC",
		}
	}

	endpoint := fmt.Sprintf("%s/me/todo/lists/%s/tasks", graphAPIBase, url.PathEscape(listID))
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updateTask(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	taskID, _ := params["task_id"].(string)

	body := map[string]interface{}{}

	// Optional: title
	if title, ok := params["title"].(string); ok && title != "" {
		body["title"] = title
	}

	// Optional: body content
	if bodyText, ok := params["body"].(string); ok && bodyText != "" {
		body["body"] = map[string]interface{}{
			"content":     bodyText,
			"contentType": "text",
		}
	}

	// Optional: importance
	if importance, ok := params["importance"].(string); ok && importance != "" {
		body["importance"] = importance
	}

	// Optional: status
	if status, ok := params["status"].(string); ok && status != "" {
		body["status"] = status
	}

	// Optional: due date
	if dueDate, ok := params["due_date"].(string); ok && dueDate != "" {
		body["dueDateTime"] = map[string]interface{}{
			"dateTime": dueDate + "T00:00:00",
			"timeZone": "UTC",
		}
	}

	// Optional: reminder
	if reminderDate, ok := params["reminder_date"].(string); ok && reminderDate != "" {
		body["isReminderOn"] = true
		body["reminderDateTime"] = map[string]interface{}{
			"dateTime": reminderDate,
			"timeZone": "UTC",
		}
	}

	endpoint := fmt.Sprintf("%s/me/todo/lists/%s/tasks/%s", graphAPIBase, url.PathEscape(listID), url.PathEscape(taskID))
	respBody, err := client.DoJSON("PATCH", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func completeTask(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	taskID, _ := params["task_id"].(string)

	body := map[string]interface{}{
		"status": "completed",
	}

	endpoint := fmt.Sprintf("%s/me/todo/lists/%s/tasks/%s", graphAPIBase, url.PathEscape(listID), url.PathEscape(taskID))
	respBody, err := client.DoJSON("PATCH", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteTask(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	taskID, _ := params["task_id"].(string)

	endpoint := fmt.Sprintf("%s/me/todo/lists/%s/tasks/%s", graphAPIBase, url.PathEscape(listID), url.PathEscape(taskID))

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
