package google_drive

import (
	"context"
	"fmt"
	"log"
	"strings"

	"mcpist/server/internal/broker"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/pkg/googledriveapi"
	gen "mcpist/server/pkg/googledriveapi/gen"
)

const (
	googleDriveVersion = "v3"
)

var toJSON = modules.ToJSON

// Standard file fields to request via ogen fields param
const ogenFileFields = "id,name,mimeType,size,createdTime,modifiedTime,parents,webViewLink,iconLink,trashed"

// GoogleDriveModule implements the Module interface for Google Drive API
type GoogleDriveModule struct{}

func New() *GoogleDriveModule { return &GoogleDriveModule{} }

var moduleDescriptions = modules.LocalizedText{
	"en-US": "Google Drive API - List, search, read, and manage files and folders",
	"ja-JP": "Google Drive API - ファイルとフォルダの一覧表示、検索、読み取り、管理",
}

func (m *GoogleDriveModule) Name() string                        { return "google_drive" }
func (m *GoogleDriveModule) Descriptions() modules.LocalizedText { return moduleDescriptions }
func (m *GoogleDriveModule) Description() string {
	return moduleDescriptions["en-US"]
}
func (m *GoogleDriveModule) APIVersion() string            { return googleDriveVersion }
func (m *GoogleDriveModule) Tools() []modules.Tool         { return toolDefinitions }
func (m *GoogleDriveModule) Resources() []modules.Resource { return nil }
func (m *GoogleDriveModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

func (m *GoogleDriveModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format.
func (m *GoogleDriveModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// =============================================================================
// Token and Client
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[google_drive] No auth context")
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "google_drive")
	if err != nil {
		log.Printf("[google_drive] GetModuleToken error: %v", err)
		return nil
	}
	return credentials
}

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return googledriveapi.NewClient(creds.AccessToken)
}

func getAccessToken(ctx context.Context) (string, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return "", fmt.Errorf("no credentials available")
	}
	return creds.AccessToken, nil
}

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
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
				"query":           {Type: "string", Description: "Search query (Google Drive query syntax, e.g., \"name contains 'report'\" or \"mimeType='application/pdf'\")"},
				"folder_id":       {Type: "string", Description: "Folder ID to list contents of. Use 'root' for the root folder."},
				"page_size":       {Type: "number", Description: "Maximum number of files to return (1-1000). Default: 100"},
				"page_token":      {Type: "string", Description: "Token for pagination"},
				"order_by":        {Type: "string", Description: "Sort order (e.g., 'name', 'modifiedTime desc', 'folder,name')"},
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
				"file_id":           {Type: "string", Description: "File or folder ID to share"},
				"type":              {Type: "string", Description: "Permission type: 'user', 'group', 'domain', or 'anyone'"},
				"role":              {Type: "string", Description: "Role: 'reader', 'commenter', 'writer', or 'owner'"},
				"email_address":     {Type: "string", Description: "Email address (required for type='user' or 'group')"},
				"domain":            {Type: "string", Description: "Domain (required for type='domain')"},
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

// =============================================================================
// Files (ogen)
// =============================================================================

