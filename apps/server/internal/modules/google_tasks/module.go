package google_tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"mcpist/server/internal/broker"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/pkg/googletasksapi"
	gen "mcpist/server/pkg/googletasksapi/gen"
)

const (
	googleTasksVersion = "v1"
)

var toJSON = modules.ToJSON

// GoogleTasksModule implements the Module interface for Google Tasks API
type GoogleTasksModule struct{}

func New() *GoogleTasksModule { return &GoogleTasksModule{} }

var moduleDescriptions = modules.LocalizedText{
	"en-US": "Google Tasks API - List, create, update, and delete tasks",
	"ja-JP": "Google Tasks API - タスクの一覧表示、作成、更新、削除",
}

func (m *GoogleTasksModule) Name() string                        { return "google_tasks" }
func (m *GoogleTasksModule) Descriptions() modules.LocalizedText { return moduleDescriptions }
func (m *GoogleTasksModule) Description() string {
	return moduleDescriptions["en-US"]
}
func (m *GoogleTasksModule) APIVersion() string            { return googleTasksVersion }
func (m *GoogleTasksModule) Tools() []modules.Tool         { return toolDefinitions }
func (m *GoogleTasksModule) Resources() []modules.Resource { return nil }
func (m *GoogleTasksModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

func (m *GoogleTasksModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format.
func (m *GoogleTasksModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// =============================================================================
// Token and Client
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[google_tasks] No auth context")
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "google_tasks")
	if err != nil {
		log.Printf("[google_tasks] GetModuleToken error: %v", err)
		return nil
	}
	return credentials
}

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return googletasksapi.NewClient(creds.AccessToken)
}

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
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
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListTaskLists(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getTaskList(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetTaskList(ctx, gen.GetTaskListParams{TaskListId: taskListID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Tasks
// =============================================================================

func listTasks(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)

	p := gen.ListTasksParams{
		TaskListId:    taskListID,
		MaxResults:    gen.NewOptInt(100),
		ShowCompleted: gen.NewOptBool(true),
		ShowHidden:    gen.NewOptBool(false),
	}

	if mr, ok := params["max_results"].(float64); ok {
		p.MaxResults = gen.NewOptInt(int(mr))
	}
	if sc, ok := params["show_completed"].(bool); ok {
		p.ShowCompleted = gen.NewOptBool(sc)
	}
	if sh, ok := params["show_hidden"].(bool); ok {
		p.ShowHidden = gen.NewOptBool(sh)
	}
	if dm, ok := params["due_min"].(string); ok && dm != "" {
		p.DueMin = gen.NewOptString(dm)
	}
	if dx, ok := params["due_max"].(string); ok && dx != "" {
		p.DueMax = gen.NewOptString(dx)
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListTasks(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getTask(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)
	taskID, _ := params["task_id"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetTask(ctx, gen.GetTaskParams{TaskListId: taskListID, TaskId: taskID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func createTask(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)
	title, _ := params["title"].(string)

	req := &gen.TaskRequest{
		Title: gen.NewOptNilString(title),
	}
	if notes, ok := params["notes"].(string); ok && notes != "" {
		req.Notes = gen.NewOptNilString(notes)
	}
	if due, ok := params["due"].(string); ok && due != "" {
		req.Due = gen.NewOptNilString(due)
	}

	p := gen.CreateTaskParams{TaskListId: taskListID}
	if parent, ok := params["parent"].(string); ok && parent != "" {
		p.Parent = gen.NewOptString(parent)
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.CreateTask(ctx, req, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func updateTask(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)
	taskID, _ := params["task_id"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	// Get existing task to preserve fields
	existing, err := c.GetTask(ctx, gen.GetTaskParams{TaskListId: taskListID, TaskId: taskID})
	if err != nil {
		return "", err
	}

	req := &gen.TaskRequest{
		Title:  existing.Title,
		Notes:  existing.Notes,
		Status: existing.Status,
		Due:    existing.Due,
	}

	// Override with provided params
	if title, ok := params["title"].(string); ok && title != "" {
		req.Title = gen.NewOptNilString(title)
	}
	if notes, ok := params["notes"].(string); ok {
		req.Notes = gen.NewOptNilString(notes)
	}
	if due, ok := params["due"].(string); ok {
		if due == "" {
			req.Due = gen.OptNilString{}
		} else {
			req.Due = gen.NewOptNilString(due)
		}
	}
	if status, ok := params["status"].(string); ok && status != "" {
		req.Status = gen.NewOptNilString(status)
		if status == "completed" {
			req.Completed = gen.NewOptNilString(time.Now().UTC().Format(time.RFC3339))
		} else {
			req.Completed = gen.OptNilString{}
		}
	}

	res, err := c.UpdateTask(ctx, req, gen.UpdateTaskParams{TaskListId: taskListID, TaskId: taskID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func deleteTask(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)
	taskID, _ := params["task_id"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	err = c.DeleteTask(ctx, gen.DeleteTaskParams{TaskListId: taskListID, TaskId: taskID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Task deleted"}`, nil
}

func completeTask(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)
	taskID, _ := params["task_id"].(string)
	completed, _ := params["completed"].(bool)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	// Get existing task
	existing, err := c.GetTask(ctx, gen.GetTaskParams{TaskListId: taskListID, TaskId: taskID})
	if err != nil {
		return "", err
	}

	req := &gen.TaskRequest{
		Title: existing.Title,
		Notes: existing.Notes,
		Due:   existing.Due,
	}

	if completed {
		req.Status = gen.NewOptNilString("completed")
		req.Completed = gen.NewOptNilString(time.Now().UTC().Format(time.RFC3339))
	} else {
		req.Status = gen.NewOptNilString("needsAction")
		req.Completed = gen.OptNilString{}
	}

	res, err := c.UpdateTask(ctx, req, gen.UpdateTaskParams{TaskListId: taskListID, TaskId: taskID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func clearCompleted(ctx context.Context, params map[string]any) (string, error) {
	taskListID, _ := params["task_list_id"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	// List all tasks including completed
	res, err := c.ListTasks(ctx, gen.ListTasksParams{
		TaskListId:    taskListID,
		ShowCompleted: gen.NewOptBool(true),
		ShowHidden:    gen.NewOptBool(false),
	})
	if err != nil {
		return "", err
	}

	// Delete each completed task
	deletedCount := 0
	for _, task := range res.Items {
		if s, ok := task.Status.Get(); ok && s == "completed" {
			id, ok := task.ID.Get()
			if !ok {
				continue
			}
			err := c.DeleteTask(ctx, gen.DeleteTaskParams{TaskListId: taskListID, TaskId: id})
			if err != nil {
				log.Printf("[google_tasks] Failed to delete task %s: %v", id, err)
				continue
			}
			deletedCount++
		}
	}

	result := map[string]any{
		"success":       true,
		"deleted_count": deletedCount,
		"message":       fmt.Sprintf("Deleted %d completed tasks", deletedCount),
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}
