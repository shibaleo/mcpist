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

	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/store"
	"mcpist/server/pkg/microsofttodoapi"
	gen "mcpist/server/pkg/microsofttodoapi/gen"
)

const (
	microsoftTokenURL  = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	apiVersion         = "v1.0"
	tokenRefreshBuffer = 5 * 60 // Refresh 5 minutes before expiry
)

// MicrosoftTodoModule implements the Module interface for Microsoft To Do API
type MicrosoftTodoModule struct{}

// New creates a new MicrosoftTodoModule instance
func New() *MicrosoftTodoModule {
	return &MicrosoftTodoModule{}
}

var moduleDescriptions = modules.LocalizedText{
	"en-US": "Microsoft To Do API - List, create, update, and delete tasks and task lists",
	"ja-JP": "Microsoft To Do API - タスクとタスクリストの一覧表示、作成、更新、削除",
}

func (m *MicrosoftTodoModule) Name() string                        { return "microsoft_todo" }
func (m *MicrosoftTodoModule) Descriptions() modules.LocalizedText { return moduleDescriptions }
func (m *MicrosoftTodoModule) Description(lang string) string {
	return modules.GetLocalizedText(moduleDescriptions, lang)
}
func (m *MicrosoftTodoModule) APIVersion() string          { return apiVersion }
func (m *MicrosoftTodoModule) Tools() []modules.Tool       { return toolDefinitions }
func (m *MicrosoftTodoModule) Resources() []modules.Resource { return nil }
func (m *MicrosoftTodoModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

func (m *MicrosoftTodoModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format.
// Implements modules.CompactConverter interface.
func (m *MicrosoftTodoModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
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

	if credentials.AuthType == store.AuthTypeOAuth2 && credentials.RefreshToken != "" {
		if needsRefresh(credentials) {
			log.Printf("[microsoft_todo] Token expired or expiring soon, refreshing...")
			refreshed, err := refreshToken(ctx, authCtx.UserID, credentials)
			if err != nil {
				log.Printf("[microsoft_todo] Token refresh failed: %v", err)
				return credentials
			}
			log.Printf("[microsoft_todo] Token refreshed successfully")
			return refreshed
		}
	}
	return credentials
}

func needsRefresh(creds *store.Credentials) bool {
	if creds.ExpiresAt == 0 {
		return false
	}
	return time.Now().Unix() >= (int64(creds.ExpiresAt) - tokenRefreshBuffer)
}

func refreshToken(ctx context.Context, userID string, creds *store.Credentials) (*store.Credentials, error) {
	oauthApp, err := store.GetTokenStore().GetOAuthAppCredentials(ctx, "microsoft")
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth app credentials: %w", err)
	}

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
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	refreshTokenToSave := creds.RefreshToken
	if tokenResp.RefreshToken != "" {
		refreshTokenToSave = tokenResp.RefreshToken
	}

	newCreds := &store.Credentials{
		AuthType:     store.AuthTypeOAuth2,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: refreshTokenToSave,
		ExpiresAt:    store.FlexibleTime(time.Now().Unix() + tokenResp.ExpiresIn),
	}

	if err := store.GetTokenStore().UpdateModuleToken(ctx, userID, "microsoft_todo", newCreds); err != nil {
		log.Printf("[microsoft_todo] Failed to save refreshed token: %v", err)
	}
	return newCreds, nil
}

// =============================================================================
// ogen client helpers
// =============================================================================

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return microsofttodoapi.NewClient(creds.AccessToken)
}

var toJSON = modules.ToJSON

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	{
		ID:   "microsoft_todo:list_lists",
		Name: "list_lists",
		Descriptions: modules.LocalizedText{
			"en-US": "Get all task lists for the user.",
			"ja-JP": "ユーザーのすべてのタスクリストを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "microsoft_todo:get_list",
		Name: "get_list",
		Descriptions: modules.LocalizedText{
			"en-US": "Get a specific task list by ID.",
			"ja-JP": "IDで特定のタスクリストを取得します。",
		},
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
		ID:   "microsoft_todo:create_list",
		Name: "create_list",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new task list.",
			"ja-JP": "新しいタスクリストを作成します。",
		},
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
		ID:   "microsoft_todo:update_list",
		Name: "update_list",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing task list.",
			"ja-JP": "既存のタスクリストを更新します。",
		},
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
		ID:   "microsoft_todo:delete_list",
		Name: "delete_list",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a task list.",
			"ja-JP": "タスクリストを削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"list_id": {Type: "string", Description: "The ID of the task list to delete"},
			},
			Required: []string{"list_id"},
		},
	},
	{
		ID:   "microsoft_todo:list_tasks",
		Name: "list_tasks",
		Descriptions: modules.LocalizedText{
			"en-US": "Get all tasks in a task list.",
			"ja-JP": "タスクリスト内のすべてのタスクを取得します。",
		},
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
		ID:   "microsoft_todo:get_task",
		Name: "get_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Get a specific task by ID.",
			"ja-JP": "IDで特定のタスクを取得します。",
		},
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
		ID:   "microsoft_todo:create_task",
		Name: "create_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new task in a task list.",
			"ja-JP": "タスクリストに新しいタスクを作成します。",
		},
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
		ID:   "microsoft_todo:update_task",
		Name: "update_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing task.",
			"ja-JP": "既存のタスクを更新します。",
		},
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
		ID:   "microsoft_todo:complete_task",
		Name: "complete_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Mark a task as completed.",
			"ja-JP": "タスクを完了としてマークします。",
		},
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
		ID:   "microsoft_todo:delete_task",
		Name: "delete_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a task.",
			"ja-JP": "タスクを削除します。",
		},
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
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListLists(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(res.Value)
}

