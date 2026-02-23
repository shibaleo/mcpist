package google_apps_script

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-faster/jx"

	"mcpist/server/internal/broker"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/pkg/googleappsscriptapi"
	gen "mcpist/server/pkg/googleappsscriptapi/gen"
	"mcpist/server/pkg/googledriveapi"
	driveGen "mcpist/server/pkg/googledriveapi/gen"
)

const (
	appsScriptVersion = "v1"
)

var toJSON = modules.ToJSON

// GoogleAppsScriptModule implements the Module interface for Google Apps Script API
type GoogleAppsScriptModule struct{}

func New() *GoogleAppsScriptModule { return &GoogleAppsScriptModule{} }

var moduleDescriptions = modules.LocalizedText{
	"en-US": "Google Apps Script API - Manage script projects, deployments, versions, and executions",
	"ja-JP": "Google Apps Script API - スクリプトプロジェクト、デプロイ、バージョン、実行の管理",
}

func (m *GoogleAppsScriptModule) Name() string                        { return "google_apps_script" }
func (m *GoogleAppsScriptModule) Descriptions() modules.LocalizedText { return moduleDescriptions }
func (m *GoogleAppsScriptModule) Description() string {
	return moduleDescriptions["en-US"]
}
func (m *GoogleAppsScriptModule) APIVersion() string           { return appsScriptVersion }
func (m *GoogleAppsScriptModule) Tools() []modules.Tool        { return toolDefinitions }
func (m *GoogleAppsScriptModule) Resources() []modules.Resource { return nil }
func (m *GoogleAppsScriptModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

func (m *GoogleAppsScriptModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format.
func (m *GoogleAppsScriptModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// =============================================================================
// Token and Client
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[google_apps_script] No auth context")
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "google_apps_script")
	if err != nil {
		log.Printf("[google_apps_script] GetModuleToken error: %v", err)
		return nil
	}
	return credentials
}

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return googleappsscriptapi.NewClient(creds.AccessToken)
}