func listFiles(ctx context.Context, params map[string]any) (string, error) {
	var queryParts []string
	if q, ok := params["query"].(string); ok && q != "" {
		queryParts = append(queryParts, q)
	}
	if folderID, ok := params["folder_id"].(string); ok && folderID != "" {
		queryParts = append(queryParts, fmt.Sprintf("'%s' in parents", folderID))
	}
	includeTrashed, _ := params["include_trashed"].(bool)
	if !includeTrashed {
		queryParts = append(queryParts, "trashed=false")
	}

	p := gen.ListFilesParams{
		PageSize: gen.NewOptInt(100),
		Fields:   gen.NewOptString(fmt.Sprintf("nextPageToken,files(%s)", ogenFileFields)),
	}
	if len(queryParts) > 0 {
		p.Q = gen.NewOptString(strings.Join(queryParts, " and "))
	}
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		size := int(ps)
		if size > 1000 {
			size = 1000
		}
		p.PageSize = gen.NewOptInt(size)
	}
	if pt, ok := params["page_token"].(string); ok && pt != "" {
		p.PageToken = gen.NewOptString(pt)
	}
	if ob, ok := params["order_by"].(string); ok && ob != "" {
		p.OrderBy = gen.NewOptString(ob)
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListFiles(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetFile(ctx, gen.GetFileParams{
		FileId: fileID,
		Fields: gen.NewOptString(ogenFileFields),
	})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func searchFiles(ctx context.Context, params map[string]any) (string, error) {
	var queryParts []string
	if name, ok := params["name"].(string); ok && name != "" {
		queryParts = append(queryParts, fmt.Sprintf("name contains '%s'", name))
	}
	if fullText, ok := params["full_text"].(string); ok && fullText != "" {
		queryParts = append(queryParts, fmt.Sprintf("fullText contains '%s'", fullText))
	}
	if mimeType, ok := params["mime_type"].(string); ok && mimeType != "" {
		queryParts = append(queryParts, fmt.Sprintf("mimeType='%s'", mimeType))
	}
	queryParts = append(queryParts, "trashed=false")

	p := gen.ListFilesParams{
		PageSize: gen.NewOptInt(100),
		Fields:   gen.NewOptString(fmt.Sprintf("files(%s)", ogenFileFields)),
	}
	if len(queryParts) > 0 {
		p.Q = gen.NewOptString(strings.Join(queryParts, " and "))
	}
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		size := int(ps)
		if size > 1000 {
			size = 1000
		}
		p.PageSize = gen.NewOptInt(size)
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListFiles(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Folders (ogen)
// =============================================================================

func createFolder(ctx context.Context, params map[string]any) (string, error) {
	name, _ := params["name"].(string)

	meta := &gen.FileMetadata{
		Name:     gen.NewOptNilString(name),
		MimeType: gen.NewOptNilString("application/vnd.google-apps.folder"),
	}
	if parentID, ok := params["parent_id"].(string); ok && parentID != "" {
		meta.Parents = gen.NewOptNilStringArray([]string{parentID})
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.CreateFile(ctx, meta)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// File Operations (ogen)
// =============================================================================

func copyFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	req := &gen.CopyRequest{}
	if name, ok := params["name"].(string); ok && name != "" {
		req.Name = gen.NewOptNilString(name)
	}
	if parentID, ok := params["parent_id"].(string); ok && parentID != "" {
		req.Parents = gen.NewOptNilStringArray([]string{parentID})
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.CopyFile(ctx, req, gen.CopyFileParams{FileId: fileID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func moveFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)
	newParentID, _ := params["new_parent_id"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	p := gen.UpdateFileParams{
		FileId:     fileID,
		AddParents: gen.NewOptString(newParentID),
		Fields:     gen.NewOptString(ogenFileFields),
	}

	if removeParentID, ok := params["remove_parent_id"].(string); ok && removeParentID != "" {
		p.RemoveParents = gen.NewOptString(removeParentID)
	} else {
		existing, err := c.GetFile(ctx, gen.GetFileParams{FileId: fileID, Fields: gen.NewOptString("parents")})
		if err == nil && existing.Parents.Set {
			p.RemoveParents = gen.NewOptString(strings.Join(existing.Parents.Value, ","))
		}
	}

	res, err := c.UpdateFile(ctx, &gen.FileMetadata{}, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func renameFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)
	newName, _ := params["new_name"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.UpdateFile(ctx,
		&gen.FileMetadata{Name: gen.NewOptNilString(newName)},
		gen.UpdateFileParams{FileId: fileID, Fields: gen.NewOptString(ogenFileFields)},
	)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func deleteFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.UpdateFile(ctx,
		&gen.FileMetadata{Trashed: gen.NewOptNilBool(true)},
		gen.UpdateFileParams{FileId: fileID, Fields: gen.NewOptString(ogenFileFields)},
	)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func restoreFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.UpdateFile(ctx,
		&gen.FileMetadata{Trashed: gen.NewOptNilBool(false)},
		gen.UpdateFileParams{FileId: fileID, Fields: gen.NewOptString(ogenFileFields)},
	)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// About (ogen)
// =============================================================================

func getAbout(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetAbout(ctx, gen.GetAboutParams{Fields: gen.NewOptString("user,storageQuota")})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Upload & Update (manual HTTP — see httpclient.go)
// =============================================================================

func uploadFile(ctx context.Context, params map[string]any) (string, error) {
	token, err := getAccessToken(ctx)
	if err != nil {
		return "", err
	}
	name, _ := params["name"].(string)
	content, _ := params["content"].(string)
	mimeType, _ := params["mime_type"].(string)
	parentID, _ := params["parent_id"].(string)
	return doUploadFile(ctx, token, name, content, mimeType, parentID)
}

func updateFileContent(ctx context.Context, params map[string]any) (string, error) {
	token, err := getAccessToken(ctx)
	if err != nil {
		return "", err
	}
	fileID, _ := params["file_id"].(string)
	content, _ := params["content"].(string)
	return doUpdateFileContent(ctx, token, fileID, content)
}

// =============================================================================
// Permissions (ogen)
// =============================================================================

func listPermissions(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListPermissions(ctx, gen.ListPermissionsParams{
		FileId: fileID,
		Fields: gen.NewOptString("permissions(id,type,role,emailAddress,domain,displayName)"),
	})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func shareFile(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)
	permType, _ := params["type"].(string)
	role, _ := params["role"].(string)

	req := &gen.PermissionRequest{
		Type: gen.NewOptString(permType),
		Role: gen.NewOptString(role),
	}
	if email, ok := params["email_address"].(string); ok && email != "" {
		req.EmailAddress = gen.NewOptNilString(email)
	}
	if domain, ok := params["domain"].(string); ok && domain != "" {
		req.Domain = gen.NewOptNilString(domain)
	}

	p := gen.CreatePermissionParams{
		FileId: fileID,
		Fields: gen.NewOptString("id,type,role,emailAddress,domain,displayName"),
	}

	sendNotification := true
	if sn, ok := params["send_notification"].(bool); ok {
		sendNotification = sn
	}
	if !sendNotification {
		p.SendNotificationEmail = gen.NewOptBool(false)
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.CreatePermission(ctx, req, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func deletePermission(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)
	permissionID, _ := params["permission_id"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	err = c.DeletePermission(ctx, gen.DeletePermissionParams{FileId: fileID, PermissionId: permissionID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Permission deleted successfully"}`, nil
}

// =============================================================================
// Comments (ogen)
// =============================================================================

func listComments(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	p := gen.ListCommentsParams{
		FileId:   fileID,
		PageSize: gen.NewOptInt(20),
		Fields:   gen.NewOptString("comments(id,content,author,createdTime,modifiedTime,resolved)"),
	}
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		size := int(ps)
		if size > 100 {
			size = 100
		}
		p.PageSize = gen.NewOptInt(size)
	}
	if pt, ok := params["page_token"].(string); ok && pt != "" {
		p.PageToken = gen.NewOptString(pt)
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListComments(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func createComment(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)
	content, _ := params["content"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.CreateComment(ctx,
		&gen.CommentRequest{Content: gen.NewOptString(content)},
		gen.CreateCommentParams{
			FileId: fileID,
			Fields: gen.NewOptString("id,content,author,createdTime"),
		},
	)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Revisions (ogen)
// =============================================================================

func listRevisions(ctx context.Context, params map[string]any) (string, error) {
	fileID, _ := params["file_id"].(string)

	p := gen.ListRevisionsParams{
		FileId:   fileID,
		PageSize: gen.NewOptInt(100),
		Fields:   gen.NewOptString("revisions(id,mimeType,modifiedTime,keepForever,size)"),
	}
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		size := int(ps)
		if size > 1000 {
			size = 1000
		}
		p.PageSize = gen.NewOptInt(size)
	}
	if pt, ok := params["page_token"].(string); ok && pt != "" {
		p.PageToken = gen.NewOptString(pt)
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListRevisions(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Trash (manual HTTP — see httpclient.go)
// =============================================================================

func emptyTrash(ctx context.Context, params map[string]any) (string, error) {
	token, err := getAccessToken(ctx)
	if err != nil {
		return "", err
	}
	return doEmptyTrash(ctx, token)
}

// =============================================================================
// Shared Drives (ogen)
// =============================================================================

func listSharedDrives(ctx context.Context, params map[string]any) (string, error) {
	p := gen.ListDrivesParams{
		PageSize: gen.NewOptInt(100),
	}
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		size := int(ps)
		if size > 100 {
			size = 100
		}
		p.PageSize = gen.NewOptInt(size)
	}
	if pt, ok := params["page_token"].(string); ok && pt != "" {
		p.PageToken = gen.NewOptString(pt)
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListDrives(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Read & Export (manual HTTP — see httpclient.go)
// =============================================================================

func readFile(ctx context.Context, params map[string]any) (string, error) {
	token, err := getAccessToken(ctx)
	if err != nil {
		return "", err
	}
	fileID, _ := params["file_id"].(string)
	return doReadFile(ctx, token, fileID)
}

func exportFile(ctx context.Context, params map[string]any) (string, error) {
	token, err := getAccessToken(ctx)
	if err != nil {
		return "", err
	}
	fileID, _ := params["file_id"].(string)
	mimeType, _ := params["mime_type"].(string)
	return doExportFile(ctx, token, fileID, mimeType)
}