func getList(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetList(ctx, gen.GetListParams{ListId: listID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func createList(ctx context.Context, params map[string]any) (string, error) {
	displayName, _ := params["display_name"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.CreateList(ctx, &gen.CreateTaskListReq{DisplayName: displayName})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func updateList(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	displayName, _ := params["display_name"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	req := &gen.UpdateTaskListReq{}
	req.DisplayName.SetTo(displayName)
	res, err := c.UpdateList(ctx, req, gen.UpdateListParams{ListId: listID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func deleteList(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	if err := c.DeleteList(ctx, gen.DeleteListParams{ListId: listID}); err != nil {
		return "", err
	}
	return `{"success":true,"message":"List deleted"}`, nil
}

// =============================================================================
// Tasks
// =============================================================================

func listTasks(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	p := gen.ListTasksParams{ListId: listID}
	if filter, ok := params["filter"].(string); ok && filter != "" {
		p.Filter.SetTo(filter)
	}
	if top, ok := params["top"].(float64); ok {
		p.Top.SetTo(int(top))
	}
	res, err := c.ListTasks(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res.Value)
}

func getTask(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	taskID, _ := params["task_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetTask(ctx, gen.GetTaskParams{ListId: listID, TaskId: taskID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func createTask(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	title, _ := params["title"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	req := &gen.CreateTaskReq{Title: title}
	if body, ok := params["body"].(string); ok && body != "" {
		req.Body.SetTo(gen.ItemBody{
			Content:     gen.NewOptString(body),
			ContentType: gen.NewOptString("text"),
		})
	}
	if importance, ok := params["importance"].(string); ok && importance != "" {
		req.Importance.SetTo(importance)
	}
	if dueDate, ok := params["due_date"].(string); ok && dueDate != "" {
		req.DueDateTime.SetTo(gen.DateTimeTimeZone{
			DateTime: gen.NewOptString(dueDate + "T00:00:00"),
			TimeZone: gen.NewOptString("UTC"),
		})
	}
	if reminderDate, ok := params["reminder_date"].(string); ok && reminderDate != "" {
		req.IsReminderOn.SetTo(true)
		req.ReminderDateTime.SetTo(gen.DateTimeTimeZone{
			DateTime: gen.NewOptString(reminderDate),
			TimeZone: gen.NewOptString("UTC"),
		})
	}
	res, err := c.CreateTask(ctx, req, gen.CreateTaskParams{ListId: listID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func updateTask(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	taskID, _ := params["task_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	req := &gen.UpdateTaskReq{}
	if title, ok := params["title"].(string); ok && title != "" {
		req.Title.SetTo(title)
	}
	if body, ok := params["body"].(string); ok && body != "" {
		req.Body.SetTo(gen.ItemBody{
			Content:     gen.NewOptString(body),
			ContentType: gen.NewOptString("text"),
		})
	}
	if importance, ok := params["importance"].(string); ok && importance != "" {
		req.Importance.SetTo(importance)
	}
	if status, ok := params["status"].(string); ok && status != "" {
		req.Status.SetTo(status)
	}
	if dueDate, ok := params["due_date"].(string); ok && dueDate != "" {
		req.DueDateTime.SetTo(gen.DateTimeTimeZone{
			DateTime: gen.NewOptString(dueDate + "T00:00:00"),
			TimeZone: gen.NewOptString("UTC"),
		})
	}
	if reminderDate, ok := params["reminder_date"].(string); ok && reminderDate != "" {
		req.IsReminderOn.SetTo(true)
		req.ReminderDateTime.SetTo(gen.DateTimeTimeZone{
			DateTime: gen.NewOptString(reminderDate),
			TimeZone: gen.NewOptString("UTC"),
		})
	}
	res, err := c.UpdateTask(ctx, req, gen.UpdateTaskParams{ListId: listID, TaskId: taskID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func completeTask(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	taskID, _ := params["task_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	req := &gen.UpdateTaskReq{}
	req.Status.SetTo("completed")
	res, err := c.UpdateTask(ctx, req, gen.UpdateTaskParams{ListId: listID, TaskId: taskID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func deleteTask(ctx context.Context, params map[string]any) (string, error) {
	listID, _ := params["list_id"].(string)
	taskID, _ := params["task_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	if err := c.DeleteTask(ctx, gen.DeleteTaskParams{ListId: listID, TaskId: taskID}); err != nil {
		return "", err
	}
	return `{"success":true,"message":"Task deleted"}`, nil
}
