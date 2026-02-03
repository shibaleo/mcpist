package ticktick

import (
	"context"
	"fmt"
	"strings"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/store"
)

const (
	ticktickAPIBase = "https://api.ticktick.com/open/v1"
	ticktickVersion = "v1"
)

var client = httpclient.New()

// TickTickModule implements the Module interface for TickTick API
type TickTickModule struct{}

// New creates a new TickTickModule instance
func New() *TickTickModule {
	return &TickTickModule{}
}

// Module descriptions in multiple languages
var moduleDescriptions = modules.LocalizedText{
	"en-US": "TickTick API - Task and project management with creation, updates, and completion tracking",
	"ja-JP": "TickTick API - タスクとプロジェクトの管理（作成、更新、完了追跡）",
}

// Name returns the module name
func (m *TickTickModule) Name() string {
	return "ticktick"
}

// Descriptions returns the module descriptions in all languages
func (m *TickTickModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description for a specific language
func (m *TickTickModule) Description(lang string) string {
	return modules.GetLocalizedText(moduleDescriptions, lang)
}

// APIVersion returns the TickTick API version
func (m *TickTickModule) APIVersion() string {
	return ticktickVersion
}

// Tools returns all available tools
func (m *TickTickModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *TickTickModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for TickTick)
func (m *TickTickModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *TickTickModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *store.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil
	}
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "ticktick")
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

	return map[string]string{
		"Authorization": "Bearer " + creds.AccessToken,
		"Content-Type":  "application/json",
	}
}

// =============================================================================
// Tool Definitions
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	// Project tools
	"list_projects":    listProjects,
	"get_project":      getProject,
	"get_project_data": getProjectData,
	"create_project":   createProject,
	"update_project":   updateProject,
	"delete_project":   deleteProject,
	// Task tools
	"get_task":      getTask,
	"create_task":   createTask,
	"update_task":   updateTask,
	"complete_task": completeTask,
	"delete_task":   deleteTask,
}

