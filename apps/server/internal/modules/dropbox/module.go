package dropbox

import (
	"context"
	"encoding/json"
	"fmt"

	"mcpist/server/internal/broker"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
)

const (
	dropboxAPIBase     = "https://api.dropboxapi.com/2"
	dropboxContentBase = "https://content.dropboxapi.com/2"
	dropboxVersion     = "v2"
)

var toJSON = modules.ToJSON

// DropboxModule implements the Module interface for Dropbox API
type DropboxModule struct{}

func New() *DropboxModule { return &DropboxModule{} }

var moduleDescriptions = modules.LocalizedText{
	"en-US": "Dropbox API - File and folder operations (list, search, upload, download, share)",
	"ja-JP": "Dropbox API - ファイルとフォルダの操作（一覧、検索、アップロード、ダウンロード、共有）",
}

func (m *DropboxModule) Name() string                        { return "dropbox" }
func (m *DropboxModule) Descriptions() modules.LocalizedText { return moduleDescriptions }
func (m *DropboxModule) Description() string {
	return moduleDescriptions["en-US"]
}
func (m *DropboxModule) APIVersion() string { return dropboxVersion }
func (m *DropboxModule) Tools() []modules.Tool {
	return toolDefinitions
}

func (m *DropboxModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format.
func (m *DropboxModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

func (m *DropboxModule) Resources() []modules.Resource { return nil }
func (m *DropboxModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// Token Management
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "dropbox")
	if err != nil {
		return nil
	}
	return credentials
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
	return doPost(ctx, "/users/get_current_account", json.RawMessage("null"))
}

func getSpaceUsage(ctx context.Context, params map[string]any) (string, error) {
	return doPost(ctx, "/users/get_space_usage", json.RawMessage("null"))
}

func listFolder(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)

	body := map[string]any{"path": path}
	if v, ok := params["recursive"].(bool); ok {
		body["recursive"] = v
	}
	if v, ok := params["include_deleted"].(bool); ok {
		body["include_deleted"] = v
	}
	if v, ok := params["include_media_info"].(bool); ok {
		body["include_media_info"] = v
	}
	if v, ok := params["limit"].(float64); ok {
		body["limit"] = int(v)
	}

	return doPost(ctx, "/files/list_folder", body)
}

func listFolderContinue(ctx context.Context, params map[string]any) (string, error) {
	cursor, _ := params["cursor"].(string)
	if cursor == "" {
		return "", fmt.Errorf("cursor is required")
	}
	return doPost(ctx, "/files/list_folder/continue", map[string]any{"cursor": cursor})
}

func getMetadata(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	body := map[string]any{"path": path}
	if v, ok := params["include_media_info"].(bool); ok {
		body["include_media_info"] = v
	}
	if v, ok := params["include_deleted"].(bool); ok {
		body["include_deleted"] = v
	}

	return doPost(ctx, "/files/get_metadata", body)
}

func searchFiles(ctx context.Context, params map[string]any) (string, error) {
	query, _ := params["query"].(string)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	body := map[string]any{"query": query}

	options := map[string]any{}
	if v, ok := params["path"].(string); ok && v != "" {
		options["path"] = v
	}
	if v, ok := params["max_results"].(float64); ok {
		options["max_results"] = int(v)
	}
	if categories, ok := params["file_categories"].([]any); ok && len(categories) > 0 {
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

	return doPost(ctx, "/files/search_v2", body)
}

func readFile(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	return doContentDownload(ctx, "/files/download", map[string]string{"path": path})
}

func listSharedLinks(ctx context.Context, params map[string]any) (string, error) {
	body := map[string]any{}
	if v, ok := params["path"].(string); ok && v != "" {
		body["path"] = v
	}
	if v, ok := params["cursor"].(string); ok && v != "" {
		body["cursor"] = v
	}
	return doPost(ctx, "/sharing/list_shared_links", body)
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

	apiArg := map[string]any{"path": path, "mode": "add"}
	if v, ok := params["mode"].(string); ok && v != "" {
		apiArg["mode"] = v
	}
	if v, ok := params["autorename"].(bool); ok {
		apiArg["autorename"] = v
	}
	if v, ok := params["mute"].(bool); ok {
		apiArg["mute"] = v
	}

	return doContentUpload(ctx, "/files/upload", apiArg, content)
}

func createFolder(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	body := map[string]any{"path": path}
	if v, ok := params["autorename"].(bool); ok {
		body["autorename"] = v
	}

	return doPost(ctx, "/files/create_folder_v2", body)
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

	body := map[string]any{"from_path": fromPath, "to_path": toPath}
	if v, ok := params["autorename"].(bool); ok {
		body["autorename"] = v
	}

	return doPost(ctx, "/files/copy_v2", body)
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

	body := map[string]any{"from_path": fromPath, "to_path": toPath}
	if v, ok := params["autorename"].(bool); ok {
		body["autorename"] = v
	}

	return doPost(ctx, "/files/move_v2", body)
}

func deleteFile(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	return doPost(ctx, "/files/delete_v2", map[string]any{"path": path})
}

func createSharedLink(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	body := map[string]any{"path": path}
	if v, ok := params["requested_visibility"].(string); ok && v != "" {
		body["settings"] = map[string]any{"requested_visibility": v}
	}

	return doPost(ctx, "/sharing/create_shared_link_with_settings", body)
}

func listRevisions(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	body := map[string]any{"path": path}
	if v, ok := params["mode"].(string); ok && v != "" {
		body["mode"] = v
	}
	if v, ok := params["limit"].(float64); ok {
		body["limit"] = int(v)
	}

	return doPost(ctx, "/files/list_revisions", body)
}