func newDriveClient(ctx context.Context) (*driveGen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return googledriveapi.NewClient(creds.AccessToken)
}

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	// =========================================================================
	// Project Operations
	// =========================================================================
	{
		ID:   "google_apps_script:list_projects",
		Name: "list_projects",
		Descriptions: modules.LocalizedText{
			"en-US": "List Google Apps Script projects in Google Drive.",
			"ja-JP": "Google Drive内のApps Scriptプロジェクト一覧を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"query":     {Type: "string", Description: "Search query (searches in project name)"},
				"page_size": {Type: "number", Description: "Maximum results (1-100). Default: 20"},
			},
		},
	},
	{
		ID:   "google_apps_script:get_project",
		Name: "get_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Get metadata for a script project.",
			"ja-JP": "スクリプトプロジェクトのメタデータを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id": {Type: "string", Description: "Script project ID"},
			},
			Required: []string{"script_id"},
		},
	},
	{
		ID:   "google_apps_script:create_project",
		Name: "create_project",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new standalone script project.",
			"ja-JP": "新しいスタンドアロンスクリプトプロジェクトを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"title":     {Type: "string", Description: "Project title"},
				"parent_id": {Type: "string", Description: "Parent folder ID in Google Drive (optional)"},
			},
			Required: []string{"title"},
		},
	},
	{
		ID:   "google_apps_script:get_content",
		Name: "get_content",
		Descriptions: modules.LocalizedText{
			"en-US": "Get the content (source code and metadata) of a script project.",
			"ja-JP": "スクリプトプロジェクトのコンテンツ（ソースコードとメタデータ）を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id":      {Type: "string", Description: "Script project ID"},
				"version_number": {Type: "number", Description: "Version number to retrieve (optional, defaults to HEAD)"},
			},
			Required: []string{"script_id"},
		},
	},
	{
		ID:   "google_apps_script:update_content",
		Name: "update_content",
		Descriptions: modules.LocalizedText{
			"en-US": "Update the content of a script project. Replaces all files.",
			"ja-JP": "スクリプトプロジェクトのコンテンツを更新します。すべてのファイルを置き換えます。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id": {Type: "string", Description: "Script project ID"},
				"files":     {Type: "array", Description: "Array of file objects: [{name, type, source}]. Type: 'SERVER_JS' for .gs files, 'HTML' for .html files, 'JSON' for appsscript.json"},
			},
			Required: []string{"script_id", "files"},
		},
	},
	// =========================================================================
	// Version Operations
	// =========================================================================
	{
		ID:   "google_apps_script:list_versions",
		Name: "list_versions",
		Descriptions: modules.LocalizedText{
			"en-US": "List all versions of a script project.",
			"ja-JP": "スクリプトプロジェクトのすべてのバージョンを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id":  {Type: "string", Description: "Script project ID"},
				"page_size":  {Type: "number", Description: "Maximum results (1-50). Default: 50"},
				"page_token": {Type: "string", Description: "Pagination token"},
			},
			Required: []string{"script_id"},
		},
	},
	{
		ID:   "google_apps_script:get_version",
		Name: "get_version",
		Descriptions: modules.LocalizedText{
			"en-US": "Get a specific version of a script project.",
			"ja-JP": "スクリプトプロジェクトの特定のバージョンを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id":      {Type: "string", Description: "Script project ID"},
				"version_number": {Type: "number", Description: "Version number"},
			},
			Required: []string{"script_id", "version_number"},
		},
	},
	{
		ID:   "google_apps_script:create_version",
		Name: "create_version",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new immutable version of the script project.",
			"ja-JP": "スクリプトプロジェクトの新しいイミュータブルバージョンを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id":   {Type: "string", Description: "Script project ID"},
				"description": {Type: "string", Description: "Version description"},
			},
			Required: []string{"script_id"},
		},
	},
	// =========================================================================
	// Deployment Operations
	// =========================================================================
	{
		ID:   "google_apps_script:list_deployments",
		Name: "list_deployments",
		Descriptions: modules.LocalizedText{
			"en-US": "List all deployments of a script project.",
			"ja-JP": "スクリプトプロジェクトのすべてのデプロイメントを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id":  {Type: "string", Description: "Script project ID"},
				"page_size":  {Type: "number", Description: "Maximum results (1-50). Default: 50"},
				"page_token": {Type: "string", Description: "Pagination token"},
			},
			Required: []string{"script_id"},
		},
	},
	{
		ID:   "google_apps_script:get_deployment",
		Name: "get_deployment",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific deployment.",
			"ja-JP": "特定のデプロイメントの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id":     {Type: "string", Description: "Script project ID"},
				"deployment_id": {Type: "string", Description: "Deployment ID"},
			},
			Required: []string{"script_id", "deployment_id"},
		},
	},
	{
		ID:   "google_apps_script:create_deployment",
		Name: "create_deployment",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new deployment for a script project.",
			"ja-JP": "スクリプトプロジェクトの新しいデプロイメントを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id":      {Type: "string", Description: "Script project ID"},
				"version_number": {Type: "number", Description: "Version number to deploy"},
				"description":    {Type: "string", Description: "Deployment description"},
			},
			Required: []string{"script_id", "version_number"},
		},
	},
	{
		ID:   "google_apps_script:update_deployment",
		Name: "update_deployment",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing deployment.",
			"ja-JP": "既存のデプロイメントを更新します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id":      {Type: "string", Description: "Script project ID"},
				"deployment_id":  {Type: "string", Description: "Deployment ID to update"},
				"version_number": {Type: "number", Description: "New version number to deploy"},
				"description":    {Type: "string", Description: "New deployment description"},
			},
			Required: []string{"script_id", "deployment_id"},
		},
	},
	{
		ID:   "google_apps_script:delete_deployment",
		Name: "delete_deployment",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a deployment.",
			"ja-JP": "デプロイメントを削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id":     {Type: "string", Description: "Script project ID"},
				"deployment_id": {Type: "string", Description: "Deployment ID to delete"},
			},
			Required: []string{"script_id", "deployment_id"},
		},
	},
	// =========================================================================
	// Execution Operations
	// =========================================================================
	{
		ID:   "google_apps_script:run_function",
		Name: "run_function",
		Descriptions: modules.LocalizedText{
			"en-US": "Execute a function in a script project. The project must be deployed as an API executable.",
			"ja-JP": "スクリプトプロジェクト内の関数を実行します。プロジェクトはAPI実行可能としてデプロイされている必要があります。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id":     {Type: "string", Description: "Script project ID"},
				"function_name": {Type: "string", Description: "Name of the function to execute"},
				"parameters":    {Type: "array", Description: "Parameters to pass to the function (optional)"},
				"dev_mode":      {Type: "boolean", Description: "Execute in development mode (uses HEAD instead of deployed version). Default: false"},
			},
			Required: []string{"script_id", "function_name"},
		},
	},
	{
		ID:   "google_apps_script:list_executions",
		Name: "list_executions",
		Descriptions: modules.LocalizedText{
			"en-US": "List recent executions of a script project.",
			"ja-JP": "スクリプトプロジェクトの最近の実行履歴を一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id":     {Type: "string", Description: "Script project ID"},
				"function_name": {Type: "string", Description: "Filter by function name (optional)"},
				"page_size":     {Type: "number", Description: "Maximum results (1-50). Default: 20"},
				"page_token":    {Type: "string", Description: "Pagination token"},
			},
			Required: []string{"script_id"},
		},
	},
	// =========================================================================
	// Process Operations
	// =========================================================================
	{
		ID:   "google_apps_script:list_processes",
		Name: "list_processes",
		Descriptions: modules.LocalizedText{
			"en-US": "List user's script processes with filtering options.",
			"ja-JP": "ユーザーのスクリプトプロセスをフィルタリングオプション付きで一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id":     {Type: "string", Description: "Filter by script project ID (optional)"},
				"function_name": {Type: "string", Description: "Filter by function name (optional)"},
				"statuses":      {Type: "array", Description: "Filter by status: RUNNING, PAUSED, COMPLETED, CANCELED, FAILED, TIMED_OUT, UNKNOWN (optional)"},
				"types":         {Type: "array", Description: "Filter by type: ADD_ON, EXECUTION_API, TIME_DRIVEN, TRIGGER, WEBAPP, EDITOR (optional)"},
				"page_size":     {Type: "number", Description: "Maximum results (1-50). Default: 50"},
				"page_token":    {Type: "string", Description: "Pagination token"},
			},
		},
	},
	// =========================================================================
	// Metrics Operations
	// =========================================================================
	{
		ID:   "google_apps_script:get_metrics",
		Name: "get_metrics",
		Descriptions: modules.LocalizedText{
			"en-US": "Get execution metrics for a script project.",
			"ja-JP": "スクリプトプロジェクトの実行メトリクスを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"script_id":           {Type: "string", Description: "Script project ID"},
				"metrics_granularity": {Type: "string", Description: "Metrics granularity: 'WEEKLY' or 'DAILY'. Default: 'WEEKLY'"},
				"deployment_id":       {Type: "string", Description: "Filter metrics by deployment ID (optional)"},
			},
			Required: []string{"script_id"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"list_projects":     listProjects,
	"get_project":       getProject,
	"create_project":    createProject,
	"get_content":       getContent,
	"update_content":    updateContent,
	"list_versions":     listVersions,
	"get_version":       getVersion,
	"create_version":    createVersion,
	"list_deployments":  listDeployments,
	"get_deployment":    getDeployment,
	"create_deployment": createDeployment,
	"update_deployment": updateDeployment,
	"delete_deployment": deleteDeployment,
	"run_function":      runFunction,
	"list_executions":   listExecutions,
	"list_processes":    listProcesses,
	"get_metrics":       getMetrics,
}

// =============================================================================
// Project Operations
// =============================================================================

// listProjects uses the Drive API to search for Apps Script files.
func listProjects(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newDriveClient(ctx)
	if err != nil {
		return "", err
	}

	q := "mimeType='application/vnd.google-apps.script'"
	if searchQuery, ok := params["query"].(string); ok && searchQuery != "" {
		q += fmt.Sprintf(" and name contains '%s'", searchQuery)
	}

	pageSize := 20
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		pageSize = int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
	}

	resp, err := cli.ListFiles(ctx, driveGen.ListFilesParams{
		Q:        driveGen.NewOptString(q),
		PageSize: driveGen.NewOptInt(pageSize),
		Fields:   driveGen.NewOptString("files(id,name,createdTime,modifiedTime,webViewLink)"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}
	return toJSON(resp)
}

func getProject(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)

	resp, err := cli.GetProject(ctx, gen.GetProjectParams{ScriptId: scriptID})
	if err != nil {
		return "", fmt.Errorf("failed to get project: %w", err)
	}
	return toJSON(resp)
}

func createProject(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	title, _ := params["title"].(string)

	reqBody := &gen.CreateProjectRequest{Title: title}
	if parentID, ok := params["parent_id"].(string); ok && parentID != "" {
		reqBody.ParentId = gen.NewOptString(parentID)
	}

	resp, err := cli.CreateProject(ctx, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}
	return toJSON(resp)
}

func getContent(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)

	p := gen.GetContentParams{ScriptId: scriptID}
	if vn, ok := params["version_number"].(float64); ok && vn > 0 {
		p.VersionNumber = gen.NewOptInt(int(vn))
	}

	resp, err := cli.GetContent(ctx, p)
	if err != nil {
		return "", fmt.Errorf("failed to get content: %w", err)
	}
	return toJSON(resp)
}

