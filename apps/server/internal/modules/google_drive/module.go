package google_drive

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
	googleDriveAPIBase = "https://www.googleapis.com/drive/v3"
	googleDriveVersion = "v3"
	googleTokenURL     = "https://oauth2.googleapis.com/token"
	tokenRefreshBuffer = 5 * 60 // Refresh 5 minutes before expiry
)

var client = httpclient.New()

// GoogleDriveModule implements the Module interface for Google Drive API
type GoogleDriveModule struct{}

// New creates a new GoogleDriveModule instance
func New() *GoogleDriveModule {
	return &GoogleDriveModule{}
}

// Module descriptions in multiple languages
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Google Drive API - List, search, read, and manage files and folders",
	"ja-JP": "Google Drive API - ファイルとフォルダの一覧表示、検索、読み取り、管理",
}

// Name returns the module name
func (m *GoogleDriveModule) Name() string {
	return "google_drive"
}

// Descriptions returns the module descriptions in all languages
func (m *GoogleDriveModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description for a specific language
func (m *GoogleDriveModule) Description(lang string) string {
	return modules.GetLocalizedText(moduleDescriptions, lang)
}

// APIVersion returns the Google Drive API version
func (m *GoogleDriveModule) APIVersion() string {
	return googleDriveVersion
}

// Tools returns all available tools
func (m *GoogleDriveModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *GoogleDriveModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for Google Drive)
func (m *GoogleDriveModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *GoogleDriveModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *store.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[google_drive] No auth context")
		return nil
	}
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "google_drive")
	if err != nil {
		log.Printf("[google_drive] GetModuleToken error: %v", err)
		return nil
	}
	log.Printf("[google_drive] Got credentials: auth_type=%s, has_access_token=%v", credentials.AuthType, credentials.AccessToken != "")

	// Check if token needs refresh (OAuth2 only)
	if credentials.AuthType == store.AuthTypeOAuth2 && credentials.RefreshToken != "" {
		if needsRefresh(credentials) {
			log.Printf("[google_drive] Token expired or expiring soon, refreshing...")
			refreshed, err := refreshToken(ctx, authCtx.UserID, credentials)
			if err != nil {
				log.Printf("[google_drive] Token refresh failed: %v", err)
				// Return original credentials and let the API call fail
				return credentials
			}
			log.Printf("[google_drive] Token refreshed successfully")
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
	return now >= (int64(creds.ExpiresAt) - tokenRefreshBuffer)
}

// refreshToken exchanges the refresh token for a new access token
func refreshToken(ctx context.Context, userID string, creds *store.Credentials) (*store.Credentials, error) {
	// Get OAuth app credentials (client_id, client_secret) - shared with other Google modules
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
		ExpiresAt:    store.FlexibleTime(time.Now().Unix() + tokenResp.ExpiresIn),
	}

	// Save updated credentials to Vault
	err = store.GetTokenStore().UpdateModuleToken(ctx, userID, "google_drive", newCreds)
	if err != nil {
		log.Printf("[google_drive] Failed to save refreshed token: %v", err)
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
	// Files
	{
		ID:   "google_drive:list_files",
		Name: "list_files",
		Descriptions: modules.LocalizedText{
			"en-US": "List files and folders in Google Drive.",
			"ja-JP": "Google Drive内のファイルとフォルダを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"query":       {Type: "string", Description: "Search query (Google Drive query syntax, e.g., \"name contains 'report'\" or \"mimeType='application/pdf'\")"},
				"folder_id":   {Type: "string", Description: "Folder ID to list contents of. Use 'root' for the root folder."},
				"page_size":   {Type: "number", Description: "Maximum number of files to return (1-1000). Default: 100"},
				"page_token":  {Type: "string", Description: "Token for pagination"},
				"order_by":    {Type: "string", Description: "Sort order (e.g., 'name', 'modifiedTime desc', 'folder,name')"},
				"include_trashed": {Type: "boolean", Description: "Include trashed files. Default: false"},
			},
		},
	},
	{
		ID:   "google_drive:get_file",
		Name: "get_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Get metadata of a specific file or folder.",
			"ja-JP": "特定のファイルまたはフォルダのメタデータを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id": {Type: "string", Description: "File or folder ID"},
			},
			Required: []string{"file_id"},
		},
	},
	{
		ID:   "google_drive:search_files",
		Name: "search_files",
		Descriptions: modules.LocalizedText{
			"en-US": "Search for files in Google Drive by name or content.",
			"ja-JP": "名前またはコンテンツでGoogle Drive内のファイルを検索します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"name":      {Type: "string", Description: "Search by file name (partial match)"},
				"full_text": {Type: "string", Description: "Full-text search in file content"},
				"mime_type": {Type: "string", Description: "Filter by MIME type (e.g., 'application/pdf', 'application/vnd.google-apps.document')"},
				"page_size": {Type: "number", Description: "Maximum number of results (1-1000). Default: 100"},
			},
		},
	},
	{
		ID:   "google_drive:read_file",
		Name: "read_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Read the content of a file. For Google Docs/Sheets/Slides, exports as text/csv/plain text.",
			"ja-JP": "ファイルの内容を読み取ります。Google Docs/Sheets/Slidesの場合はテキスト/CSV/プレーンテキストとしてエクスポートします。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id": {Type: "string", Description: "File ID to read"},
			},
			Required: []string{"file_id"},
		},
	},
	// Folders
	{
		ID:   "google_drive:create_folder",
		Name: "create_folder",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new folder in Google Drive.",
			"ja-JP": "Google Driveに新しいフォルダを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"name":      {Type: "string", Description: "Folder name"},
				"parent_id": {Type: "string", Description: "Parent folder ID. Use 'root' for the root folder."},
			},
			Required: []string{"name"},
		},
	},
	// File operations
	{
		ID:   "google_drive:copy_file",
		Name: "copy_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a copy of a file.",
			"ja-JP": "ファイルのコピーを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id":   {Type: "string", Description: "File ID to copy"},
				"name":      {Type: "string", Description: "Name for the copy"},
				"parent_id": {Type: "string", Description: "Parent folder ID for the copy"},
			},
			Required: []string{"file_id"},
		},
	},
	{
		ID:   "google_drive:move_file",
		Name: "move_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Move a file to a different folder.",
			"ja-JP": "ファイルを別のフォルダに移動します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id":          {Type: "string", Description: "File ID to move"},
				"new_parent_id":    {Type: "string", Description: "New parent folder ID"},
				"remove_parent_id": {Type: "string", Description: "Current parent folder ID to remove from"},
			},
			Required: []string{"file_id", "new_parent_id"},
		},
	},
	{
		ID:   "google_drive:rename_file",
		Name: "rename_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Rename a file or folder.",
			"ja-JP": "ファイルまたはフォルダの名前を変更します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id":  {Type: "string", Description: "File or folder ID"},
				"new_name": {Type: "string", Description: "New name"},
			},
			Required: []string{"file_id", "new_name"},
		},
	},
	{
		ID:   "google_drive:delete_file",
		Name: "delete_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Move a file or folder to trash.",
			"ja-JP": "ファイルまたはフォルダをゴミ箱に移動します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id": {Type: "string", Description: "File or folder ID to delete"},
			},
			Required: []string{"file_id"},
		},
	},
	// About
	{
		ID:   "google_drive:get_about",
		Name: "get_about",
		Descriptions: modules.LocalizedText{
			"en-US": "Get information about the user's Drive (storage quota, user info).",
			"ja-JP": "ユーザーのDrive情報（ストレージ容量、ユーザー情報）を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	// Upload & Update
	{
		ID:   "google_drive:upload_file",
		Name: "upload_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Upload a new file to Google Drive (text content only).",
			"ja-JP": "Google Driveに新しいファイルをアップロードします（テキストコンテンツのみ）。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"name":      {Type: "string", Description: "File name"},
				"content":   {Type: "string", Description: "File content (text)"},
				"mime_type": {Type: "string", Description: "MIME type (e.g., 'text/plain', 'application/json'). Default: text/plain"},
				"parent_id": {Type: "string", Description: "Parent folder ID. Use 'root' for the root folder."},
			},
			Required: []string{"name", "content"},
		},
	},
	{
		ID:   "google_drive:update_file_content",
		Name: "update_file_content",
		Descriptions: modules.LocalizedText{
			"en-US": "Update the content of an existing file (text content only).",
			"ja-JP": "既存ファイルの内容を更新します（テキストコンテンツのみ）。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id": {Type: "string", Description: "File ID to update"},
				"content": {Type: "string", Description: "New file content (text)"},
			},
			Required: []string{"file_id", "content"},
		},
	},
	// Sharing & Permissions
	{
		ID:   "google_drive:list_permissions",
		Name: "list_permissions",
		Descriptions: modules.LocalizedText{
			"en-US": "List permissions (sharing settings) for a file or folder.",
			"ja-JP": "ファイルまたはフォルダの権限（共有設定）を一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id": {Type: "string", Description: "File or folder ID"},
			},
			Required: []string{"file_id"},
		},
	},
	{
		ID:   "google_drive:share_file",
		Name: "share_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Share a file or folder with a user, group, domain, or anyone.",
			"ja-JP": "ファイルまたはフォルダをユーザー、グループ、ドメイン、または全員と共有します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id":       {Type: "string", Description: "File or folder ID to share"},
				"type":          {Type: "string", Description: "Permission type: 'user', 'group', 'domain', or 'anyone'"},
				"role":          {Type: "string", Description: "Role: 'reader', 'commenter', 'writer', or 'owner'"},
				"email_address": {Type: "string", Description: "Email address (required for type='user' or 'group')"},
				"domain":        {Type: "string", Description: "Domain (required for type='domain')"},
				"send_notification": {Type: "boolean", Description: "Send notification email. Default: true"},
			},
			Required: []string{"file_id", "type", "role"},
		},
	},
	{
		ID:   "google_drive:delete_permission",
		Name: "delete_permission",
		Descriptions: modules.LocalizedText{
			"en-US": "Remove a permission (unshare) from a file or folder.",
			"ja-JP": "ファイルまたはフォルダから権限（共有）を削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id":       {Type: "string", Description: "File or folder ID"},
				"permission_id": {Type: "string", Description: "Permission ID to delete"},
			},
			Required: []string{"file_id", "permission_id"},
		},
	},
	// Comments
	{
		ID:   "google_drive:list_comments",
		Name: "list_comments",
		Descriptions: modules.LocalizedText{
			"en-US": "List comments on a file.",
			"ja-JP": "ファイルのコメントを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id":    {Type: "string", Description: "File ID"},
				"page_size":  {Type: "number", Description: "Maximum number of comments (1-100). Default: 20"},
				"page_token": {Type: "string", Description: "Token for pagination"},
			},
			Required: []string{"file_id"},
		},
	},
	{
		ID:   "google_drive:create_comment",
		Name: "create_comment",
		Descriptions: modules.LocalizedText{
			"en-US": "Add a comment to a file.",
			"ja-JP": "ファイルにコメントを追加します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id": {Type: "string", Description: "File ID"},
				"content": {Type: "string", Description: "Comment content"},
			},
			Required: []string{"file_id", "content"},
		},
	},
	// Revisions
	{
		ID:   "google_drive:list_revisions",
		Name: "list_revisions",
		Descriptions: modules.LocalizedText{
			"en-US": "List revisions (version history) of a file.",
			"ja-JP": "ファイルのリビジョン（バージョン履歴）を一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id":    {Type: "string", Description: "File ID"},
				"page_size":  {Type: "number", Description: "Maximum number of revisions (1-1000). Default: 100"},
				"page_token": {Type: "string", Description: "Token for pagination"},
			},
			Required: []string{"file_id"},
		},
	},
	// Trash operations
	{
		ID:   "google_drive:restore_file",
		Name: "restore_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Restore a file from trash.",
			"ja-JP": "ゴミ箱からファイルを復元します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id": {Type: "string", Description: "File ID to restore"},
			},
			Required: []string{"file_id"},
		},
	},
	{
		ID:   "google_drive:empty_trash",
		Name: "empty_trash",
		Descriptions: modules.LocalizedText{
			"en-US": "Permanently delete all files in trash.",
			"ja-JP": "ゴミ箱内のすべてのファイルを完全に削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	// Shared Drives
	{
		ID:   "google_drive:list_shared_drives",
		Name: "list_shared_drives",
		Descriptions: modules.LocalizedText{
			"en-US": "List shared drives the user has access to.",
			"ja-JP": "ユーザーがアクセスできる共有ドライブを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"page_size":  {Type: "number", Description: "Maximum number of shared drives (1-100). Default: 100"},
				"page_token": {Type: "string", Description: "Token for pagination"},
			},
		},
	},
	// Export
	{
		ID:   "google_drive:export_file",
		Name: "export_file",
		Descriptions: modules.LocalizedText{
			"en-US": "Export a Google Workspace file (Docs, Sheets, Slides) to a specific format.",
			"ja-JP": "Google Workspaceファイル（Docs、Sheets、Slides）を特定の形式でエクスポートします。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"file_id":   {Type: "string", Description: "File ID to export"},
				"mime_type": {Type: "string", Description: "Export format. Docs: 'application/pdf', 'text/plain', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'. Sheets: 'application/pdf', 'text/csv', 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'. Slides: 'application/pdf', 'application/vnd.openxmlformats-officedocument.presentationml.presentation'."},
			},
			Required: []string{"file_id", "mime_type"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"list_files":          listFiles,
	"get_file":            getFile,
	"search_files":        searchFiles,
	"read_file":           readFile,
	"create_folder":       createFolder,
	"copy_file":           copyFile,
	"move_file":           moveFile,
	"rename_file":         renameFile,
	"delete_file":         deleteFile,
	"get_about":           getAbout,
	"upload_file":         uploadFile,
	"update_file_content": updateFileContent,
	"list_permissions":    listPermissions,
	"share_file":          shareFile,
	"delete_permission":   deletePermission,
	"list_comments":       listComments,
	"create_comment":      createComment,
	"list_revisions":      listRevisions,
	"restore_file":        restoreFile,
	"empty_trash":         emptyTrash,
	"list_shared_drives":  listSharedDrives,
	"export_file":         exportFile,
}

// Standard file fields to request
const fileFields = "id,name,mimeType,size,createdTime,modifiedTime,parents,webViewLink,iconLink,trashed"

// =============================================================================
// Files
// =============================================================================

func listFiles(ctx context.Context, params map[string]any) (string, error) {
	query := url.Values{}

	// Build query string
	var queryParts []string

	if q, ok := params["query"].(string); ok && q != "" {
		queryParts = append(queryParts, q)
	}

	if folderID, ok := params["folder_id"].(string); ok && folderID != "" {
		queryParts = append(queryParts, fmt.Sprintf("'%s' in parents", folderID))
	}

	// Include trashed
	includeTrashed := false
	if it, ok := params["include_trashed"].(bool); ok {
		includeTrashed = it
	}
	if !includeTrashed {
		queryParts = append(queryParts, "trashed=false")
	}

	if len(queryParts) > 0 {
		query.Set("q", strings.Join(queryParts, " and "))
	}

	// Page size
	pageSize := 100
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		pageSize = int(ps)
		if pageSize > 1000 {
			pageSize = 1000
		}
	}
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))

	// Page token
	if pt, ok := params["page_token"].(string); ok && pt != "" {
		query.Set("pageToken", pt)
	}

	// Order by
	if ob, ok := params["order_by"].(string); ok && ob != "" {
		query.Set("orderBy", ob)
	}

	// Fields
	query.Set("fields", fmt.Sprintf("nextPageToken,files(%s)", fileFields))

	endpoint := fmt.Sprintf("%s/files?%s", googleDriveAPIBase, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	query := url.Values{}
	query.Set("fields", fileFields)

	endpoint := fmt.Sprintf("%s/files/%s?%s", googleDriveAPIBase, url.PathEscape(fileID), query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func searchFiles(ctx context.Context, params map[string]any) (string, error) {
	var queryParts []string

	// Name search
	if name, ok := params["name"].(string); ok && name != "" {
		queryParts = append(queryParts, fmt.Sprintf("name contains '%s'", name))
	}

	// Full-text search
	if fullText, ok := params["full_text"].(string); ok && fullText != "" {
		queryParts = append(queryParts, fmt.Sprintf("fullText contains '%s'", fullText))
	}

	// MIME type filter
	if mimeType, ok := params["mime_type"].(string); ok && mimeType != "" {
		queryParts = append(queryParts, fmt.Sprintf("mimeType='%s'", mimeType))
	}

	// Exclude trashed
	queryParts = append(queryParts, "trashed=false")

	query := url.Values{}
	if len(queryParts) > 0 {
		query.Set("q", strings.Join(queryParts, " and "))
	}

	// Page size
	pageSize := 100
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		pageSize = int(ps)
		if pageSize > 1000 {
			pageSize = 1000
		}
	}
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))
	query.Set("fields", fmt.Sprintf("files(%s)", fileFields))

	endpoint := fmt.Sprintf("%s/files?%s", googleDriveAPIBase, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func readFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	// First, get file metadata to determine MIME type
	metaEndpoint := fmt.Sprintf("%s/files/%s?fields=mimeType,name", googleDriveAPIBase, url.PathEscape(fileID))
	metaBody, err := client.DoJSON("GET", metaEndpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	var meta struct {
		MimeType string `json:"mimeType"`
		Name     string `json:"name"`
	}
	if err := json.Unmarshal(metaBody, &meta); err != nil {
		return "", fmt.Errorf("failed to parse file metadata: %w", err)
	}

	// Determine export format for Google Workspace files
	var endpoint string
	switch meta.MimeType {
	case "application/vnd.google-apps.document":
		// Google Docs -> export as plain text
		endpoint = fmt.Sprintf("%s/files/%s/export?mimeType=text/plain", googleDriveAPIBase, url.PathEscape(fileID))
	case "application/vnd.google-apps.spreadsheet":
		// Google Sheets -> export as CSV
		endpoint = fmt.Sprintf("%s/files/%s/export?mimeType=text/csv", googleDriveAPIBase, url.PathEscape(fileID))
	case "application/vnd.google-apps.presentation":
		// Google Slides -> export as plain text
		endpoint = fmt.Sprintf("%s/files/%s/export?mimeType=text/plain", googleDriveAPIBase, url.PathEscape(fileID))
	default:
		// Binary/text files -> download directly
		endpoint = fmt.Sprintf("%s/files/%s?alt=media", googleDriveAPIBase, url.PathEscape(fileID))
	}

	// Make request
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	hdrs := headers(ctx)
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to read file: status %d", resp.StatusCode)
	}

	// Read content (limit to 1MB for safety)
	const maxSize = 1 * 1024 * 1024 // 1MB
	content := make([]byte, maxSize)
	n, err := resp.Body.Read(content)
	if err != nil && err.Error() != "EOF" {
		// Ignore EOF, it's expected
		if n == 0 {
			return "", fmt.Errorf("failed to read content: %w", err)
		}
	}

	result := map[string]interface{}{
		"name":      meta.Name,
		"mime_type": meta.MimeType,
		"content":   string(content[:n]),
		"truncated": n >= maxSize,
	}
	resultBytes, _ := json.Marshal(result)
	return httpclient.PrettyJSON(resultBytes), nil
}