var toolDefinitions = []modules.Tool{
	// -------------------------------------------------------------------------
	// Project Tools
	// -------------------------------------------------------------------------
	{
		ID:   "ticktick:list_projects",
		Name: "list_projects",
		Descriptions: modules.LocalizedText{
			"en-US": "List all projects for the user.",
			"ja-JP": "ユーザーのすべてのプロジェクトを一覧表示します。",
		},
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
		Annotations: modules.AnnotateReadOnly,
	},
	{
		ID:   "ticktick:get_project",
		Name: "get_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific project.",
			"ja-JP": "特定のプロジェクトの詳細を取得します。",
		},
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_id": {Type: "string", Description: "Project ID"},
			},
			Required: []string{"project_id"},
		},
		Annotations: modules.AnnotateReadOnly,
	},
	{
		ID:   "ticktick:get_project_data",
		Name: "get_project_data",
		Descriptions: modules.LocalizedText{
			"en-US": "Get project details including all tasks and columns.",
			"ja-JP": "タスクとカラムを含むプロジェクトの詳細データを取得します。",
		},
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_id": {Type: "string", Description: "Project ID"},
			},
			Required: []string{"project_id"},
		},
		Annotations: modules.AnnotateReadOnly,
	},
	{
		ID:   "ticktick:create_project",
		Name: "create_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new project.",
			"ja-JP": "新しいプロジェクトを作成します。",
		},
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"name":      {Type: "string", Description: "Project name"},
				"color":     {Type: "string", Description: "Color code (optional)"},
				"view_mode": {Type: "string", Description: "View mode: list, kanban, timeline (optional)"},
				"kind":      {Type: "string", Description: "Project kind: TASK or NOTE (optional, default: TASK)"},
			},
			Required: []string{"name"},
		},
		Annotations: modules.AnnotateCreate,
	},
	{
		ID:   "ticktick:update_project",
		Name: "update_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing project.",
			"ja-JP": "既存のプロジェクトを更新します。",
		},
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_id": {Type: "string", Description: "Project ID"},
				"name":       {Type: "string", Description: "New project name (optional)"},
				"color":      {Type: "string", Description: "New color code (optional)"},
				"view_mode":  {Type: "string", Description: "View mode: list, kanban, timeline (optional)"},
				"kind":       {Type: "string", Description: "Project kind: TASK or NOTE (optional)"},
			},
			Required: []string{"project_id"},
		},
		Annotations: modules.AnnotateUpdate,
	},
	{
		ID:   "ticktick:delete_project",
		Name: "delete_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a project.",
			"ja-JP": "プロジェクトを削除します。",
		},
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_id": {Type: "string", Description: "Project ID"},
			},
			Required: []string{"project_id"},
		},
		Annotations: modules.AnnotateDelete,
	},

	// -------------------------------------------------------------------------
	// Task Tools
	// -------------------------------------------------------------------------
	{
		ID:   "ticktick:get_task",
		Name: "get_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific task.",
			"ja-JP": "特定のタスクの詳細を取得します。",
		},
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_id": {Type: "string", Description: "Project ID"},
				"task_id":    {Type: "string", Description: "Task ID"},
			},
			Required: []string{"project_id", "task_id"},
		},
		Annotations: modules.AnnotateReadOnly,
	},
	{
		ID:   "ticktick:create_task",
		Name: "create_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new task with optional subtasks, due dates, and reminders.",
			"ja-JP": "サブタスク、期限、リマインダーを指定して新しいタスクを作成します。",
		},
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"title":       {Type: "string", Description: "Task title"},
				"project_id":  {Type: "string", Description: "Project ID to add the task to (optional, defaults to inbox)"},
				"content":     {Type: "string", Description: "Task content/notes (optional)"},
				"desc":        {Type: "string", Description: "Task description (optional)"},
				"is_all_day":  {Type: "boolean", Description: "Whether it's an all-day task (optional)"},
				"start_date":  {Type: "string", Description: "Start date in ISO 8601 format, e.g. 2025-01-01T00:00:00+0000 (optional)"},
				"due_date":    {Type: "string", Description: "Due date in ISO 8601 format, e.g. 2025-01-01T00:00:00+0000 (optional)"},
				"time_zone":   {Type: "string", Description: "Timezone, e.g. Asia/Tokyo (optional)"},
				"reminders":   {Type: "array", Description: "Array of reminder triggers, e.g. ['TRIGGER:P0DT9H0M0S', 'TRIGGER:PT0S'] (optional)"},
				"repeat_flag": {Type: "string", Description: "Recurrence rule in RRULE format (optional)"},
				"priority":    {Type: "number", Description: "Priority: 0 (none), 1 (low), 3 (medium), 5 (high) (optional)"},
				"sort_order":  {Type: "number", Description: "Sort order value (optional)"},
				"items": {Type: "array", Description: `Subtask items as array of objects: [{"title": "subtask1", "status": 0}] (optional). status: 0=normal, 2=completed`},
			},
			Required: []string{"title"},
		},
		Annotations: modules.AnnotateCreate,
	},
	{
		ID:   "ticktick:update_task",
		Name: "update_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing task's properties.",
			"ja-JP": "既存のタスクのプロパティを更新します。",
		},
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_id":     {Type: "string", Description: "Task ID"},
				"project_id":  {Type: "string", Description: "Project ID the task belongs to"},
				"title":       {Type: "string", Description: "New task title (optional)"},
				"content":     {Type: "string", Description: "New task content/notes (optional)"},
				"desc":        {Type: "string", Description: "New task description (optional)"},
				"is_all_day":  {Type: "boolean", Description: "Whether it's an all-day task (optional)"},
				"start_date":  {Type: "string", Description: "Start date in ISO 8601 format (optional)"},
				"due_date":    {Type: "string", Description: "Due date in ISO 8601 format (optional)"},
				"time_zone":   {Type: "string", Description: "Timezone (optional)"},
				"reminders":   {Type: "array", Description: "Array of reminder triggers (optional)"},
				"repeat_flag": {Type: "string", Description: "Recurrence rule in RRULE format (optional)"},
				"priority":    {Type: "number", Description: "Priority: 0 (none), 1 (low), 3 (medium), 5 (high) (optional)"},
				"sort_order":  {Type: "number", Description: "Sort order value (optional)"},
				"items":       {Type: "array", Description: "Subtask items (optional)"},
			},
			Required: []string{"task_id", "project_id"},
		},
		Annotations: modules.AnnotateUpdate,
	},
	{
		ID:   "ticktick:complete_task",
		Name: "complete_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Mark a task as completed.",
			"ja-JP": "タスクを完了にします。",
		},
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_id": {Type: "string", Description: "Project ID"},
				"task_id":    {Type: "string", Description: "Task ID"},
			},
			Required: []string{"project_id", "task_id"},
		},
		Annotations: modules.AnnotateUpdate,
	},
	{
		ID:   "ticktick:delete_task",
		Name: "delete_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a task.",
			"ja-JP": "タスクを削除します。",
		},
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_id": {Type: "string", Description: "Project ID"},
				"task_id":    {Type: "string", Description: "Task ID"},
			},
			Required: []string{"project_id", "task_id"},
		},
		Annotations: modules.AnnotateDelete,
	},
}

// =============================================================================
// Project Handlers
// =============================================================================