func updateContent(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)
	filesRaw, _ := params["files"].([]interface{})

	var files []gen.ScriptFileInput
	for _, f := range filesRaw {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		fi := gen.ScriptFileInput{}
		if name, ok := fm["name"].(string); ok {
			fi.Name = gen.NewOptString(name)
		}
		if typ, ok := fm["type"].(string); ok {
			fi.Type = gen.NewOptString(typ)
		}
		if src, ok := fm["source"].(string); ok {
			fi.Source = gen.NewOptString(src)
		}
		files = append(files, fi)
	}

	resp, err := cli.UpdateContent(ctx, &gen.UpdateContentRequest{Files: files}, gen.UpdateContentParams{ScriptId: scriptID})
	if err != nil {
		return "", fmt.Errorf("failed to update content: %w", err)
	}
	return toJSON(resp)
}

// =============================================================================
// Version Operations
// =============================================================================

func listVersions(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)

	p := gen.ListVersionsParams{ScriptId: scriptID}
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		size := int(ps)
		if size > 50 {
			size = 50
		}
		p.PageSize = gen.NewOptInt(size)
	}
	if pt, ok := params["page_token"].(string); ok && pt != "" {
		p.PageToken = gen.NewOptString(pt)
	}

	resp, err := cli.ListVersions(ctx, p)
	if err != nil {
		return "", fmt.Errorf("failed to list versions: %w", err)
	}
	return toJSON(resp)
}

