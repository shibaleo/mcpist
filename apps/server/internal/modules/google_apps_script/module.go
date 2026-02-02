package google_apps_script

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
	appsScriptAPIBase  = "https://script.googleapis.com/v1"
	googleDriveAPIBase = "https://www.googleapis.com/drive/v3"
	appsScriptVersion  = "v1"
	googleTokenURL     = "https://oauth2.googleapis.com/token"
	tokenRefreshBuffer = 5 * 60 // Refresh 5 minutes before expiry
)

var client = httpclient.New()

// GoogleAppsScriptModule implements the Module interface for Google Apps Script API
type GoogleAppsScriptModule struct{}

// New creates a new GoogleAppsScriptModule instance
func New() *GoogleAppsScriptModule {
	return &GoogleAppsScriptModule{}
}

// Module descriptions in multiple languages
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Google Apps Script API - Manage script projects, deployments, versions, and executions",
	"ja-JP": "Google Apps Script API - スクリプトプロジェクト、デプロイ、バージョン、実行の管理",
}

// Name returns the module name
func (m *GoogleAppsScriptModule) Name() string {
	return "google_apps_script"
}

// Descriptions returns the module descriptions in all languages
func (m *GoogleAppsScriptModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description for a specific language
func (m *GoogleAppsScriptModule) Description(lang string) string {
	return modules.GetLocalizedText(moduleDescriptions, lang)
}

// APIVersion returns the Google Apps Script API version
func (m *GoogleAppsScriptModule) APIVersion() string {
	return appsScriptVersion
}

// Tools returns all available tools
func (m *GoogleAppsScriptModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *GoogleAppsScriptModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for Google Apps Script)
func (m *GoogleAppsScriptModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *GoogleAppsScriptModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *store.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[google_apps_script] No auth context")
		return nil
	}
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "google_apps_script")
	if err != nil {
		log.Printf("[google_apps_script] GetModuleToken error: %v", err)
		return nil
	}
	log.Printf("[google_apps_script] Got credentials: auth_type=%s, has_access_token=%v", credentials.AuthType, credentials.AccessToken != "")

	// Check if token needs refresh (OAuth2 only)
	if credentials.AuthType == store.AuthTypeOAuth2 && credentials.RefreshToken != "" {
		if needsRefresh(credentials) {
			log.Printf("[google_apps_script] Token expired or expiring soon, refreshing...")
			refreshed, err := refreshToken(ctx, authCtx.UserID, credentials)
			if err != nil {
				log.Printf("[google_apps_script] Token refresh failed: %v", err)
				return credentials
			}
			log.Printf("[google_apps_script] Token refreshed successfully")
			return refreshed
		}
	}

	return credentials
}

// needsRefresh checks if the token is expired or will expire soon
func needsRefresh(creds *store.Credentials) bool {
	if creds.ExpiresAt == 0 {
		return false
	}
	now := time.Now().Unix()
	return now >= (int64(creds.ExpiresAt) - tokenRefreshBuffer)
}

// refreshToken exchanges the refresh token for a new access token
func refreshToken(ctx context.Context, userID string, creds *store.Credentials) (*store.Credentials, error) {
	oauthApp, err := store.GetTokenStore().GetOAuthAppCredentials(ctx, "google")
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth app credentials: %w", err)
	}

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
		return nil, fmt.Errorf("failed to send refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status: %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	expiresAt := time.Now().Unix() + int64(tokenResp.ExpiresIn)

	newCreds := &store.Credentials{
		AuthType:     creds.AuthType,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: creds.RefreshToken,
		ExpiresAt:    store.FlexibleTime(expiresAt),
	}

	if err := store.GetTokenStore().UpdateModuleToken(ctx, userID, "google_apps_script", newCreds); err != nil {
		log.Printf("[google_apps_script] Failed to update token in store: %v", err)
	}

	return newCreds, nil
}

