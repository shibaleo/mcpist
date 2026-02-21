package ticktick

import (
	"context"
	"fmt"
	"log"

	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/broker"
	"mcpist/server/pkg/ticktickapi"
	gen "mcpist/server/pkg/ticktickapi/gen"
)

const ticktickVersion = "v1"

// TickTickModule implements the Module interface for TickTick API
type TickTickModule struct{}

// New creates a new TickTickModule instance
func New() *TickTickModule {
	return &TickTickModule{}
}

// Module descriptions
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

// Description returns the module description (English)
func (m *TickTickModule) Description() string {
	return moduleDescriptions["en-US"]
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

// ToCompact converts JSON result to compact format.
// Implements modules.CompactConverter interface.
func (m *TickTickModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
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
// ogen client helpers
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[ticktick] No auth context")
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "ticktick")
	if err != nil {
		log.Printf("[ticktick] GetModuleToken error: %v", err)
		return nil
	}
	return credentials
}

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return ticktickapi.NewClient(creds.AccessToken)
}

var toJSON = modules.ToJSON
var toStringSlice = modules.ToStringSlice

// =============================================================================
// Tool Definitions
// =============================================================================

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
				"items":       {Type: "array", Description: `Subtask items as array of objects: [{"title": "subtask1", "status": 0}] (optional). status: 0=normal, 2=completed`},
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
// Tool Handlers
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

// =============================================================================
// Project Handlers
// =============================================================================

func listProjects(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListProjects(ctx)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
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

func getProjectData(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetProjectData(ctx, gen.GetProjectDataParams{ProjectId: projectID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func createProject(ctx context.Context, params map[string]any) (string, error) {
	name, _ := params["name"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	req := gen.CreateProjectReq{Name: name}
	if v, ok := params["color"].(string); ok && v != "" {
		req.Color.SetTo(v)
	}
	if v, ok := params["view_mode"].(string); ok && v != "" {
		req.ViewMode.SetTo(v)
	}
	if v, ok := params["kind"].(string); ok && v != "" {
		req.Kind.SetTo(v)
	}
	res, err := c.CreateProject(ctx, &req)
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func updateProject(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	req := gen.UpdateProjectReq{}
	if v, ok := params["name"].(string); ok && v != "" {
		req.Name.SetTo(v)
	}
	if v, ok := params["color"].(string); ok && v != "" {
		req.Color.SetTo(v)
	}
	if v, ok := params["view_mode"].(string); ok && v != "" {
		req.ViewMode.SetTo(v)
	}
	if v, ok := params["kind"].(string); ok && v != "" {
		req.Kind.SetTo(v)
	}
	res, err := c.UpdateProject(ctx, &req, gen.UpdateProjectParams{ProjectId: projectID})
	if err != nil {
		return "", err
	}
	jsonStr, err := toJSON(res)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func deleteProject(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	err = c.DeleteProject(ctx, gen.DeleteProjectParams{ProjectId: projectID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Project deleted"}`, nil
}

// =============================================================================
// Task Handlers
// =============================================================================

func getTask(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	taskID, _ := params["task_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetTask(ctx, gen.GetTaskParams{ProjectId: projectID, TaskId: taskID})
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
	title, _ := params["title"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	req := gen.CreateTaskReq{Title: title}
	if v, ok := params["project_id"].(string); ok && v != "" {
		req.ProjectId.SetTo(v)
	}
	if v, ok := params["content"].(string); ok && v != "" {
		req.Content.SetTo(v)
	}
	if v, ok := params["desc"].(string); ok && v != "" {
		req.Desc.SetTo(v)
	}
	if v, ok := params["is_all_day"].(bool); ok {
		req.IsAllDay.SetTo(v)
	}
	if v, ok := params["start_date"].(string); ok && v != "" {
		req.StartDate.SetTo(v)
	}
	if v, ok := params["due_date"].(string); ok && v != "" {
		req.DueDate.SetTo(v)
	}
	if v, ok := params["time_zone"].(string); ok && v != "" {
		req.TimeZone.SetTo(v)
	}
	if v, ok := params["reminders"].([]interface{}); ok && len(v) > 0 {
		req.Reminders = toStringSlice(v)
	}
	if v, ok := params["repeat_flag"].(string); ok && v != "" {
		req.RepeatFlag.SetTo(v)
	}
	if v, ok := params["priority"].(float64); ok {
		req.Priority.SetTo(int(v))
	}
	if v, ok := params["sort_order"].(float64); ok {
		req.SortOrder.SetTo(int64(v))
	}
	if v, ok := params["items"].([]interface{}); ok && len(v) > 0 {
		req.Items = toChecklistItems(v)
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
	projectID, _ := params["project_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	req := gen.UpdateTaskReq{
		ID:        taskID,
		ProjectId: projectID,
	}
	if v, ok := params["title"].(string); ok && v != "" {
		req.Title.SetTo(v)
	}
	if v, ok := params["content"].(string); ok && v != "" {
		req.Content.SetTo(v)
	}
	if v, ok := params["desc"].(string); ok && v != "" {
		req.Desc.SetTo(v)
	}
	if v, ok := params["is_all_day"].(bool); ok {
		req.IsAllDay.SetTo(v)
	}
	if v, ok := params["start_date"].(string); ok && v != "" {
		req.StartDate.SetTo(v)
	}
	if v, ok := params["due_date"].(string); ok && v != "" {
		req.DueDate.SetTo(v)
	}
	if v, ok := params["time_zone"].(string); ok && v != "" {
		req.TimeZone.SetTo(v)
	}
	if v, ok := params["reminders"].([]interface{}); ok && len(v) > 0 {
		req.Reminders = toStringSlice(v)
	}
	if v, ok := params["repeat_flag"].(string); ok && v != "" {
		req.RepeatFlag.SetTo(v)
	}
	if v, ok := params["priority"].(float64); ok {
		req.Priority.SetTo(int(v))
	}
	if v, ok := params["sort_order"].(float64); ok {
		req.SortOrder.SetTo(int64(v))
	}
	if v, ok := params["items"].([]interface{}); ok && len(v) > 0 {
		req.Items = toChecklistItems(v)
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
	projectID, _ := params["project_id"].(string)
	taskID, _ := params["task_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	err = c.CompleteTask(ctx, gen.CompleteTaskParams{ProjectId: projectID, TaskId: taskID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Task completed"}`, nil
}

func deleteTask(ctx context.Context, params map[string]any) (string, error) {
	projectID, _ := params["project_id"].(string)
	taskID, _ := params["task_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	err = c.DeleteTask(ctx, gen.DeleteTaskParams{ProjectId: projectID, TaskId: taskID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Task deleted"}`, nil
}

// =============================================================================
// Helpers
// =============================================================================

// toChecklistItems converts []interface{} from MCP params to []gen.ChecklistItem.
func toChecklistItems(v []interface{}) []gen.ChecklistItem {
	items := make([]gen.ChecklistItem, 0, len(v))
	for _, raw := range v {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		item := gen.ChecklistItem{}
		if t, ok := m["title"].(string); ok {
			item.Title.SetTo(t)
		}
		if s, ok := m["status"].(float64); ok {
			item.Status.SetTo(int(s))
		}
		items = append(items, item)
	}
	return items
}