func getVersion(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)
	versionNumber, _ := params["version_number"].(float64)

	resp, err := cli.GetVersion(ctx, gen.GetVersionParams{
		ScriptId:      scriptID,
		VersionNumber: int(versionNumber),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}
	return toJSON(resp)
}

func createVersion(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)

	reqBody := &gen.CreateVersionRequest{}
	if desc, ok := params["description"].(string); ok && desc != "" {
		reqBody.Description = gen.NewOptString(desc)
	}

	resp, err := cli.CreateVersion(ctx, reqBody, gen.CreateVersionParams{ScriptId: scriptID})
	if err != nil {
		return "", fmt.Errorf("failed to create version: %w", err)
	}
	return toJSON(resp)
}

// =============================================================================
// Deployment Operations
// =============================================================================

func listDeployments(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)

	p := gen.ListDeploymentsParams{ScriptId: scriptID}
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		size := int(ps)
		if size > 50 {
			size = 50
		}
		p.PageSize = gen.NewOptInt(size)
	}
	if pt, ok := params["page_token"].(string); ok && pt != "" {
		p.PageToken = gen.NewOptString(pt)
	}

	resp, err := cli.ListDeployments(ctx, p)
	if err != nil {
		return "", fmt.Errorf("failed to list deployments: %w", err)
	}
	return toJSON(resp)
}

func getDeployment(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)
	deploymentID, _ := params["deployment_id"].(string)

	resp, err := cli.GetDeployment(ctx, gen.GetDeploymentParams{
		ScriptId:     scriptID,
		DeploymentId: deploymentID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get deployment: %w", err)
	}
	return toJSON(resp)
}

func createDeployment(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)
	versionNumber, _ := params["version_number"].(float64)

	reqBody := &gen.CreateDeploymentRequest{VersionNumber: int(versionNumber)}
	if desc, ok := params["description"].(string); ok && desc != "" {
		reqBody.Description = gen.NewOptString(desc)
	}

	resp, err := cli.CreateDeployment(ctx, reqBody, gen.CreateDeploymentParams{ScriptId: scriptID})
	if err != nil {
		return "", fmt.Errorf("failed to create deployment: %w", err)
	}
	return toJSON(resp)
}

func updateDeployment(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)
	deploymentID, _ := params["deployment_id"].(string)

	config := gen.DeploymentConfig{
		ScriptId: gen.NewOptString(scriptID),
	}
	if vn, ok := params["version_number"].(float64); ok && vn > 0 {
		config.VersionNumber = gen.NewOptInt(int(vn))
	}
	if desc, ok := params["description"].(string); ok && desc != "" {
		config.Description = gen.NewOptString(desc)
	}

	reqBody := &gen.UpdateDeploymentRequest{
		DeploymentConfig: gen.NewOptDeploymentConfig(config),
	}

	resp, err := cli.UpdateDeployment(ctx, reqBody, gen.UpdateDeploymentParams{
		ScriptId:     scriptID,
		DeploymentId: deploymentID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to update deployment: %w", err)
	}
	return toJSON(resp)
}