func listProjects(ctx context.Context, params map[string]any) (string, error) {
	endpoint := ticktickAPIBase + "/project"
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getProject(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	if projectID == "" {
		return "", fmt.Errorf("project_id is required")
	}
	endpoint := ticktickAPIBase + "/project/" + projectID
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getProjectData(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	if projectID == "" {
		return "", fmt.Errorf("project_id is required")
	}
	endpoint := ticktickAPIBase + "/project/" + projectID + "/data"
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createProject(ctx context.Context, params map[string]any) (string, error) {
	name, _ := params["name"].(string)
	if name == "" {
		return "", fmt.Errorf("name is required")
	}

	body := map[string]any{"name": name}
	if color, ok := params["color"].(string); ok && color != "" {
		body["color"] = color
	}
	if viewMode, ok := params["view_mode"].(string); ok && viewMode != "" {
		body["viewMode"] = viewMode
	}
	if kind, ok := params["kind"].(string); ok && kind != "" {
		body["kind"] = kind
	}

	endpoint := ticktickAPIBase + "/project"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updateProject(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	if projectID == "" {
		return "", fmt.Errorf("project_id is required")
	}

	body := map[string]any{}
	if name, ok := params["name"].(string); ok && name != "" {
		body["name"] = name
	}
	if color, ok := params["color"].(string); ok && color != "" {
		body["color"] = color
	}
	if viewMode, ok := params["view_mode"].(string); ok && viewMode != "" {
		body["viewMode"] = viewMode
	}
	if kind, ok := params["kind"].(string); ok && kind != "" {
		body["kind"] = kind
	}

	endpoint := ticktickAPIBase + "/project/" + projectID
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteProject(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	if projectID == "" {
		return "", fmt.Errorf("project_id is required")
	}

	endpoint := ticktickAPIBase + "/project/" + projectID
	respBody, err := client.DoJSON("DELETE", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	result := strings.TrimSpace(string(respBody))
	if result == "" || result == "null" {
		return `{"deleted": true}`, nil
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Task Handlers
// =============================================================================

func getTask(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	taskID, _ := params["task_id"].(string)
	if projectID == "" || taskID == "" {
		return "", fmt.Errorf("project_id and task_id are required")
	}

	endpoint := ticktickAPIBase + "/project/" + projectID + "/task/" + taskID
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createTask(ctx context.Context, params map[string]any) (string, error) {
	title, _ := params["title"].(string)
	if title == "" {
		return "", fmt.Errorf("title is required")
	}

	body := map[string]any{"title": title}

	// Optional string fields
	for _, field := range []struct{ param, api string }{
		{"project_id", "projectId"},
		{"content", "content"},
		{"desc", "desc"},
		{"start_date", "startDate"},
		{"due_date", "dueDate"},
		{"time_zone", "timeZone"},
		{"repeat_flag", "repeatFlag"},
	} {
		if v, ok := params[field.param].(string); ok && v != "" {
			body[field.api] = v
		}
	}

	// Optional boolean field
	if v, ok := params["is_all_day"].(bool); ok {
		body["isAllDay"] = v
	}

	// Optional number fields
	if v, ok := params["priority"].(float64); ok {
		body["priority"] = int(v)
	}
	if v, ok := params["sort_order"].(float64); ok {
		body["sortOrder"] = int64(v)
	}

	// Optional array fields
	if v, ok := params["reminders"]; ok && v != nil {
		body["reminders"] = v
	}
	if v, ok := params["items"]; ok && v != nil {
		body["items"] = v
	}

	endpoint := ticktickAPIBase + "/task"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updateTask(ctx context.Context, params map[string]any) (string, error) {
	taskID, _ := params["task_id"].(string)
	projectID, _ := params["project_id"].(string)
	if taskID == "" || projectID == "" {
		return "", fmt.Errorf("task_id and project_id are required")
	}

	body := map[string]any{
		"id":        taskID,
		"projectId": projectID,
	}

	// Optional string fields
	for _, field := range []struct{ param, api string }{
		{"title", "title"},
		{"content", "content"},
		{"desc", "desc"},
		{"start_date", "startDate"},
		{"due_date", "dueDate"},
		{"time_zone", "timeZone"},
		{"repeat_flag", "repeatFlag"},
	} {
		if v, ok := params[field.param].(string); ok && v != "" {
			body[field.api] = v
		}
	}

	// Optional boolean field
	if v, ok := params["is_all_day"].(bool); ok {
		body["isAllDay"] = v
	}

	// Optional number fields
	if v, ok := params["priority"].(float64); ok {
		body["priority"] = int(v)
	}
	if v, ok := params["sort_order"].(float64); ok {
		body["sortOrder"] = int64(v)
	}

	// Optional array fields
	if v, ok := params["reminders"]; ok && v != nil {
		body["reminders"] = v
	}
	if v, ok := params["items"]; ok && v != nil {
		body["items"] = v
	}

	endpoint := ticktickAPIBase + "/task/" + taskID
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func completeTask(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	taskID, _ := params["task_id"].(string)
	if projectID == "" || taskID == "" {
		return "", fmt.Errorf("project_id and task_id are required")
	}

	endpoint := ticktickAPIBase + "/project/" + projectID + "/task/" + taskID + "/complete"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	result := strings.TrimSpace(string(respBody))
	if result == "" || result == "null" {
		return `{"completed": true}`, nil
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteTask(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	taskID, _ := params["task_id"].(string)
	if projectID == "" || taskID == "" {
		return "", fmt.Errorf("project_id and task_id are required")
	}

	endpoint := ticktickAPIBase + "/project/" + projectID + "/task/" + taskID
	respBody, err := client.DoJSON("DELETE", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	result := strings.TrimSpace(string(respBody))
	if result == "" || result == "null" {
		return `{"deleted": true}`, nil
	}
	return httpclient.PrettyJSON(respBody), nil
}
