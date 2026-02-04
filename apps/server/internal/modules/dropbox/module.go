package dropbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	dropboxAPIBase     = "https://api.dropboxapi.com/2"
	dropboxContentBase = "https://content.dropboxapi.com/2"
	dropboxTokenURL    = "https://api.dropboxapi.com/oauth2/token"
	dropboxVersion     = "v2"
	tokenRefreshBuffer = 5 * 60 // Refresh 5 minutes before expiry
)

var client = httpclient.New()

// DropboxModule implements the Module interface for Dropbox API
type DropboxModule struct{}

// New creates a new DropboxModule instance
func New() *DropboxModule {
	return &DropboxModule{}
}

// Module descriptions in multiple languages
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Dropbox API - File and folder operations (list, search, upload, download, share)",
	"ja-JP": "Dropbox API - ファイルとフォルダの操作（一覧、検索、アップロード、ダウンロード、共有）",
}

// Name returns the module name
func (m *DropboxModule) Name() string {
	return "dropbox"
}

// Descriptions returns the module descriptions in all languages
func (m *DropboxModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description for a specific language
func (m *DropboxModule) Description(lang string) string {
	return modules.GetLocalizedText(moduleDescriptions, lang)
}

// APIVersion returns the Dropbox API version
func (m *DropboxModule) APIVersion() string {
	return dropboxVersion
}

// Tools returns all available tools
func (m *DropboxModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *DropboxModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for Dropbox)
func (m *DropboxModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *DropboxModule) ReadResource(ctx context.Context, uri string) (string, error) {
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
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "dropbox")
	if err != nil {
		return nil
	}

	// Check if token needs refresh (OAuth2 only)
	if credentials.AuthType == store.AuthTypeOAuth2 && credentials.RefreshToken != "" {
		if needsRefresh(credentials) {
			refreshed, err := refreshToken(ctx, authCtx.UserID, credentials)
			if err != nil {
				return credentials
			}
			return refreshed
		}
	}

	return credentials
}

func needsRefresh(creds *store.Credentials) bool {
	if creds.ExpiresAt == 0 {
		return false
	}
	now := time.Now().Unix()
	return now >= (int64(creds.ExpiresAt) - tokenRefreshBuffer)
}

func refreshToken(ctx context.Context, userID string, creds *store.Credentials) (*store.Credentials, error) {
	oauthApp, err := store.GetTokenStore().GetOAuthAppCredentials(ctx, "dropbox")
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth app credentials: %w", err)
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", creds.RefreshToken)
	data.Set("client_id", oauthApp.ClientID)
	data.Set("client_secret", oauthApp.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", dropboxTokenURL, strings.NewReader(data.Encode()))
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
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	newCreds := &store.Credentials{
		AuthType:     store.AuthTypeOAuth2,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: creds.RefreshToken,
		ExpiresAt:    store.FlexibleTime(time.Now().Unix() + tokenResp.ExpiresIn),
	}

	err = store.GetTokenStore().UpdateModuleToken(ctx, userID, "dropbox", newCreds)
	if err != nil {
		// Continue anyway, the token is still valid for this request
	}

	return newCreds, nil
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
// Content Endpoint Helpers
// =============================================================================

// doContentDownload handles Dropbox content download endpoints.
// Parameters are sent via Dropbox-API-Arg header, response body is file content.
func doContentDownload(ctx context.Context, path string, apiArg interface{}) (string, error) {
	argJSON, err := json.Marshal(apiArg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal API arg: %w", err)
	}

	endpoint := dropboxContentBase + path
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	creds := getCredentials(contextFromRequest(ctx))
	if creds != nil {
		req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
	}
	req.Header.Set("Dropbox-API-Arg", string(argJSON))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("download failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Read content (limit to 1MB)
	const maxSize = 1 * 1024 * 1024
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}
	truncated := len(content) > maxSize
	if truncated {
		content = content[:maxSize]
	}

	// Metadata comes back in Dropbox-API-Result header
	apiResult := resp.Header.Get("Dropbox-API-Result")

	result := map[string]interface{}{
		"content":   string(content),
		"truncated": truncated,
	}
	if apiResult != "" {
		result["metadata"] = json.RawMessage(apiResult)
	}
	resultBytes, _ := json.Marshal(result)
	return httpclient.PrettyJSON(resultBytes), nil
}

// doContentUpload handles Dropbox content upload endpoints.
// Parameters are sent via Dropbox-API-Arg header, request body is file content.
func doContentUpload(ctx context.Context, path string, apiArg interface{}, content string) (string, error) {
	argJSON, err := json.Marshal(apiArg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal API arg: %w", err)
	}

	endpoint := dropboxContentBase + path
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	creds := getCredentials(contextFromRequest(ctx))
	if creds != nil {
		req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
	}
	req.Header.Set("Dropbox-API-Arg", string(argJSON))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, string(respBody))
	}
	return httpclient.PrettyJSON(respBody), nil
}