func deleteDeployment(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)
	deploymentID, _ := params["deployment_id"].(string)

	err = cli.DeleteDeployment(ctx, gen.DeleteDeploymentParams{
		ScriptId:     scriptID,
		DeploymentId: deploymentID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to delete deployment: %w", err)
	}
	return `{"success":true}`, nil
}

// =============================================================================
// Execution Operations
// =============================================================================

func runFunction(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)
	functionName, _ := params["function_name"].(string)

	reqBody := &gen.ExecutionRequest{Function: functionName}

	if parameters, ok := params["parameters"].([]interface{}); ok && len(parameters) > 0 {
		for _, p := range parameters {
			raw, err := json.Marshal(p)
			if err != nil {
				continue
			}
			reqBody.Parameters = append(reqBody.Parameters, jx.Raw(raw))
		}
	}

	if devMode, ok := params["dev_mode"].(bool); ok && devMode {
		reqBody.DevMode = gen.NewOptBool(true)
	}

	resp, err := cli.RunScript(ctx, reqBody, gen.RunScriptParams{ScriptId: scriptID})
	if err != nil {
		return "", fmt.Errorf("failed to run function: %w", err)
	}
	return toJSON(resp)
}

func listExecutions(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)

	p := gen.ListScriptProcessesParams{ScriptId: scriptID}
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		size := int(ps)
		if size > 50 {
			size = 50
		}
		p.PageSize = gen.NewOptInt(size)
	}
	if pt, ok := params["page_token"].(string); ok && pt != "" {
		p.PageToken = gen.NewOptString(pt)
	}
	if fn, ok := params["function_name"].(string); ok && fn != "" {
		p.ScriptProcessFilterFunctionName = gen.NewOptString(fn)
	}

	resp, err := cli.ListScriptProcesses(ctx, p)
	if err != nil {
		return "", fmt.Errorf("failed to list executions: %w", err)
	}
	return toJSON(resp)
}

// =============================================================================
// Process Operations
// =============================================================================

func listProcesses(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	p := gen.ListProcessesParams{}
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		size := int(ps)
		if size > 50 {
			size = 50
		}
		p.PageSize = gen.NewOptInt(size)
	}
	if pt, ok := params["page_token"].(string); ok && pt != "" {
		p.PageToken = gen.NewOptString(pt)
	}
	if sid, ok := params["script_id"].(string); ok && sid != "" {
		p.UserProcessFilterScriptId = gen.NewOptString(sid)
	}
	if fn, ok := params["function_name"].(string); ok && fn != "" {
		p.UserProcessFilterFunctionName = gen.NewOptString(fn)
	}
	if statuses, ok := params["statuses"].([]interface{}); ok && len(statuses) > 0 {
		for _, s := range statuses {
			if status, ok := s.(string); ok {
				p.UserProcessFilterStatuses = append(p.UserProcessFilterStatuses, status)
			}
		}
	}
	if types, ok := params["types"].([]interface{}); ok && len(types) > 0 {
		for _, t := range types {
			if typ, ok := t.(string); ok {
				p.UserProcessFilterTypes = append(p.UserProcessFilterTypes, typ)
			}
		}
	}

	resp, err := cli.ListProcesses(ctx, p)
	if err != nil {
		return "", fmt.Errorf("failed to list processes: %w", err)
	}
	return toJSON(resp)
}

// =============================================================================
// Metrics Operations
// =============================================================================

func getMetrics(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	scriptID, _ := params["script_id"].(string)

	granularity := gen.GetMetricsMetricsGranularityWEEKLY
	if g, ok := params["metrics_granularity"].(string); ok && g == "DAILY" {
		granularity = gen.GetMetricsMetricsGranularityDAILY
	}

	p := gen.GetMetricsParams{
		ScriptId:           scriptID,
		MetricsGranularity: granularity,
	}
	if deploymentID, ok := params["deployment_id"].(string); ok && deploymentID != "" {
		p.MetricsFilterDeploymentId = gen.NewOptString(deploymentID)
	}

	resp, err := cli.GetMetrics(ctx, p)
	if err != nil {
		return "", fmt.Errorf("failed to get metrics: %w", err)
	}
	return toJSON(resp)
}