// =============================================================================
// Folders
// =============================================================================

func createFolder(ctx context.Context, params map[string]any) (string, error) {
	name, _ := params["name"].(string)

	body := map[string]interface{}{
		"name":     name,
		"mimeType": "application/vnd.google-apps.folder",
	}

	if parentID, ok := params["parent_id"].(string); ok && parentID != "" {
		body["parents"] = []string{parentID}
	}

	endpoint := googleDriveAPIBase + "/files"
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// File Operations
// =============================================================================

func copyFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	body := map[string]interface{}{}

	if name, ok := params["name"].(string); ok && name != "" {
		body["name"] = name
	}

	if parentID, ok := params["parent_id"].(string); ok && parentID != "" {
		body["parents"] = []string{parentID}
	}

	endpoint := fmt.Sprintf("%s/files/%s/copy", googleDriveAPIBase, url.PathEscape(fileID))
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func moveFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)
	newParentID, _ := params["new_parent_id"].(string)

	query := url.Values{}
	query.Set("addParents", newParentID)

	if removeParentID, ok := params["remove_parent_id"].(string); ok && removeParentID != "" {
		query.Set("removeParents", removeParentID)
	} else {
		// If remove_parent_id not specified, we need to get current parents
		metaEndpoint := fmt.Sprintf("%s/files/%s?fields=parents", googleDriveAPIBase, url.PathEscape(fileID))
		metaBody, err := client.DoJSON("GET", metaEndpoint, headers(ctx), nil)
		if err != nil {
			return "", err
		}
		var meta struct {
			Parents []string `json:"parents"`
		}
		if err := json.Unmarshal(metaBody, &meta); err == nil && len(meta.Parents) > 0 {
			query.Set("removeParents", strings.Join(meta.Parents, ","))
		}
	}

	query.Set("fields", fileFields)

	endpoint := fmt.Sprintf("%s/files/%s?%s", googleDriveAPIBase, url.PathEscape(fileID), query.Encode())
	respBody, err := client.DoJSON("PATCH", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func renameFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)
	newName, _ := params["new_name"].(string)

	body := map[string]interface{}{
		"name": newName,
	}

	query := url.Values{}
	query.Set("fields", fileFields)

	endpoint := fmt.Sprintf("%s/files/%s?%s", googleDriveAPIBase, url.PathEscape(fileID), query.Encode())
	respBody, err := client.DoJSON("PATCH", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	// Move to trash (soft delete)
	body := map[string]interface{}{
		"trashed": true,
	}

	query := url.Values{}
	query.Set("fields", fileFields)

	endpoint := fmt.Sprintf("%s/files/%s?%s", googleDriveAPIBase, url.PathEscape(fileID), query.Encode())
	respBody, err := client.DoJSON("PATCH", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// About
// =============================================================================

func getAbout(ctx context.Context, params map[string]any) (string, error) {
	query := url.Values{}
	query.Set("fields", "user,storageQuota")

	endpoint := fmt.Sprintf("%s/about?%s", googleDriveAPIBase, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Upload & Update
// =============================================================================

const googleDriveUploadBase = "https://www.googleapis.com/upload/drive/v3"

func uploadFile(ctx context.Context, params map[string]any) (string, error) {
	name, _ := params["name"].(string)
	content, _ := params["content"].(string)
	mimeType := "text/plain"
	if mt, ok := params["mime_type"].(string); ok && mt != "" {
		mimeType = mt
	}

	// Build metadata
	metadata := map[string]interface{}{
		"name":     name,
		"mimeType": mimeType,
	}
	if parentID, ok := params["parent_id"].(string); ok && parentID != "" {
		metadata["parents"] = []string{parentID}
	}

	// Use multipart upload for simplicity
	boundary := "========multipart_boundary========"

	// Build multipart body
	var body strings.Builder
	body.WriteString("--")
	body.WriteString(boundary)
	body.WriteString("\r\nContent-Type: application/json; charset=UTF-8\r\n\r\n")
	metaJSON, _ := json.Marshal(metadata)
	body.Write(metaJSON)
	body.WriteString("\r\n--")
	body.WriteString(boundary)
	body.WriteString("\r\nContent-Type: ")
	body.WriteString(mimeType)
	body.WriteString("\r\n\r\n")
	body.WriteString(content)
	body.WriteString("\r\n--")
	body.WriteString(boundary)
	body.WriteString("--")

	endpoint := fmt.Sprintf("%s/files?uploadType=multipart&fields=%s", googleDriveUploadBase, fileFields)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(body.String()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	hdrs := headers(ctx)
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "multipart/related; boundary="+boundary)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to upload file: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	resultBytes, _ := json.Marshal(result)
	return httpclient.PrettyJSON(resultBytes), nil
}

func updateFileContent(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)
	content, _ := params["content"].(string)

	// Get current file metadata to preserve mime type
	metaEndpoint := fmt.Sprintf("%s/files/%s?fields=mimeType", googleDriveAPIBase, url.PathEscape(fileID))
	metaBody, err := client.DoJSON("GET", metaEndpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	var meta struct {
		MimeType string `json:"mimeType"`
	}
	json.Unmarshal(metaBody, &meta)
	mimeType := meta.MimeType
	if mimeType == "" {
		mimeType = "text/plain"
	}

	endpoint := fmt.Sprintf("%s/files/%s?uploadType=media&fields=%s", googleDriveUploadBase, url.PathEscape(fileID), fileFields)

	req, err := http.NewRequestWithContext(ctx, "PATCH", endpoint, strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	hdrs := headers(ctx)
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", mimeType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to update file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to update file: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	resultBytes, _ := json.Marshal(result)
	return httpclient.PrettyJSON(resultBytes), nil
}

// =============================================================================
// Permissions
// =============================================================================

func listPermissions(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	query := url.Values{}
	query.Set("fields", "permissions(id,type,role,emailAddress,domain,displayName)")

	endpoint := fmt.Sprintf("%s/files/%s/permissions?%s", googleDriveAPIBase, url.PathEscape(fileID), query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func shareFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)
	permType, _ := params["type"].(string)
	role, _ := params["role"].(string)

	body := map[string]interface{}{
		"type": permType,
		"role": role,
	}

	if email, ok := params["email_address"].(string); ok && email != "" {
		body["emailAddress"] = email
	}
	if domain, ok := params["domain"].(string); ok && domain != "" {
		body["domain"] = domain
	}

	query := url.Values{}
	query.Set("fields", "id,type,role,emailAddress,domain,displayName")

	// Send notification by default
	sendNotification := true
	if sn, ok := params["send_notification"].(bool); ok {
		sendNotification = sn
	}
	if !sendNotification {
		query.Set("sendNotificationEmail", "false")
	}

	endpoint := fmt.Sprintf("%s/files/%s/permissions?%s", googleDriveAPIBase, url.PathEscape(fileID), query.Encode())
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deletePermission(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)
	permissionID, _ := params["permission_id"].(string)

	endpoint := fmt.Sprintf("%s/files/%s/permissions/%s", googleDriveAPIBase, url.PathEscape(fileID), url.PathEscape(permissionID))

	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	hdrs := headers(ctx)
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to delete permission: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to delete permission: status %d", resp.StatusCode)
	}

	result := map[string]interface{}{
		"success": true,
		"message": "Permission deleted successfully",
	}
	resultBytes, _ := json.Marshal(result)
	return httpclient.PrettyJSON(resultBytes), nil
}

// =============================================================================
// Comments
// =============================================================================

func listComments(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	query := url.Values{}
	query.Set("fields", "comments(id,content,author,createdTime,modifiedTime,resolved)")

	pageSize := 20
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		pageSize = int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
	}
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))

	if pt, ok := params["page_token"].(string); ok && pt != "" {
		query.Set("pageToken", pt)
	}

	endpoint := fmt.Sprintf("%s/files/%s/comments?%s", googleDriveAPIBase, url.PathEscape(fileID), query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createComment(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)
	content, _ := params["content"].(string)

	body := map[string]interface{}{
		"content": content,
	}

	query := url.Values{}
	query.Set("fields", "id,content,author,createdTime")

	endpoint := fmt.Sprintf("%s/files/%s/comments?%s", googleDriveAPIBase, url.PathEscape(fileID), query.Encode())
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Revisions
// =============================================================================

func listRevisions(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	query := url.Values{}
	query.Set("fields", "revisions(id,mimeType,modifiedTime,keepForever,size)")

	pageSize := 100
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		pageSize = int(ps)
		if pageSize > 1000 {
			pageSize = 1000
		}
	}
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))

	if pt, ok := params["page_token"].(string); ok && pt != "" {
		query.Set("pageToken", pt)
	}

	endpoint := fmt.Sprintf("%s/files/%s/revisions?%s", googleDriveAPIBase, url.PathEscape(fileID), query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Trash Operations
// =============================================================================

func restoreFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	body := map[string]interface{}{
		"trashed": false,
	}

	query := url.Values{}
	query.Set("fields", fileFields)

	endpoint := fmt.Sprintf("%s/files/%s?%s", googleDriveAPIBase, url.PathEscape(fileID), query.Encode())
	respBody, err := client.DoJSON("PATCH", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func emptyTrash(ctx context.Context, params map[string]any) (string, error) {
	endpoint := fmt.Sprintf("%s/files/trash", googleDriveAPIBase)

	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	hdrs := headers(ctx)
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to empty trash: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to empty trash: status %d", resp.StatusCode)
	}

	result := map[string]interface{}{
		"success": true,
		"message": "Trash emptied successfully",
	}
	resultBytes, _ := json.Marshal(result)
	return httpclient.PrettyJSON(resultBytes), nil
}

// =============================================================================
// Shared Drives
// =============================================================================

func listSharedDrives(ctx context.Context, params map[string]any) (string, error) {
	query := url.Values{}

	pageSize := 100
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		pageSize = int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
	}
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))

	if pt, ok := params["page_token"].(string); ok && pt != "" {
		query.Set("pageToken", pt)
	}

	endpoint := fmt.Sprintf("%s/drives?%s", googleDriveAPIBase, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Export
// =============================================================================

func exportFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)
	mimeType, _ := params["mime_type"].(string)

	endpoint := fmt.Sprintf("%s/files/%s/export?mimeType=%s", googleDriveAPIBase, url.PathEscape(fileID), url.QueryEscape(mimeType))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	hdrs := headers(ctx)
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to export file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to export file: status %d", resp.StatusCode)
	}

	// Read content (limit to 1MB for safety)
	const maxSize = 1 * 1024 * 1024 // 1MB
	content := make([]byte, maxSize)
	n, err := resp.Body.Read(content)
	if err != nil && err.Error() != "EOF" {
		if n == 0 {
			return "", fmt.Errorf("failed to read content: %w", err)
		}
	}

	result := map[string]interface{}{
		"mime_type": mimeType,
		"content":   string(content[:n]),
		"truncated": n >= maxSize,
	}
	resultBytes, _ := json.Marshal(result)
	return httpclient.PrettyJSON(resultBytes), nil
}