// headers builds request headers with auth token
func headers(ctx context.Context) map[string]string {
	creds := getCredentials(ctx)
	if creds == nil {
		log.Printf("[google_apps_script] No credentials available")
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
				"files": {Type: "array", Description: "Array of file objects: [{name, type, source}]. Type: 'SERVER_JS' for .gs files, 'HTML' for .html files, 'JSON' for appsscript.json"},
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
				"script_id": {Type: "string", Description: "Script project ID"},
				"filter":    {Type: "string", Description: "Filter expression for metrics (optional)"},
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
	// Project operations
	"list_projects":  listProjects,
	"get_project":    getProject,
	"create_project": createProject,
	"get_content":    getContent,
	"update_content": updateContent,
	// Version operations
	"list_versions":  listVersions,
	"get_version":    getVersion,
	"create_version": createVersion,
	// Deployment operations
	"list_deployments":  listDeployments,
	"get_deployment":    getDeployment,
	"create_deployment": createDeployment,
	"update_deployment": updateDeployment,
	"delete_deployment": deleteDeployment,
	// Execution operations
	"run_function":    runFunction,
	"list_executions": listExecutions,
	// Process operations
	"list_processes": listProcesses,
	// Metrics operations
	"get_metrics": getMetrics,
}

// =============================================================================
// Project Operations
// =============================================================================

func listProjects(ctx context.Context, params map[string]any) (string, error) {
	query := url.Values{}
	q := "mimeType='application/vnd.google-apps.script'"
	if searchQuery, ok := params["query"].(string); ok && searchQuery != "" {
		q += fmt.Sprintf(" and name contains '%s'", searchQuery)
	}
	query.Set("q", q)

	pageSize := 20
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		pageSize = int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
	}
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))
	query.Set("fields", "files(id,name,createdTime,modifiedTime,webViewLink)")

	endpoint := fmt.Sprintf("%s/files?%s", googleDriveAPIBase, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getProject(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)

	endpoint := fmt.Sprintf("%s/projects/%s", appsScriptAPIBase, url.PathEscape(scriptID))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createProject(ctx context.Context, params map[string]any) (string, error) {
	title, _ := params["title"].(string)

	body := map[string]interface{}{
		"title": title,
	}

	if parentID, ok := params["parent_id"].(string); ok && parentID != "" {
		body["parentId"] = parentID
	}

	endpoint := fmt.Sprintf("%s/projects", appsScriptAPIBase)
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getContent(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)

	endpoint := fmt.Sprintf("%s/projects/%s/content", appsScriptAPIBase, url.PathEscape(scriptID))

	if versionNumber, ok := params["version_number"].(float64); ok && versionNumber > 0 {
		endpoint += fmt.Sprintf("?versionNumber=%d", int(versionNumber))
	}

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updateContent(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)
	files, _ := params["files"].([]interface{})

	body := map[string]interface{}{
		"files": files,
	}

	endpoint := fmt.Sprintf("%s/projects/%s/content", appsScriptAPIBase, url.PathEscape(scriptID))
	respBody, err := client.DoJSON("PUT", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Version Operations
// =============================================================================

func listVersions(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)

	query := url.Values{}
	pageSize := 50
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		pageSize = int(ps)
		if pageSize > 50 {
			pageSize = 50
		}
	}
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))

	if pageToken, ok := params["page_token"].(string); ok && pageToken != "" {
		query.Set("pageToken", pageToken)
	}

	endpoint := fmt.Sprintf("%s/projects/%s/versions?%s", appsScriptAPIBase, url.PathEscape(scriptID), query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getVersion(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)
	versionNumber, _ := params["version_number"].(float64)

	endpoint := fmt.Sprintf("%s/projects/%s/versions/%d", appsScriptAPIBase, url.PathEscape(scriptID), int(versionNumber))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createVersion(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)

	body := map[string]interface{}{}
	if description, ok := params["description"].(string); ok && description != "" {
		body["description"] = description
	}

	endpoint := fmt.Sprintf("%s/projects/%s/versions", appsScriptAPIBase, url.PathEscape(scriptID))
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Deployment Operations
// =============================================================================

func listDeployments(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)

	query := url.Values{}
	pageSize := 50
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		pageSize = int(ps)
		if pageSize > 50 {
			pageSize = 50
		}
	}
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))

	if pageToken, ok := params["page_token"].(string); ok && pageToken != "" {
		query.Set("pageToken", pageToken)
	}

	endpoint := fmt.Sprintf("%s/projects/%s/deployments?%s", appsScriptAPIBase, url.PathEscape(scriptID), query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getDeployment(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)
	deploymentID, _ := params["deployment_id"].(string)

	endpoint := fmt.Sprintf("%s/projects/%s/deployments/%s", appsScriptAPIBase, url.PathEscape(scriptID), url.PathEscape(deploymentID))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createDeployment(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)
	versionNumber, _ := params["version_number"].(float64)

	body := map[string]interface{}{
		"versionNumber": int(versionNumber),
	}

	if description, ok := params["description"].(string); ok && description != "" {
		body["description"] = description
	}

	endpoint := fmt.Sprintf("%s/projects/%s/deployments", appsScriptAPIBase, url.PathEscape(scriptID))
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updateDeployment(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)
	deploymentID, _ := params["deployment_id"].(string)

	deploymentConfig := map[string]interface{}{
		"scriptId": scriptID,
	}

	if versionNumber, ok := params["version_number"].(float64); ok && versionNumber > 0 {
		deploymentConfig["versionNumber"] = int(versionNumber)
	}
	if description, ok := params["description"].(string); ok && description != "" {
		deploymentConfig["description"] = description
	}

	body := map[string]interface{}{
		"deploymentConfig": deploymentConfig,
	}

	endpoint := fmt.Sprintf("%s/projects/%s/deployments/%s", appsScriptAPIBase, url.PathEscape(scriptID), url.PathEscape(deploymentID))
	respBody, err := client.DoJSON("PUT", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteDeployment(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)
	deploymentID, _ := params["deployment_id"].(string)

	endpoint := fmt.Sprintf("%s/projects/%s/deployments/%s", appsScriptAPIBase, url.PathEscape(scriptID), url.PathEscape(deploymentID))
	respBody, err := client.DoJSON("DELETE", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	if respBody == nil {
		return `{"success": true}`, nil
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Execution Operations
// =============================================================================

func runFunction(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)
	functionName, _ := params["function_name"].(string)

	body := map[string]interface{}{
		"function": functionName,
	}

	if parameters, ok := params["parameters"].([]interface{}); ok && len(parameters) > 0 {
		body["parameters"] = parameters
	}

	if devMode, ok := params["dev_mode"].(bool); ok && devMode {
		body["devMode"] = true
	}

	endpoint := fmt.Sprintf("%s/scripts/%s:run", appsScriptAPIBase, url.PathEscape(scriptID))
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func listExecutions(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)

	query := url.Values{}
	query.Set("scriptId", scriptID)

	pageSize := 20
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		pageSize = int(ps)
		if pageSize > 50 {
			pageSize = 50
		}
	}
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))

	if pageToken, ok := params["page_token"].(string); ok && pageToken != "" {
		query.Set("pageToken", pageToken)
	}

	// Build filter for script processes
	filter := fmt.Sprintf("scriptId=%s", scriptID)
	if functionName, ok := params["function_name"].(string); ok && functionName != "" {
		filter += fmt.Sprintf(" AND functionName=%s", functionName)
	}

	endpoint := fmt.Sprintf("%s/processes:listScriptProcesses?%s", appsScriptAPIBase, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Process Operations
// =============================================================================

func listProcesses(ctx context.Context, params map[string]any) (string, error) {
	query := url.Values{}

	pageSize := 50
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		pageSize = int(ps)
		if pageSize > 50 {
			pageSize = 50
		}
	}
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))

	if pageToken, ok := params["page_token"].(string); ok && pageToken != "" {
		query.Set("pageToken", pageToken)
	}

	// Build userProcessFilter
	filterParts := []string{}
	if scriptID, ok := params["script_id"].(string); ok && scriptID != "" {
		filterParts = append(filterParts, fmt.Sprintf("userProcessFilter.scriptId=%s", scriptID))
	}
	if functionName, ok := params["function_name"].(string); ok && functionName != "" {
		filterParts = append(filterParts, fmt.Sprintf("userProcessFilter.functionName=%s", functionName))
	}
	if statuses, ok := params["statuses"].([]interface{}); ok && len(statuses) > 0 {
		for _, s := range statuses {
			if status, ok := s.(string); ok {
				filterParts = append(filterParts, fmt.Sprintf("userProcessFilter.statuses=%s", status))
			}
		}
	}
	if types, ok := params["types"].([]interface{}); ok && len(types) > 0 {
		for _, t := range types {
			if typ, ok := t.(string); ok {
				filterParts = append(filterParts, fmt.Sprintf("userProcessFilter.types=%s", typ))
			}
		}
	}

	for _, part := range filterParts {
		query.Add("userProcessFilter", part)
	}

	endpoint := fmt.Sprintf("%s/processes?%s", appsScriptAPIBase, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Metrics Operations
// =============================================================================

func getMetrics(ctx context.Context, params map[string]any) (string, error) {
	scriptID, _ := params["script_id"].(string)

	query := url.Values{}
	if filter, ok := params["filter"].(string); ok && filter != "" {
		query.Set("metricsFilter.deploymentId", filter)
	}

	endpoint := fmt.Sprintf("%s/projects/%s/metrics", appsScriptAPIBase, url.PathEscape(scriptID))
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}
