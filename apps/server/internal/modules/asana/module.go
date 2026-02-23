package asana

import (
	"context"
	"fmt"
	"log"

	"mcpist/server/internal/broker"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/pkg/asanaapi"
	gen "mcpist/server/pkg/asanaapi/gen"
)

const (
	asanaVersion = "1.0"
)

// AsanaModule implements the Module interface for Asana API
type AsanaModule struct{}

// New creates a new AsanaModule instance
func New() *AsanaModule {
	return &AsanaModule{}
}

// Module descriptions
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Asana API - Workspaces, projects, tasks, sections, and tags management",
	"ja-JP": "Asana API - ワークスペース、プロジェクト、タスク、セクション、タグの管理",
}

// Name returns the module name
func (m *AsanaModule) Name() string {
	return "asana"
}

// Descriptions returns the module descriptions in all languages
func (m *AsanaModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description (English)
func (m *AsanaModule) Description() string {
	return moduleDescriptions["en-US"]
}

// APIVersion returns the Asana API version
func (m *AsanaModule) APIVersion() string {
	return asanaVersion
}

// Tools returns all available tools
func (m *AsanaModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *AsanaModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format (MD or CSV)
// Implements modules.CompactConverter interface
func (m *AsanaModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// Resources returns all available resources (none for Asana)
func (m *AsanaModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *AsanaModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[asana] No auth context")
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "asana")
	if err != nil {
		log.Printf("[asana] GetModuleToken error: %v", err)
		return nil
	}
	return credentials
}

// =============================================================================
// ogen client helpers
// =============================================================================

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return asanaapi.NewClient(creds.AccessToken)
}

var toJSON = modules.ToJSON

var toStringSlice = modules.ToStringSlice

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	// User
	{
		ID:   "asana:get_me",
		Name: "get_me",
		Descriptions: modules.LocalizedText{
			"en-US": "Get the current authenticated user's information.",
			"ja-JP": "現在認証されているユーザーの情報を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	// Workspaces
	{
		ID:   "asana:list_workspaces",
		Name: "list_workspaces",
		Descriptions: modules.LocalizedText{
			"en-US": "List all workspaces the user has access to.",
			"ja-JP": "ユーザーがアクセスできるすべてのワークスペースを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "asana:get_workspace",
		Name: "get_workspace",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific workspace.",
			"ja-JP": "特定のワークスペースの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"workspace_gid": {Type: "string", Description: "Workspace GID"},
			},
			Required: []string{"workspace_gid"},
		},
	},
	// Projects
	{
		ID:   "asana:list_projects",
		Name: "list_projects",
		Descriptions: modules.LocalizedText{
			"en-US": "List projects in a workspace or team.",
			"ja-JP": "ワークスペースまたはチーム内のプロジェクトを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"workspace_gid": {Type: "string", Description: "Workspace GID"},
				"team_gid":      {Type: "string", Description: "Team GID (optional)"},
				"archived":      {Type: "boolean", Description: "Include archived projects"},
			},
			Required: []string{"workspace_gid"},
		},
	},
	{
		ID:   "asana:get_project",
		Name: "get_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific project.",
			"ja-JP": "特定のプロジェクトの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_gid": {Type: "string", Description: "Project GID"},
			},
			Required: []string{"project_gid"},
		},
	},
	{
		ID:   "asana:create_project",
		Name: "create_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new project in a workspace or team.",
			"ja-JP": "ワークスペースまたはチームに新しいプロジェクトを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"name":          {Type: "string", Description: "Project name (required)"},
				"workspace_gid": {Type: "string", Description: "Workspace GID (required if team_gid not provided)"},
				"team_gid":      {Type: "string", Description: "Team GID (required if workspace_gid not provided)"},
				"notes":         {Type: "string", Description: "Project description"},
				"color":         {Type: "string", Description: "Project color (dark-pink, dark-green, dark-blue, dark-red, dark-teal, dark-brown, dark-orange, dark-purple, dark-warm-gray, light-pink, light-green, light-blue, light-red, light-teal, light-brown, light-orange, light-purple, light-warm-gray, none)"},
				"default_view":  {Type: "string", Description: "Default view: list, board, calendar, timeline"},
				"due_on":        {Type: "string", Description: "Due date (YYYY-MM-DD format)"},
			},
			Required: []string{"name"},
		},
	},
	{
		ID:   "asana:update_project",
		Name: "update_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing project.",
			"ja-JP": "既存のプロジェクトを更新します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_gid":  {Type: "string", Description: "Project GID (required)"},
				"name":         {Type: "string", Description: "New project name"},
				"notes":        {Type: "string", Description: "New project description"},
				"color":        {Type: "string", Description: "Project color"},
				"default_view": {Type: "string", Description: "Default view"},
				"due_on":       {Type: "string", Description: "Due date (YYYY-MM-DD format)"},
				"archived":     {Type: "boolean", Description: "Archive status"},
			},
			Required: []string{"project_gid"},
		},
	},
	{
		ID:   "asana:delete_project",
		Name: "delete_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a project.",
			"ja-JP": "プロジェクトを削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_gid": {Type: "string", Description: "Project GID"},
			},
			Required: []string{"project_gid"},
		},
	},
	// Sections
	{
		ID:   "asana:list_sections",
		Name: "list_sections",
		Descriptions: modules.LocalizedText{
			"en-US": "List all sections in a project.",
			"ja-JP": "プロジェクト内のすべてのセクションを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_gid": {Type: "string", Description: "Project GID"},
			},
			Required: []string{"project_gid"},
		},
	},
	{
		ID:   "asana:create_section",
		Name: "create_section",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new section in a project.",
			"ja-JP": "プロジェクトに新しいセクションを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_gid": {Type: "string", Description: "Project GID (required)"},
				"name":        {Type: "string", Description: "Section name (required)"},
			},
			Required: []string{"project_gid", "name"},
		},
	},
	// Tasks
	{
		ID:   "asana:list_tasks",
		Name: "list_tasks",
		Descriptions: modules.LocalizedText{
			"en-US": "List tasks in a project or section.",
			"ja-JP": "プロジェクトまたはセクション内のタスクを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"project_gid":   {Type: "string", Description: "Project GID"},
				"section_gid":   {Type: "string", Description: "Section GID"},
				"assignee_gid":  {Type: "string", Description: "Assignee user GID"},
				"workspace_gid": {Type: "string", Description: "Workspace GID (required when using assignee)"},
				"completed":     {Type: "boolean", Description: "Filter by completion status"},
			},
		},
	},
	{
		ID:   "asana:get_task",
		Name: "get_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific task.",
			"ja-JP": "特定のタスクの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_gid": {Type: "string", Description: "Task GID"},
			},
			Required: []string{"task_gid"},
		},
	},
	{
		ID:   "asana:create_task",
		Name: "create_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new task.",
			"ja-JP": "新しいタスクを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"name":          {Type: "string", Description: "Task name (required)"},
				"workspace_gid": {Type: "string", Description: "Workspace GID (required if projects not provided)"},
				"projects":      {Type: "array", Description: "Array of project GIDs"},
				"section_gid":   {Type: "string", Description: "Section GID to add task to"},
				"parent_gid":    {Type: "string", Description: "Parent task GID for subtasks"},
				"notes":         {Type: "string", Description: "Task description (plain text)"},
				"html_notes":    {Type: "string", Description: "Task description (HTML)"},
				"due_on":        {Type: "string", Description: "Due date (YYYY-MM-DD format)"},
				"due_at":        {Type: "string", Description: "Due datetime (ISO 8601 format)"},
				"start_on":      {Type: "string", Description: "Start date (YYYY-MM-DD format)"},
				"assignee_gid":  {Type: "string", Description: "Assignee user GID"},
				"tags":          {Type: "array", Description: "Array of tag GIDs"},
			},
			Required: []string{"name"},
		},
	},
	{
		ID:   "asana:update_task",
		Name: "update_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing task.",
			"ja-JP": "既存のタスクを更新します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_gid":     {Type: "string", Description: "Task GID (required)"},
				"name":         {Type: "string", Description: "New task name"},
				"notes":        {Type: "string", Description: "Task description (plain text)"},
				"html_notes":   {Type: "string", Description: "Task description (HTML)"},
				"due_on":       {Type: "string", Description: "Due date (YYYY-MM-DD format)"},
				"due_at":       {Type: "string", Description: "Due datetime (ISO 8601 format)"},
				"start_on":     {Type: "string", Description: "Start date (YYYY-MM-DD format)"},
				"completed":    {Type: "boolean", Description: "Completion status"},
				"assignee_gid": {Type: "string", Description: "Assignee user GID"},
			},
			Required: []string{"task_gid"},
		},
	},
	{
		ID:   "asana:complete_task",
		Name: "complete_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Mark a task as completed.",
			"ja-JP": "タスクを完了としてマークします。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_gid": {Type: "string", Description: "Task GID"},
			},
			Required: []string{"task_gid"},
		},
	},
	{
		ID:   "asana:delete_task",
		Name: "delete_task",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a task.",
			"ja-JP": "タスクを削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_gid": {Type: "string", Description: "Task GID"},
			},
			Required: []string{"task_gid"},
		},
	},
	// Subtasks
	{
		ID:   "asana:list_subtasks",
		Name: "list_subtasks",
		Descriptions: modules.LocalizedText{
			"en-US": "List subtasks of a task.",
			"ja-JP": "タスクのサブタスクを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_gid": {Type: "string", Description: "Parent task GID"},
			},
			Required: []string{"task_gid"},
		},
	},
	{
		ID:   "asana:create_subtask",
		Name: "create_subtask",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a subtask under a parent task.",
			"ja-JP": "親タスクの下にサブタスクを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"parent_gid":   {Type: "string", Description: "Parent task GID (required)"},
				"name":         {Type: "string", Description: "Subtask name (required)"},
				"notes":        {Type: "string", Description: "Subtask description"},
				"due_on":       {Type: "string", Description: "Due date (YYYY-MM-DD format)"},
				"assignee_gid": {Type: "string", Description: "Assignee user GID"},
			},
			Required: []string{"parent_gid", "name"},
		},
	},
	// Stories (Comments)
	{
		ID:   "asana:list_stories",
		Name: "list_stories",
		Descriptions: modules.LocalizedText{
			"en-US": "List stories (comments and activity) on a task.",
			"ja-JP": "タスクのストーリー（コメントとアクティビティ）を一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_gid": {Type: "string", Description: "Task GID"},
			},
			Required: []string{"task_gid"},
		},
	},
	{
		ID:   "asana:add_comment",
		Name: "add_comment",
		Descriptions: modules.LocalizedText{
			"en-US": "Add a comment to a task.",
			"ja-JP": "タスクにコメントを追加します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"task_gid": {Type: "string", Description: "Task GID (required)"},
				"text":     {Type: "string", Description: "Comment text (required)"},
			},
			Required: []string{"task_gid", "text"},
		},
	},
	// Tags
	{
		ID:   "asana:list_tags",
		Name: "list_tags",
		Descriptions: modules.LocalizedText{
			"en-US": "List tags in a workspace.",
			"ja-JP": "ワークスペース内のタグを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"workspace_gid": {Type: "string", Description: "Workspace GID"},
			},
			Required: []string{"workspace_gid"},
		},
	},
	{
		ID:   "asana:create_tag",
		Name: "create_tag",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new tag in a workspace.",
			"ja-JP": "ワークスペースに新しいタグを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"workspace_gid": {Type: "string", Description: "Workspace GID (required)"},
				"name":          {Type: "string", Description: "Tag name (required)"},
				"color":         {Type: "string", Description: "Tag color"},
			},
			Required: []string{"workspace_gid", "name"},
		},
	},
	// Search
	{
		ID:   "asana:search_tasks",
		Name: "search_tasks",
		Descriptions: modules.LocalizedText{
			"en-US": "Search for tasks in a workspace using advanced filters.",
			"ja-JP": "高度なフィルターを使用してワークスペース内のタスクを検索します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"workspace_gid":         {Type: "string", Description: "Workspace GID (required)"},
				"text":                  {Type: "string", Description: "Search text"},
				"completed":             {Type: "boolean", Description: "Filter by completion status"},
				"is_subtask":            {Type: "boolean", Description: "Filter subtasks only"},
				"assignee_gid":          {Type: "string", Description: "Filter by assignee"},
				"projects_gid":          {Type: "string", Description: "Filter by project"},
				"due_on_before":         {Type: "string", Description: "Due on or before date (YYYY-MM-DD)"},
				"due_on_after":          {Type: "string", Description: "Due on or after date (YYYY-MM-DD)"},
				"sort_by":               {Type: "string", Description: "Sort by: due_date, created_at, completed_at, likes, modified_at"},
				"sort_ascending":        {Type: "boolean", Description: "Sort ascending (default: false)"},
			},
			Required: []string{"workspace_gid"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	// User
	"get_me": getMe,
	// Workspaces
	"list_workspaces": listWorkspaces,
	"get_workspace":   getWorkspace,
	// Projects
	"list_projects":   listProjects,
	"get_project":     getProject,
	"create_project":  createProject,
	"update_project":  updateProject,
	"delete_project":  deleteProject,
	// Sections
	"list_sections":   listSections,
	"create_section":  createSection,
	// Tasks
	"list_tasks":    listTasks,
	"get_task":      getTask,
	"create_task":   createTask,
	"update_task":   updateTask,
	"complete_task": completeTask,
	"delete_task":   deleteTask,
	// Subtasks
	"list_subtasks":   listSubtasks,
	"create_subtask":  createSubtask,
	// Stories
	"list_stories": listStories,
	"add_comment":  addComment,
	// Tags
	"list_tags":   listTags,
	"create_tag":  createTag,
	// Search
	"search_tasks": searchTasks,
}

// =============================================================================
// User
// =============================================================================

func getMe(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetMe(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

// =============================================================================
// Workspaces
// =============================================================================

func listWorkspaces(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListWorkspaces(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

func getWorkspace(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	workspaceGID, _ := params["workspace_gid"].(string)
	res, err := c.GetWorkspace(ctx, gen.GetWorkspaceParams{WorkspaceGid: workspaceGID})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

// =============================================================================
// Projects
// =============================================================================

func listProjects(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	workspaceGID, _ := params["workspace_gid"].(string)
	teamGID, _ := params["team_gid"].(string)

	var archived gen.OptBool
	if a, ok := params["archived"].(bool); ok {
		archived = gen.NewOptBool(a)
	}

	if teamGID != "" {
		res, err := c.ListProjectsByTeam(ctx, gen.ListProjectsByTeamParams{
			TeamGid:  teamGID,
			Archived: archived,
		})
		if err != nil {
			return "", err
		}
		return toJSON(res.Data)
	}

	res, err := c.ListProjectsByWorkspace(ctx, gen.ListProjectsByWorkspaceParams{
		WorkspaceGid: workspaceGID,
		Archived:     archived,
	})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

func getProject(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectGID, _ := params["project_gid"].(string)
	res, err := c.GetProject(ctx, gen.GetProjectParams{ProjectGid: projectGID})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

func createProject(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	reqData := gen.CreateProjectRequestData{}
	if name, ok := params["name"].(string); ok {
		reqData.Name.SetTo(name)
	}
	if workspaceGID, ok := params["workspace_gid"].(string); ok && workspaceGID != "" {
		reqData.Workspace.SetTo(workspaceGID)
	}
	if teamGID, ok := params["team_gid"].(string); ok && teamGID != "" {
		reqData.Team.SetTo(teamGID)
	}
	if notes, ok := params["notes"].(string); ok {
		reqData.Notes.SetTo(notes)
	}
	if color, ok := params["color"].(string); ok {
		reqData.Color.SetTo(color)
	}
	if defaultView, ok := params["default_view"].(string); ok {
		reqData.DefaultView.SetTo(defaultView)
	}
	if dueOn, ok := params["due_on"].(string); ok {
		reqData.DueOn.SetTo(dueOn)
	}
	res, err := c.CreateProject(ctx, &gen.CreateProjectRequest{Data: reqData})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

func updateProject(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectGID, _ := params["project_gid"].(string)
	reqData := gen.UpdateProjectRequestData{}
	if name, ok := params["name"].(string); ok && name != "" {
		reqData.Name.SetTo(name)
	}
	if notes, ok := params["notes"].(string); ok {
		reqData.Notes.SetTo(notes)
	}
	if color, ok := params["color"].(string); ok {
		reqData.Color.SetTo(color)
	}
	if defaultView, ok := params["default_view"].(string); ok {
		reqData.DefaultView.SetTo(defaultView)
	}
	if dueOn, ok := params["due_on"].(string); ok {
		reqData.DueOn.SetTo(dueOn)
	}
	if archived, ok := params["archived"].(bool); ok {
		reqData.Archived.SetTo(archived)
	}
	res, err := c.UpdateProject(ctx, &gen.UpdateProjectRequest{Data: reqData}, gen.UpdateProjectParams{ProjectGid: projectGID})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

func deleteProject(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectGID, _ := params["project_gid"].(string)
	_, err = c.DeleteProject(ctx, gen.DeleteProjectParams{ProjectGid: projectGID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Project deleted"}`, nil
}

// =============================================================================
// Sections
// =============================================================================

func listSections(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectGID, _ := params["project_gid"].(string)
	res, err := c.ListSections(ctx, gen.ListSectionsParams{ProjectGid: projectGID})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

func createSection(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	projectGID, _ := params["project_gid"].(string)
	name, _ := params["name"].(string)
	reqData := gen.CreateSectionRequestData{}
	reqData.Name.SetTo(name)
	res, err := c.CreateSection(ctx, &gen.CreateSectionRequest{Data: reqData}, gen.CreateSectionParams{ProjectGid: projectGID})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

// =============================================================================
// Tasks
// =============================================================================

func listTasks(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	projectGID, hasProject := params["project_gid"].(string)
	sectionGID, hasSection := params["section_gid"].(string)
	assigneeGID, hasAssignee := params["assignee_gid"].(string)
	workspaceGID, _ := params["workspace_gid"].(string)

	var completedSince gen.OptString
	if completed, ok := params["completed"].(bool); ok && completed {
		completedSince = gen.NewOptString("2000-01-01T00:00:00Z")
	}

	optFields := gen.NewOptString("name,completed,due_on,due_at,assignee.name,notes")

	if hasSection && sectionGID != "" {
		res, err := c.ListTasksBySection(ctx, gen.ListTasksBySectionParams{
			SectionGid:     sectionGID,
			CompletedSince: completedSince,
			OptFields:      optFields,
		})
		if err != nil {
			return "", err
		}
		return toJSON(res.Data)
	}

	if hasProject && projectGID != "" {
		res, err := c.ListTasksByProject(ctx, gen.ListTasksByProjectParams{
			ProjectGid:     projectGID,
			CompletedSince: completedSince,
			OptFields:      optFields,
		})
		if err != nil {
			return "", err
		}
		return toJSON(res.Data)
	}

	if hasAssignee && assigneeGID != "" && workspaceGID != "" {
		res, err := c.ListTasksByAssignee(ctx, gen.ListTasksByAssigneeParams{
			Assignee:       assigneeGID,
			Workspace:      workspaceGID,
			CompletedSince: completedSince,
			OptFields:      optFields,
		})
		if err != nil {
			return "", err
		}
		return toJSON(res.Data)
	}

	return "", fmt.Errorf("must provide project_gid, section_gid, or (assignee_gid + workspace_gid)")
}

func getTask(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	taskGID, _ := params["task_gid"].(string)
	res, err := c.GetTask(ctx, gen.GetTaskParams{TaskGid: taskGID})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

func createTask(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	reqData := gen.CreateTaskRequestData{}
	if name, ok := params["name"].(string); ok {
		reqData.Name.SetTo(name)
	}
	if workspaceGID, ok := params["workspace_gid"].(string); ok && workspaceGID != "" {
		reqData.Workspace.SetTo(workspaceGID)
	}
	if projects, ok := params["projects"].([]interface{}); ok && len(projects) > 0 {
		reqData.Projects = toStringSlice(projects)
	}
	if notes, ok := params["notes"].(string); ok {
		reqData.Notes.SetTo(notes)
	}
	if htmlNotes, ok := params["html_notes"].(string); ok {
		reqData.HTMLNotes.SetTo(htmlNotes)
	}
	if dueOn, ok := params["due_on"].(string); ok {
		reqData.DueOn.SetTo(dueOn)
	}
	if dueAt, ok := params["due_at"].(string); ok {
		reqData.DueAt.SetTo(dueAt)
	}
	if startOn, ok := params["start_on"].(string); ok {
		reqData.StartOn.SetTo(startOn)
	}
	if assigneeGID, ok := params["assignee_gid"].(string); ok {
		reqData.Assignee.SetTo(assigneeGID)
	}
	if tags, ok := params["tags"].([]interface{}); ok && len(tags) > 0 {
		reqData.Tags = toStringSlice(tags)
	}
	if parentGID, ok := params["parent_gid"].(string); ok && parentGID != "" {
		reqData.Parent.SetTo(parentGID)
	}

	res, err := c.CreateTask(ctx, &gen.CreateTaskRequest{Data: reqData})
	if err != nil {
		return "", err
	}

	// Add to section if specified
	sectionGID, hasSection := params["section_gid"].(string)
	if hasSection && sectionGID != "" {
		if taskData, ok := res.Data.Get(); ok {
			if taskGID, ok := taskData.Gid.Get(); ok && taskGID != "" {
				addReqData := gen.AddTaskToSectionRequestData{}
				addReqData.Task.SetTo(taskGID)
				_, _ = c.AddTaskToSection(ctx, &gen.AddTaskToSectionRequest{Data: addReqData}, gen.AddTaskToSectionParams{SectionGid: sectionGID})
			}
		}
	}

	return toJSON(res.Data)
}

func updateTask(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	taskGID, _ := params["task_gid"].(string)
	reqData := gen.UpdateTaskRequestData{}
	if name, ok := params["name"].(string); ok && name != "" {
		reqData.Name.SetTo(name)
	}
	if notes, ok := params["notes"].(string); ok {
		reqData.Notes.SetTo(notes)
	}
	if htmlNotes, ok := params["html_notes"].(string); ok {
		reqData.HTMLNotes.SetTo(htmlNotes)
	}
	if dueOn, ok := params["due_on"].(string); ok {
		reqData.DueOn.SetTo(dueOn)
	}
	if dueAt, ok := params["due_at"].(string); ok {
		reqData.DueAt.SetTo(dueAt)
	}
	if startOn, ok := params["start_on"].(string); ok {
		reqData.StartOn.SetTo(startOn)
	}
	if completed, ok := params["completed"].(bool); ok {
		reqData.Completed.SetTo(completed)
	}
	if assigneeGID, ok := params["assignee_gid"].(string); ok {
		reqData.Assignee.SetTo(assigneeGID)
	}
	res, err := c.UpdateTask(ctx, &gen.UpdateTaskRequest{Data: reqData}, gen.UpdateTaskParams{TaskGid: taskGID})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

func completeTask(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	taskGID, _ := params["task_gid"].(string)
	reqData := gen.UpdateTaskRequestData{}
	reqData.Completed.SetTo(true)
	res, err := c.UpdateTask(ctx, &gen.UpdateTaskRequest{Data: reqData}, gen.UpdateTaskParams{TaskGid: taskGID})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

func deleteTask(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	taskGID, _ := params["task_gid"].(string)
	_, err = c.DeleteTask(ctx, gen.DeleteTaskParams{TaskGid: taskGID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Task deleted"}`, nil
}

// =============================================================================
// Subtasks
// =============================================================================

func listSubtasks(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	taskGID, _ := params["task_gid"].(string)
	res, err := c.ListSubtasks(ctx, gen.ListSubtasksParams{
		TaskGid:   taskGID,
		OptFields: gen.NewOptString("name,completed,due_on,assignee.name"),
	})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

func createSubtask(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	parentGID, _ := params["parent_gid"].(string)
	reqData := gen.CreateSubtaskRequestData{}
	if name, ok := params["name"].(string); ok {
		reqData.Name.SetTo(name)
	}
	if notes, ok := params["notes"].(string); ok {
		reqData.Notes.SetTo(notes)
	}
	if dueOn, ok := params["due_on"].(string); ok {
		reqData.DueOn.SetTo(dueOn)
	}
	if assigneeGID, ok := params["assignee_gid"].(string); ok {
		reqData.Assignee.SetTo(assigneeGID)
	}
	res, err := c.CreateSubtask(ctx, &gen.CreateSubtaskRequest{Data: reqData}, gen.CreateSubtaskParams{TaskGid: parentGID})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

// =============================================================================
// Stories (Comments)
// =============================================================================

func listStories(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	taskGID, _ := params["task_gid"].(string)
	res, err := c.ListStories(ctx, gen.ListStoriesParams{TaskGid: taskGID})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

func addComment(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	taskGID, _ := params["task_gid"].(string)
	text, _ := params["text"].(string)
	reqData := gen.CreateStoryRequestData{}
	reqData.Text.SetTo(text)
	res, err := c.CreateStory(ctx, &gen.CreateStoryRequest{Data: reqData}, gen.CreateStoryParams{TaskGid: taskGID})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

// =============================================================================
// Tags
// =============================================================================

func listTags(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	workspaceGID, _ := params["workspace_gid"].(string)
	res, err := c.ListTags(ctx, gen.ListTagsParams{WorkspaceGid: workspaceGID})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

func createTag(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	workspaceGID, _ := params["workspace_gid"].(string)
	name, _ := params["name"].(string)
	reqData := gen.CreateTagRequestData{}
	reqData.Name.SetTo(name)
	reqData.Workspace.SetTo(workspaceGID)
	if color, ok := params["color"].(string); ok {
		reqData.Color.SetTo(color)
	}
	res, err := c.CreateTag(ctx, &gen.CreateTagRequest{Data: reqData})
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}

// =============================================================================
// Search
// =============================================================================

func searchTasks(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	workspaceGID, _ := params["workspace_gid"].(string)
	p := gen.SearchTasksParams{
		WorkspaceGid: workspaceGID,
		OptFields:    gen.NewOptString("name,completed,due_on,due_at,assignee.name,notes,projects.name"),
	}
	if text, ok := params["text"].(string); ok && text != "" {
		p.Text = gen.NewOptString(text)
	}
	if completed, ok := params["completed"].(bool); ok {
		p.Completed = gen.NewOptBool(completed)
	}
	if isSubtask, ok := params["is_subtask"].(bool); ok {
		p.IsSubtask = gen.NewOptBool(isSubtask)
	}
	if assigneeGID, ok := params["assignee_gid"].(string); ok && assigneeGID != "" {
		p.AssigneeAny = gen.NewOptString(assigneeGID)
	}
	if projectsGID, ok := params["projects_gid"].(string); ok && projectsGID != "" {
		p.ProjectsAny = gen.NewOptString(projectsGID)
	}
	if dueOnBefore, ok := params["due_on_before"].(string); ok && dueOnBefore != "" {
		p.DueOnBefore = gen.NewOptString(dueOnBefore)
	}
	if dueOnAfter, ok := params["due_on_after"].(string); ok && dueOnAfter != "" {
		p.DueOnAfter = gen.NewOptString(dueOnAfter)
	}
	if sortBy, ok := params["sort_by"].(string); ok && sortBy != "" {
		p.SortBy = gen.NewOptString(sortBy)
	}
	if sortAscending, ok := params["sort_ascending"].(bool); ok && sortAscending {
		p.SortAscending = gen.NewOptBool(true)
	}
	res, err := c.SearchTasks(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res.Data)
}