// contextFromRequest returns the context as-is (for content endpoints that
// need to pass through the original context for credential resolution).
func contextFromRequest(ctx context.Context) context.Context {
	return ctx
}

// =============================================================================
// Tool Definitions
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolDefinitions = []modules.Tool{
	// =========================================================================
	// Read Tools
	// =========================================================================
	{
		ID:   "dropbox:get_current_account",
		Name: "get_current_account",
		Descriptions: modules.LocalizedText{
			"en-US": "Get information about the currently authenticated Dropbox user.",
			"ja-JP": "現在認証されているDropboxユーザーの情報を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "dropbox:get_space_usage",
		Name: "get_space_usage",
		Descriptions: modules.LocalizedText{
			"en-US": "Get the space usage information for the current account.",
			"ja-JP": "現在のアカウントのストレージ使用量情報を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "dropbox:list_folder",
		Name: "list_folder",
		Descriptions: modules.LocalizedText{
			"en-US": "List contents of a folder in Dropbox. Returns files and sub-folders.",
			"ja-JP": "Dropbox内のフォルダの内容を一覧表示します。ファイルとサブフォルダを返します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"path":               {Type: "string", Description: "Folder path (e.g., '/Documents' or '' for root)"},
				"recursive":          {Type: "boolean", Description: "List contents recursively (default: false)"},
				"include_deleted":    {Type: "boolean", Description: "Include deleted entries (default: false)"},
				"include_media_info": {Type: "boolean", Description: "Include media info for photos/videos (default: false)"},
				"limit":              {Type: "number", Description: "Maximum number of results (1-2000)"},
			},
			Required: []string{"path"},
		},
	},
	{
		ID:   "dropbox:list_folder_continue",
		Name: "list_folder_continue",
		Descriptions: modules.LocalizedText{
			"en-US": "Continue listing folder contents from a previous list_folder call using the cursor.",
			"ja-JP": "前回のlist_folder呼び出しのカーソルを使用してフォルダの内容のリスト表示を継続します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"cursor": {Type: "string", Description: "Cursor from a previous list_folder or list_folder_continue response"},
			},
			Required: []string{"cursor"},
		},
	},
	{
		ID:   "dropbox:get_metadata",
		Name: "get_metadata",
		Descriptions: modules.LocalizedText{
			"en-US": "Get metadata for a file or folder by path.",
			"ja-JP": "パスでファイルまたはフォルダのメタデータを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"path":               {Type: "string", Description: "File or folder path (e.g., '/Documents/report.pdf')"},
				"include_media_info": {Type: "boolean", Description: "Include media info (default: false)"},
				"include_deleted":    {Type: "boolean", Description: "Include deleted (default: false)"},
			},
			Required: []string{"path"},
		},
	},
	{
		ID:   "dropbox:search_files",
		Name: "search_files",
		Descriptions: modules.LocalizedText{
			"en-US": "Search for files and folders by name or content.",
			"ja-JP": "名前またはコンテンツでファイルとフォルダを検索します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"query":           {Type: "string", Description: "Search query string"},
				"path":            {Type: "string", Description: "Scope search to this path"},
				"max_results":     {Type: "number", Description: "Maximum results (1-1000, default: 100)"},
				"file_categories": {Type: "array", Description: "Filter by category: image, document, pdf, spreadsheet, presentation, audio, video, folder", Items: &modules.Property{Type: "string"}},
			},
			Required: []string{"query"},
		},
	},
	{
		ID:   "dropbox:read_file",
		Name: "read_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Read the text content of a file. Returns up to 1MB of text. Suitable for code, config, markdown, and other text files.",
			"ja-JP": "ファイルのテキスト内容を読み取ります。最大1MBのテキストを返します。コード、設定ファイル、マークダウン等のテキストファイルに適しています。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"path": {Type: "string", Description: "File path (e.g., '/Documents/readme.txt')"},
			},
			Required: []string{"path"},
		},
	},
	{
		ID:   "dropbox:list_shared_links",
		Name: "list_shared_links",
		Descriptions: modules.LocalizedText{
			"en-US": "List shared links for a file or folder, or all shared links if no path specified.",
			"ja-JP": "ファイルまたはフォルダの共有リンクを一覧表示します。パスを指定しない場合はすべての共有リンクを表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"path":   {Type: "string", Description: "File or folder path to list shared links for"},
				"cursor": {Type: "string", Description: "Cursor from a previous response for pagination"},
			},
		},
	},
	// =========================================================================
	// Write Tools
	// =========================================================================
	{
		ID:   "dropbox:write_file",
		Name: "write_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Create or overwrite a text file in Dropbox. Provide the file path and text content.",
			"ja-JP": "Dropboxにテキストファイルを作成または上書きします。ファイルパスとテキスト内容を指定します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"path":       {Type: "string", Description: "File path including filename (e.g., '/Documents/notes.md')"},
				"content":    {Type: "string", Description: "Text content to write"},
				"mode":       {Type: "string", Description: "Write mode: add (default, no overwrite), overwrite, or update"},
				"autorename": {Type: "boolean", Description: "Automatically rename if conflict (default: false)"},
				"mute":       {Type: "boolean", Description: "Suppress notifications (default: false)"},
			},
			Required: []string{"path", "content"},
		},
	},
	{
		ID:   "dropbox:create_folder",
		Name: "create_folder",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new folder in Dropbox.",
			"ja-JP": "Dropboxに新しいフォルダを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"path":       {Type: "string", Description: "Folder path to create (e.g., '/Documents/NewFolder')"},
				"autorename": {Type: "boolean", Description: "Automatically rename if conflict (default: false)"},
			},
			Required: []string{"path"},
		},
	},
	{
		ID:   "dropbox:copy_file",
		Name: "copy_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Copy a file or folder to a different location.",
			"ja-JP": "ファイルまたはフォルダを別の場所にコピーします。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"from_path":  {Type: "string", Description: "Source path"},
				"to_path":    {Type: "string", Description: "Destination path"},
				"autorename": {Type: "boolean", Description: "Automatically rename if conflict (default: false)"},
			},
			Required: []string{"from_path", "to_path"},
		},
	},
	{
		ID:   "dropbox:move_file",
		Name: "move_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Move a file or folder to a different location.",
			"ja-JP": "ファイルまたはフォルダを別の場所に移動します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"from_path":  {Type: "string", Description: "Source path"},
				"to_path":    {Type: "string", Description: "Destination path"},
				"autorename": {Type: "boolean", Description: "Automatically rename if conflict (default: false)"},
			},
			Required: []string{"from_path", "to_path"},
		},
	},
	{
		ID:   "dropbox:delete_file",
		Name: "delete_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a file or folder (moves to trash, recoverable for a limited time).",
			"ja-JP": "ファイルまたはフォルダを削除します（ゴミ箱に移動、一定期間復元可能）。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"path": {Type: "string", Description: "Path of file or folder to delete"},
			},
			Required: []string{"path"},
		},
	},
	{
		ID:   "dropbox:create_shared_link",
		Name: "create_shared_link",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a shared link for a file or folder.",
			"ja-JP": "ファイルまたはフォルダの共有リンクを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"path":                 {Type: "string", Description: "Path to create shared link for"},
				"requested_visibility": {Type: "string", Description: "Visibility: public, team_only, or password (default: public)"},
			},
			Required: []string{"path"},
		},
	},
	{
		ID:   "dropbox:list_revisions",
		Name: "list_revisions",
		Descriptions: modules.LocalizedText{
			"en-US": "List revisions of a file.",
			"ja-JP": "ファイルのリビジョンを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"path":  {Type: "string", Description: "File path"},
				"mode":  {Type: "string", Description: "Revision mode: path (default) or id"},
				"limit": {Type: "number", Description: "Maximum revisions to return (1-100, default: 10)"},
			},
			Required: []string{"path"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

var toolHandlers = map[string]toolHandler{
	// Read
	"get_current_account":  getCurrentAccount,
	"get_space_usage":      getSpaceUsage,
	"list_folder":          listFolder,
	"list_folder_continue": listFolderContinue,
	"get_metadata":         getMetadata,
	"search_files":         searchFiles,
	"read_file":            readFile,
	"list_shared_links":    listSharedLinks,
	// Write
	"write_file":         writeFile,
	"create_folder":      createFolder,
	"copy_file":          copyFile,
	"move_file":          moveFile,
	"delete_file":        deleteFile,
	"create_shared_link": createSharedLink,
	"list_revisions":     listRevisions,
}

// =============================================================================
// Read Handlers
// =============================================================================

func getCurrentAccount(ctx context.Context, params map[string]any) (string, error) {
	endpoint := dropboxAPIBase + "/users/get_current_account"
	// Dropbox requires a JSON body (even "null") for no-param endpoints
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), json.RawMessage("null"))
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getSpaceUsage(ctx context.Context, params map[string]any) (string, error) {
	endpoint := dropboxAPIBase + "/users/get_space_usage"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), json.RawMessage("null"))
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func listFolder(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)

	body := map[string]interface{}{
		"path": path,
	}
	if recursive, ok := params["recursive"].(bool); ok {
		body["recursive"] = recursive
	}
	if includeDeleted, ok := params["include_deleted"].(bool); ok {
		body["include_deleted"] = includeDeleted
	}
	if includeMediaInfo, ok := params["include_media_info"].(bool); ok {
		body["include_media_info"] = includeMediaInfo
	}
	if limit, ok := params["limit"].(float64); ok {
		body["limit"] = int(limit)
	}

	endpoint := dropboxAPIBase + "/files/list_folder"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func listFolderContinue(ctx context.Context, params map[string]any) (string, error) {
	cursor, _ := params["cursor"].(string)
	if cursor == "" {
		return "", fmt.Errorf("cursor is required")
	}

	body := map[string]interface{}{
		"cursor": cursor,
	}

	endpoint := dropboxAPIBase + "/files/list_folder/continue"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getMetadata(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	body := map[string]interface{}{
		"path": path,
	}
	if includeMediaInfo, ok := params["include_media_info"].(bool); ok {
		body["include_media_info"] = includeMediaInfo
	}
	if includeDeleted, ok := params["include_deleted"].(bool); ok {
		body["include_deleted"] = includeDeleted
	}

	endpoint := dropboxAPIBase + "/files/get_metadata"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func searchFiles(ctx context.Context, params map[string]any) (string, error) {
	query, _ := params["query"].(string)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	body := map[string]interface{}{
		"query": query,
	}

	options := map[string]interface{}{}
	if path, ok := params["path"].(string); ok && path != "" {
		options["path"] = path
	}
	if maxResults, ok := params["max_results"].(float64); ok {
		options["max_results"] = int(maxResults)
	}
	if categories, ok := params["file_categories"].([]interface{}); ok && len(categories) > 0 {
		fcs := make([]map[string]string, 0, len(categories))
		for _, c := range categories {
			if cs, ok := c.(string); ok {
				fcs = append(fcs, map[string]string{".tag": cs})
			}
		}
		options["file_categories"] = fcs
	}
	if len(options) > 0 {
		body["options"] = options
	}

	endpoint := dropboxAPIBase + "/files/search_v2"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func readFile(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	apiArg := map[string]string{
		"path": path,
	}
	return doContentDownload(ctx, "/files/download", apiArg)
}

func listSharedLinks(ctx context.Context, params map[string]any) (string, error) {
	body := map[string]interface{}{}
	if path, ok := params["path"].(string); ok && path != "" {
		body["path"] = path
	}
	if cursor, ok := params["cursor"].(string); ok && cursor != "" {
		body["cursor"] = cursor
	}

	endpoint := dropboxAPIBase + "/sharing/list_shared_links"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Write Handlers
// =============================================================================

func writeFile(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	content, _ := params["content"].(string)

	apiArg := map[string]interface{}{
		"path": path,
		"mode": "add",
	}
	if mode, ok := params["mode"].(string); ok && mode != "" {
		apiArg["mode"] = mode
	}
	if autorename, ok := params["autorename"].(bool); ok {
		apiArg["autorename"] = autorename
	}
	if mute, ok := params["mute"].(bool); ok {
		apiArg["mute"] = mute
	}

	return doContentUpload(ctx, "/files/upload", apiArg, content)
}

func createFolder(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	body := map[string]interface{}{
		"path": path,
	}
	if autorename, ok := params["autorename"].(bool); ok {
		body["autorename"] = autorename
	}

	endpoint := dropboxAPIBase + "/files/create_folder_v2"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func copyFile(ctx context.Context, params map[string]any) (string, error) {
	fromPath, _ := params["from_path"].(string)
	if fromPath == "" {
		return "", fmt.Errorf("from_path is required")
	}
	toPath, _ := params["to_path"].(string)
	if toPath == "" {
		return "", fmt.Errorf("to_path is required")
	}

	body := map[string]interface{}{
		"from_path": fromPath,
		"to_path":   toPath,
	}
	if autorename, ok := params["autorename"].(bool); ok {
		body["autorename"] = autorename
	}

	endpoint := dropboxAPIBase + "/files/copy_v2"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func moveFile(ctx context.Context, params map[string]any) (string, error) {
	fromPath, _ := params["from_path"].(string)
	if fromPath == "" {
		return "", fmt.Errorf("from_path is required")
	}
	toPath, _ := params["to_path"].(string)
	if toPath == "" {
		return "", fmt.Errorf("to_path is required")
	}

	body := map[string]interface{}{
		"from_path": fromPath,
		"to_path":   toPath,
	}
	if autorename, ok := params["autorename"].(bool); ok {
		body["autorename"] = autorename
	}

	endpoint := dropboxAPIBase + "/files/move_v2"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteFile(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	body := map[string]interface{}{
		"path": path,
	}

	endpoint := dropboxAPIBase + "/files/delete_v2"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createSharedLink(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	body := map[string]interface{}{
		"path": path,
	}
	if vis, ok := params["requested_visibility"].(string); ok && vis != "" {
		body["settings"] = map[string]interface{}{
			"requested_visibility": vis,
		}
	}

	endpoint := dropboxAPIBase + "/sharing/create_shared_link_with_settings"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func listRevisions(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	body := map[string]interface{}{
		"path": path,
	}
	if mode, ok := params["mode"].(string); ok && mode != "" {
		body["mode"] = mode
	}
	if limit, ok := params["limit"].(float64); ok {
		body["limit"] = int(limit)
	}

	endpoint := dropboxAPIBase + "/files/list_revisions"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}
