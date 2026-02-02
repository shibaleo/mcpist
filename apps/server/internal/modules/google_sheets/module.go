package google_sheets

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
	googleSheetsAPIBase = "https://sheets.googleapis.com/v4"
	googleDriveAPIBase  = "https://www.googleapis.com/drive/v3"
	googleSheetsVersion = "v4"
	googleTokenURL      = "https://oauth2.googleapis.com/token"
	tokenRefreshBuffer  = 5 * 60 // Refresh 5 minutes before expiry
)

var client = httpclient.New()

// GoogleSheetsModule implements the Module interface for Google Sheets API
type GoogleSheetsModule struct{}

// New creates a new GoogleSheetsModule instance
func New() *GoogleSheetsModule {
	return &GoogleSheetsModule{}
}

// Module descriptions in multiple languages
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Google Sheets API - Read, create, and edit Google Spreadsheets",
	"ja-JP": "Google Sheets API - Google スプレッドシートの読み取り、作成、編集",
}

// Name returns the module name
func (m *GoogleSheetsModule) Name() string {
	return "google_sheets"
}

// Descriptions returns the module descriptions in all languages
func (m *GoogleSheetsModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description for a specific language
func (m *GoogleSheetsModule) Description(lang string) string {
	return modules.GetLocalizedText(moduleDescriptions, lang)
}

// APIVersion returns the Google Sheets API version
func (m *GoogleSheetsModule) APIVersion() string {
	return googleSheetsVersion
}

// Tools returns all available tools
func (m *GoogleSheetsModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *GoogleSheetsModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for Google Sheets)
func (m *GoogleSheetsModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *GoogleSheetsModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *store.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[google_sheets] No auth context")
		return nil
	}
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "google_sheets")
	if err != nil {
		log.Printf("[google_sheets] GetModuleToken error: %v", err)
		return nil
	}
	log.Printf("[google_sheets] Got credentials: auth_type=%s, has_access_token=%v", credentials.AuthType, credentials.AccessToken != "")

	// Check if token needs refresh (OAuth2 only)
	if credentials.AuthType == store.AuthTypeOAuth2 && credentials.RefreshToken != "" {
		if needsRefresh(credentials) {
			log.Printf("[google_sheets] Token expired or expiring soon, refreshing...")
			refreshed, err := refreshToken(ctx, authCtx.UserID, credentials)
			if err != nil {
				log.Printf("[google_sheets] Token refresh failed: %v", err)
				return credentials
			}
			log.Printf("[google_sheets] Token refreshed successfully")
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

	if err := store.GetTokenStore().UpdateModuleToken(ctx, userID, "google_sheets", newCreds); err != nil {
		log.Printf("[google_sheets] Failed to update token in store: %v", err)
	}

	return newCreds, nil
}

// headers builds request headers with auth token
func headers(ctx context.Context) map[string]string {
	creds := getCredentials(ctx)
	if creds == nil {
		log.Printf("[google_sheets] No credentials available")
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
	// Spreadsheet Operations
	// =========================================================================
	{
		ID:   "google_sheets:get_spreadsheet",
		Name: "get_spreadsheet",
		Descriptions: modules.LocalizedText{
			"en-US": "Get spreadsheet metadata including title, sheets, and properties.",
			"ja-JP": "スプレッドシートのメタデータ（タイトル、シート、プロパティ）を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
			},
			Required: []string{"spreadsheet_id"},
		},
	},
	{
		ID:   "google_sheets:create_spreadsheet",
		Name: "create_spreadsheet",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new spreadsheet.",
			"ja-JP": "新しいスプレッドシートを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"title":       {Type: "string", Description: "Spreadsheet title"},
				"sheet_names": {Type: "array", Description: "Initial sheet names (optional). Default: ['Sheet1']"},
			},
			Required: []string{"title"},
		},
	},
	{
		ID:   "google_sheets:search_spreadsheets",
		Name: "search_spreadsheets",
		Descriptions: modules.LocalizedText{
			"en-US": "Search for spreadsheets in Google Drive.",
			"ja-JP": "Google Drive内のスプレッドシートを検索します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"query":     {Type: "string", Description: "Search query (searches in file name)"},
				"page_size": {Type: "number", Description: "Maximum results (1-100). Default: 20"},
			},
		},
	},
	// =========================================================================
	// Sheet (Tab) Operations
	// =========================================================================
	{
		ID:   "google_sheets:list_sheets",
		Name: "list_sheets",
		Descriptions: modules.LocalizedText{
			"en-US": "List all sheets (tabs) in a spreadsheet.",
			"ja-JP": "スプレッドシート内のすべてのシート（タブ）を一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
			},
			Required: []string{"spreadsheet_id"},
		},
	},
	{
		ID:   "google_sheets:create_sheet",
		Name: "create_sheet",
		Descriptions: modules.LocalizedText{
			"en-US": "Add a new sheet (tab) to a spreadsheet.",
			"ja-JP": "スプレッドシートに新しいシート（タブ）を追加します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"title":          {Type: "string", Description: "New sheet title"},
				"index":          {Type: "number", Description: "Position to insert (0-based). Default: append at end"},
			},
			Required: []string{"spreadsheet_id", "title"},
		},
	},
	{
		ID:   "google_sheets:delete_sheet",
		Name: "delete_sheet",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a sheet (tab) from a spreadsheet.",
			"ja-JP": "スプレッドシートからシート（タブ）を削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"sheet_id":       {Type: "number", Description: "Sheet ID to delete"},
			},
			Required: []string{"spreadsheet_id", "sheet_id"},
		},
	},
	{
		ID:   "google_sheets:rename_sheet",
		Name: "rename_sheet",
		Descriptions: modules.LocalizedText{
			"en-US": "Rename a sheet (tab) in a spreadsheet.",
			"ja-JP": "スプレッドシート内のシート（タブ）の名前を変更します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"sheet_id":       {Type: "number", Description: "Sheet ID to rename"},
				"title":          {Type: "string", Description: "New sheet title"},
			},
			Required: []string{"spreadsheet_id", "sheet_id", "title"},
		},
	},
	{
		ID:   "google_sheets:duplicate_sheet",
		Name: "duplicate_sheet",
		Descriptions: modules.LocalizedText{
			"en-US": "Duplicate a sheet within the same spreadsheet.",
			"ja-JP": "同じスプレッドシート内でシートを複製します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"sheet_id":       {Type: "number", Description: "Sheet ID to duplicate"},
				"new_title":      {Type: "string", Description: "Title for the new sheet"},
				"insert_index":   {Type: "number", Description: "Position to insert (0-based). Default: after source"},
			},
			Required: []string{"spreadsheet_id", "sheet_id"},
		},
	},
	{
		ID:   "google_sheets:copy_sheet_to",
		Name: "copy_sheet_to",
		Descriptions: modules.LocalizedText{
			"en-US": "Copy a sheet to another spreadsheet.",
			"ja-JP": "シートを別のスプレッドシートにコピーします。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"source_spreadsheet_id": {Type: "string", Description: "Source spreadsheet ID"},
				"sheet_id":              {Type: "number", Description: "Sheet ID to copy"},
				"dest_spreadsheet_id":   {Type: "string", Description: "Destination spreadsheet ID"},
			},
			Required: []string{"source_spreadsheet_id", "sheet_id", "dest_spreadsheet_id"},
		},
	},
	// =========================================================================
	// Data Operations - Read
	// =========================================================================
	{
		ID:   "google_sheets:get_values",
		Name: "get_values",
		Descriptions: modules.LocalizedText{
			"en-US": "Get cell values from a range (e.g., 'Sheet1!A1:C10').",
			"ja-JP": "指定範囲のセル値を取得します（例: 'Sheet1!A1:C10'）。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id":     {Type: "string", Description: "Spreadsheet ID"},
				"range":              {Type: "string", Description: "A1 notation range (e.g., 'Sheet1!A1:C10', 'A1:C10')"},
				"value_render":       {Type: "string", Description: "How values should be rendered: 'FORMATTED_VALUE' (default), 'UNFORMATTED_VALUE', or 'FORMULA'"},
				"date_time_render":   {Type: "string", Description: "How dates should be rendered: 'SERIAL_NUMBER' or 'FORMATTED_STRING' (default)"},
			},
			Required: []string{"spreadsheet_id", "range"},
		},
	},
	{
		ID:   "google_sheets:batch_get_values",
		Name: "batch_get_values",
		Descriptions: modules.LocalizedText{
			"en-US": "Get cell values from multiple ranges at once.",
			"ja-JP": "複数の範囲からセル値を一度に取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id":   {Type: "string", Description: "Spreadsheet ID"},
				"ranges":           {Type: "array", Description: "Array of A1 notation ranges"},
				"value_render":     {Type: "string", Description: "How values should be rendered"},
				"date_time_render": {Type: "string", Description: "How dates should be rendered"},
			},
			Required: []string{"spreadsheet_id", "ranges"},
		},
	},
	{
		ID:   "google_sheets:get_formulas",
		Name: "get_formulas",
		Descriptions: modules.LocalizedText{
			"en-US": "Get formulas from a range.",
			"ja-JP": "指定範囲の数式を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"range":          {Type: "string", Description: "A1 notation range"},
			},
			Required: []string{"spreadsheet_id", "range"},
		},
	},
	// =========================================================================
	// Data Operations - Write
	// =========================================================================
	{
		ID:   "google_sheets:update_values",
		Name: "update_values",
		Descriptions: modules.LocalizedText{
			"en-US": "Update cell values in a range.",
			"ja-JP": "指定範囲のセル値を更新します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"range":          {Type: "string", Description: "A1 notation range (e.g., 'Sheet1!A1:C3')"},
				"values":         {Type: "array", Description: "2D array of values [[row1], [row2], ...]"},
				"value_input":    {Type: "string", Description: "How input should be interpreted: 'RAW' or 'USER_ENTERED' (default)"},
			},
			Required: []string{"spreadsheet_id", "range", "values"},
		},
	},
	{
		ID:   "google_sheets:batch_update_values",
		Name: "batch_update_values",
		Descriptions: modules.LocalizedText{
			"en-US": "Update cell values in multiple ranges at once.",
			"ja-JP": "複数の範囲のセル値を一度に更新します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"data": {Type: "array", Description: "Array of {range, values} objects. Example: [{\"range\": \"A1:B2\", \"values\": [[1,2],[3,4]]}]"},
				"value_input": {Type: "string", Description: "How input should be interpreted: 'RAW' or 'USER_ENTERED' (default)"},
			},
			Required: []string{"spreadsheet_id", "data"},
		},
	},
	{
		ID:   "google_sheets:append_values",
		Name: "append_values",
		Descriptions: modules.LocalizedText{
			"en-US": "Append rows to a table (finds the last row and appends data).",
			"ja-JP": "テーブルに行を追加します（最後の行を見つけてデータを追加）。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"range":          {Type: "string", Description: "A1 notation range to search for table (e.g., 'Sheet1!A:C')"},
				"values":         {Type: "array", Description: "2D array of values to append [[row1], [row2], ...]"},
				"value_input":    {Type: "string", Description: "How input should be interpreted: 'RAW' or 'USER_ENTERED' (default)"},
				"insert_data":    {Type: "string", Description: "How to insert: 'OVERWRITE' or 'INSERT_ROWS' (default)"},
			},
			Required: []string{"spreadsheet_id", "range", "values"},
		},
	},
	{
		ID:   "google_sheets:clear_values",
		Name: "clear_values",
		Descriptions: modules.LocalizedText{
			"en-US": "Clear cell contents in a range (keeps formatting).",
			"ja-JP": "指定範囲のセル内容をクリアします（書式は保持）。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"range":          {Type: "string", Description: "A1 notation range to clear"},
			},
			Required: []string{"spreadsheet_id", "range"},
		},
	},
	// =========================================================================
	// Row/Column Operations
	// =========================================================================
	{
		ID:   "google_sheets:insert_rows",
		Name: "insert_rows",
		Descriptions: modules.LocalizedText{
			"en-US": "Insert empty rows at a specific position.",
			"ja-JP": "指定位置に空の行を挿入します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"sheet_id":       {Type: "number", Description: "Sheet ID"},
				"start_index":    {Type: "number", Description: "Row index to start inserting (0-based)"},
				"num_rows":       {Type: "number", Description: "Number of rows to insert"},
			},
			Required: []string{"spreadsheet_id", "sheet_id", "start_index", "num_rows"},
		},
	},
	{
		ID:   "google_sheets:delete_rows",
		Name: "delete_rows",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete rows from a sheet.",
			"ja-JP": "シートから行を削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"sheet_id":       {Type: "number", Description: "Sheet ID"},
				"start_index":    {Type: "number", Description: "Starting row index (0-based)"},
				"end_index":      {Type: "number", Description: "Ending row index (exclusive)"},
			},
			Required: []string{"spreadsheet_id", "sheet_id", "start_index", "end_index"},
		},
	},
	{
		ID:   "google_sheets:insert_columns",
		Name: "insert_columns",
		Descriptions: modules.LocalizedText{
			"en-US": "Insert empty columns at a specific position.",
			"ja-JP": "指定位置に空の列を挿入します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"sheet_id":       {Type: "number", Description: "Sheet ID"},
				"start_index":    {Type: "number", Description: "Column index to start inserting (0-based)"},
				"num_columns":    {Type: "number", Description: "Number of columns to insert"},
			},
			Required: []string{"spreadsheet_id", "sheet_id", "start_index", "num_columns"},
		},
	},
	{
		ID:   "google_sheets:delete_columns",
		Name: "delete_columns",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete columns from a sheet.",
			"ja-JP": "シートから列を削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"sheet_id":       {Type: "number", Description: "Sheet ID"},
				"start_index":    {Type: "number", Description: "Starting column index (0-based)"},
				"end_index":      {Type: "number", Description: "Ending column index (exclusive)"},
			},
			Required: []string{"spreadsheet_id", "sheet_id", "start_index", "end_index"},
		},
	},
	// =========================================================================
	// Formatting
	// =========================================================================
	{
		ID:   "google_sheets:format_cells",
		Name: "format_cells",
		Descriptions: modules.LocalizedText{
			"en-US": "Format cells (background color, text format, alignment, number format).",
			"ja-JP": "セルの書式を設定します（背景色、テキスト書式、配置、数値形式）。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id":   {Type: "string", Description: "Spreadsheet ID"},
				"sheet_id":         {Type: "number", Description: "Sheet ID"},
				"start_row":        {Type: "number", Description: "Start row index (0-based)"},
				"end_row":          {Type: "number", Description: "End row index (exclusive)"},
				"start_column":     {Type: "number", Description: "Start column index (0-based)"},
				"end_column":       {Type: "number", Description: "End column index (exclusive)"},
				"background_color": {Type: "object", Description: "Background color {red, green, blue, alpha} (0-1 floats)"},
				"bold":             {Type: "boolean", Description: "Make text bold"},
				"italic":           {Type: "boolean", Description: "Make text italic"},
				"font_size":        {Type: "number", Description: "Font size in points"},
				"font_color":       {Type: "object", Description: "Font color {red, green, blue, alpha} (0-1 floats)"},
				"h_align":          {Type: "string", Description: "Horizontal alignment: 'LEFT', 'CENTER', 'RIGHT'"},
				"v_align":          {Type: "string", Description: "Vertical alignment: 'TOP', 'MIDDLE', 'BOTTOM'"},
				"number_format":    {Type: "string", Description: "Number format pattern (e.g., '#,##0.00', '0%', 'yyyy-mm-dd')"},
				"wrap_strategy":    {Type: "string", Description: "Text wrap: 'OVERFLOW_CELL', 'LEGACY_WRAP', 'CLIP', 'WRAP'"},
			},
			Required: []string{"spreadsheet_id", "sheet_id", "start_row", "end_row", "start_column", "end_column"},
		},
	},
	{
		ID:   "google_sheets:merge_cells",
		Name: "merge_cells",
		Descriptions: modules.LocalizedText{
			"en-US": "Merge cells in a range.",
			"ja-JP": "指定範囲のセルを結合します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"sheet_id":       {Type: "number", Description: "Sheet ID"},
				"start_row":      {Type: "number", Description: "Start row index (0-based)"},
				"end_row":        {Type: "number", Description: "End row index (exclusive)"},
				"start_column":   {Type: "number", Description: "Start column index (0-based)"},
				"end_column":     {Type: "number", Description: "End column index (exclusive)"},
				"merge_type":     {Type: "string", Description: "Merge type: 'MERGE_ALL' (default), 'MERGE_COLUMNS', 'MERGE_ROWS'"},
			},
			Required: []string{"spreadsheet_id", "sheet_id", "start_row", "end_row", "start_column", "end_column"},
		},
	},
	{
		ID:   "google_sheets:unmerge_cells",
		Name: "unmerge_cells",
		Descriptions: modules.LocalizedText{
			"en-US": "Unmerge previously merged cells.",
			"ja-JP": "結合されたセルを解除します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"sheet_id":       {Type: "number", Description: "Sheet ID"},
				"start_row":      {Type: "number", Description: "Start row index (0-based)"},
				"end_row":        {Type: "number", Description: "End row index (exclusive)"},
				"start_column":   {Type: "number", Description: "Start column index (0-based)"},
				"end_column":     {Type: "number", Description: "End column index (exclusive)"},
			},
			Required: []string{"spreadsheet_id", "sheet_id", "start_row", "end_row", "start_column", "end_column"},
		},
	},
	{
		ID:   "google_sheets:set_borders",
		Name: "set_borders",
		Descriptions: modules.LocalizedText{
			"en-US": "Set borders for a range of cells.",
			"ja-JP": "セル範囲に罫線を設定します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"sheet_id":       {Type: "number", Description: "Sheet ID"},
				"start_row":      {Type: "number", Description: "Start row index (0-based)"},
				"end_row":        {Type: "number", Description: "End row index (exclusive)"},
				"start_column":   {Type: "number", Description: "Start column index (0-based)"},
				"end_column":     {Type: "number", Description: "End column index (exclusive)"},
				"style":          {Type: "string", Description: "Border style: 'SOLID', 'SOLID_MEDIUM', 'SOLID_THICK', 'DASHED', 'DOTTED', 'DOUBLE'"},
				"color":          {Type: "object", Description: "Border color {red, green, blue, alpha} (0-1 floats)"},
				"top":            {Type: "boolean", Description: "Apply to top border"},
				"bottom":         {Type: "boolean", Description: "Apply to bottom border"},
				"left":           {Type: "boolean", Description: "Apply to left border"},
				"right":          {Type: "boolean", Description: "Apply to right border"},
				"inner_h":        {Type: "boolean", Description: "Apply to inner horizontal borders"},
				"inner_v":        {Type: "boolean", Description: "Apply to inner vertical borders"},
			},
			Required: []string{"spreadsheet_id", "sheet_id", "start_row", "end_row", "start_column", "end_column"},
		},
	},
	{
		ID:   "google_sheets:auto_resize",
		Name: "auto_resize",
		Descriptions: modules.LocalizedText{
			"en-US": "Auto-resize columns or rows to fit content.",
			"ja-JP": "列または行のサイズをコンテンツに合わせて自動調整します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"sheet_id":       {Type: "number", Description: "Sheet ID"},
				"dimension":      {Type: "string", Description: "Dimension: 'ROWS' or 'COLUMNS'"},
				"start_index":    {Type: "number", Description: "Start index (0-based)"},
				"end_index":      {Type: "number", Description: "End index (exclusive)"},
			},
			Required: []string{"spreadsheet_id", "sheet_id", "dimension", "start_index", "end_index"},
		},
	},
	// =========================================================================
	// Find & Replace
	// =========================================================================
	{
		ID:   "google_sheets:find_replace",
		Name: "find_replace",
		Descriptions: modules.LocalizedText{
			"en-US": "Find and replace text in a spreadsheet.",
			"ja-JP": "スプレッドシート内のテキストを検索・置換します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"find":           {Type: "string", Description: "Text to find"},
				"replacement":    {Type: "string", Description: "Replacement text"},
				"match_case":     {Type: "boolean", Description: "Match case. Default: false"},
				"match_entire":   {Type: "boolean", Description: "Match entire cell content. Default: false"},
				"use_regex":      {Type: "boolean", Description: "Use regular expressions. Default: false"},
				"sheet_id":       {Type: "number", Description: "Limit to specific sheet (optional)"},
				"range":          {Type: "string", Description: "Limit to specific range in A1 notation (optional)"},
			},
			Required: []string{"spreadsheet_id", "find", "replacement"},
		},
	},
	// =========================================================================
	// Protection
	// =========================================================================
	{
		ID:   "google_sheets:protect_range",
		Name: "protect_range",
		Descriptions: modules.LocalizedText{
			"en-US": "Protect a range or sheet from editing.",
			"ja-JP": "範囲またはシートを編集から保護します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				"sheet_id":       {Type: "number", Description: "Sheet ID"},
				"description":    {Type: "string", Description: "Description of the protected range"},
				"start_row":      {Type: "number", Description: "Start row index (0-based). Omit to protect entire sheet"},
				"end_row":        {Type: "number", Description: "End row index (exclusive)"},
				"start_column":   {Type: "number", Description: "Start column index (0-based)"},
				"end_column":     {Type: "number", Description: "End column index (exclusive)"},
				"warning_only":   {Type: "boolean", Description: "Show warning instead of blocking. Default: false"},
			},
			Required: []string{"spreadsheet_id", "sheet_id"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	// Spreadsheet operations
	"get_spreadsheet":     getSpreadsheet,
	"create_spreadsheet":  createSpreadsheet,
	"search_spreadsheets": searchSpreadsheets,
	// Sheet operations
	"list_sheets":     listSheets,
	"create_sheet":    createSheet,
	"delete_sheet":    deleteSheet,
	"rename_sheet":    renameSheet,
	"duplicate_sheet": duplicateSheet,
	"copy_sheet_to":   copySheetTo,
	// Data read operations
	"get_values":       getValues,
	"batch_get_values": batchGetValues,
	"get_formulas":     getFormulas,
	// Data write operations
	"update_values":       updateValues,
	"batch_update_values": batchUpdateValues,
	"append_values":       appendValues,
	"clear_values":        clearValues,
	// Row/Column operations
	"insert_rows":    insertRows,
	"delete_rows":    deleteRows,
	"insert_columns": insertColumns,
	"delete_columns": deleteColumns,
	// Formatting
	"format_cells":  formatCells,
	"merge_cells":   mergeCells,
	"unmerge_cells": unmergeCells,
	"set_borders":   setBorders,
	"auto_resize":   autoResize,
	// Find & Replace
	"find_replace": findReplace,
	// Protection
	"protect_range": protectRange,
}

// =============================================================================
// Spreadsheet Operations
// =============================================================================

func getSpreadsheet(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)

	endpoint := fmt.Sprintf("%s/spreadsheets/%s?fields=spreadsheetId,properties,sheets.properties",
		googleSheetsAPIBase, url.PathEscape(spreadsheetID))

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createSpreadsheet(ctx context.Context, params map[string]any) (string, error) {
	title, _ := params["title"].(string)

	body := map[string]interface{}{
		"properties": map[string]interface{}{
			"title": title,
		},
	}

	// Add initial sheets if specified
	if sheetNames, ok := params["sheet_names"].([]interface{}); ok && len(sheetNames) > 0 {
		sheets := make([]map[string]interface{}, len(sheetNames))
		for i, name := range sheetNames {
			sheets[i] = map[string]interface{}{
				"properties": map[string]interface{}{
					"title": name,
				},
			}
		}
		body["sheets"] = sheets
	}

	endpoint := fmt.Sprintf("%s/spreadsheets", googleSheetsAPIBase)
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func searchSpreadsheets(ctx context.Context, params map[string]any) (string, error) {
	query := url.Values{}
	query.Set("q", "mimeType='application/vnd.google-apps.spreadsheet'")
	if q, ok := params["query"].(string); ok && q != "" {
		query.Set("q", fmt.Sprintf("mimeType='application/vnd.google-apps.spreadsheet' and name contains '%s'", q))
	}

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

// =============================================================================
// Sheet (Tab) Operations
// =============================================================================

func listSheets(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)

	endpoint := fmt.Sprintf("%s/spreadsheets/%s?fields=sheets.properties",
		googleSheetsAPIBase, url.PathEscape(spreadsheetID))

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createSheet(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	title, _ := params["title"].(string)

	request := map[string]interface{}{
		"addSheet": map[string]interface{}{
			"properties": map[string]interface{}{
				"title": title,
			},
		},
	}

	if idx, ok := params["index"].(float64); ok {
		request["addSheet"].(map[string]interface{})["properties"].(map[string]interface{})["index"] = int(idx)
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

func deleteSheet(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)

	request := map[string]interface{}{
		"deleteSheet": map[string]interface{}{
			"sheetId": int(sheetID),
		},
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

func renameSheet(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	title, _ := params["title"].(string)

	request := map[string]interface{}{
		"updateSheetProperties": map[string]interface{}{
			"properties": map[string]interface{}{
				"sheetId": int(sheetID),
				"title":   title,
			},
			"fields": "title",
		},
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

func duplicateSheet(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)

	request := map[string]interface{}{
		"duplicateSheet": map[string]interface{}{
			"sourceSheetId": int(sheetID),
		},
	}

	if newTitle, ok := params["new_title"].(string); ok && newTitle != "" {
		request["duplicateSheet"].(map[string]interface{})["newSheetName"] = newTitle
	}
	if insertIndex, ok := params["insert_index"].(float64); ok {
		request["duplicateSheet"].(map[string]interface{})["insertSheetIndex"] = int(insertIndex)
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

func copySheetTo(ctx context.Context, params map[string]any) (string, error) {
	sourceSpreadsheetID, _ := params["source_spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	destSpreadsheetID, _ := params["dest_spreadsheet_id"].(string)

	body := map[string]interface{}{
		"destinationSpreadsheetId": destSpreadsheetID,
	}

	endpoint := fmt.Sprintf("%s/spreadsheets/%s/sheets/%d:copyTo",
		googleSheetsAPIBase, url.PathEscape(sourceSpreadsheetID), int(sheetID))

	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Data Operations - Read
// =============================================================================

func getValues(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	rangeStr, _ := params["range"].(string)

	query := url.Values{}
	if vr, ok := params["value_render"].(string); ok && vr != "" {
		query.Set("valueRenderOption", vr)
	}
	if dtr, ok := params["date_time_render"].(string); ok && dtr != "" {
		query.Set("dateTimeRenderOption", dtr)
	}

	endpoint := fmt.Sprintf("%s/spreadsheets/%s/values/%s?%s",
		googleSheetsAPIBase, url.PathEscape(spreadsheetID), url.PathEscape(rangeStr), query.Encode())

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func batchGetValues(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	ranges, _ := params["ranges"].([]interface{})

	query := url.Values{}
	for _, r := range ranges {
		if rangeStr, ok := r.(string); ok {
			query.Add("ranges", rangeStr)
		}
	}
	if vr, ok := params["value_render"].(string); ok && vr != "" {
		query.Set("valueRenderOption", vr)
	}
	if dtr, ok := params["date_time_render"].(string); ok && dtr != "" {
		query.Set("dateTimeRenderOption", dtr)
	}

	endpoint := fmt.Sprintf("%s/spreadsheets/%s/values:batchGet?%s",
		googleSheetsAPIBase, url.PathEscape(spreadsheetID), query.Encode())

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getFormulas(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	rangeStr, _ := params["range"].(string)

	query := url.Values{}
	query.Set("valueRenderOption", "FORMULA")

	endpoint := fmt.Sprintf("%s/spreadsheets/%s/values/%s?%s",
		googleSheetsAPIBase, url.PathEscape(spreadsheetID), url.PathEscape(rangeStr), query.Encode())

	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Data Operations - Write
// =============================================================================

func updateValues(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	rangeStr, _ := params["range"].(string)
	values, _ := params["values"].([]interface{})

	valueInput := "USER_ENTERED"
	if vi, ok := params["value_input"].(string); ok && vi != "" {
		valueInput = vi
	}

	body := map[string]interface{}{
		"values": values,
	}

	query := url.Values{}
	query.Set("valueInputOption", valueInput)

	endpoint := fmt.Sprintf("%s/spreadsheets/%s/values/%s?%s",
		googleSheetsAPIBase, url.PathEscape(spreadsheetID), url.PathEscape(rangeStr), query.Encode())

	respBody, err := client.DoJSON("PUT", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func batchUpdateValues(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	data, _ := params["data"].([]interface{})

	valueInput := "USER_ENTERED"
	if vi, ok := params["value_input"].(string); ok && vi != "" {
		valueInput = vi
	}

	body := map[string]interface{}{
		"valueInputOption": valueInput,
		"data":             data,
	}

	endpoint := fmt.Sprintf("%s/spreadsheets/%s/values:batchUpdate",
		googleSheetsAPIBase, url.PathEscape(spreadsheetID))

	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func appendValues(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	rangeStr, _ := params["range"].(string)
	values, _ := params["values"].([]interface{})

	valueInput := "USER_ENTERED"
	if vi, ok := params["value_input"].(string); ok && vi != "" {
		valueInput = vi
	}

	insertData := "INSERT_ROWS"
	if id, ok := params["insert_data"].(string); ok && id != "" {
		insertData = id
	}

	body := map[string]interface{}{
		"values": values,
	}

	query := url.Values{}
	query.Set("valueInputOption", valueInput)
	query.Set("insertDataOption", insertData)

	endpoint := fmt.Sprintf("%s/spreadsheets/%s/values/%s:append?%s",
		googleSheetsAPIBase, url.PathEscape(spreadsheetID), url.PathEscape(rangeStr), query.Encode())

	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func clearValues(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	rangeStr, _ := params["range"].(string)

	endpoint := fmt.Sprintf("%s/spreadsheets/%s/values/%s:clear",
		googleSheetsAPIBase, url.PathEscape(spreadsheetID), url.PathEscape(rangeStr))

	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), map[string]interface{}{})
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Row/Column Operations
// =============================================================================

func insertRows(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	startIndex, _ := params["start_index"].(float64)
	numRows, _ := params["num_rows"].(float64)

	request := map[string]interface{}{
		"insertDimension": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId":    int(sheetID),
				"dimension":  "ROWS",
				"startIndex": int(startIndex),
				"endIndex":   int(startIndex) + int(numRows),
			},
			"inheritFromBefore": startIndex > 0,
		},
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

func deleteRows(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	startIndex, _ := params["start_index"].(float64)
	endIndex, _ := params["end_index"].(float64)

	request := map[string]interface{}{
		"deleteDimension": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId":    int(sheetID),
				"dimension":  "ROWS",
				"startIndex": int(startIndex),
				"endIndex":   int(endIndex),
			},
		},
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

func insertColumns(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	startIndex, _ := params["start_index"].(float64)
	numColumns, _ := params["num_columns"].(float64)

	request := map[string]interface{}{
		"insertDimension": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId":    int(sheetID),
				"dimension":  "COLUMNS",
				"startIndex": int(startIndex),
				"endIndex":   int(startIndex) + int(numColumns),
			},
			"inheritFromBefore": startIndex > 0,
		},
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

func deleteColumns(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	startIndex, _ := params["start_index"].(float64)
	endIndex, _ := params["end_index"].(float64)

	request := map[string]interface{}{
		"deleteDimension": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId":    int(sheetID),
				"dimension":  "COLUMNS",
				"startIndex": int(startIndex),
				"endIndex":   int(endIndex),
			},
		},
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

// =============================================================================
// Formatting
// =============================================================================

func formatCells(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	startRow, _ := params["start_row"].(float64)
	endRow, _ := params["end_row"].(float64)
	startColumn, _ := params["start_column"].(float64)
	endColumn, _ := params["end_column"].(float64)

	cellFormat := map[string]interface{}{}
	fields := []string{}

	// Background color
	if bgColor, ok := params["background_color"].(map[string]interface{}); ok {
		cellFormat["backgroundColor"] = bgColor
		fields = append(fields, "userEnteredFormat.backgroundColor")
	}

	// Text format
	textFormat := map[string]interface{}{}
	if bold, ok := params["bold"].(bool); ok {
		textFormat["bold"] = bold
		fields = append(fields, "userEnteredFormat.textFormat.bold")
	}
	if italic, ok := params["italic"].(bool); ok {
		textFormat["italic"] = italic
		fields = append(fields, "userEnteredFormat.textFormat.italic")
	}
	if fontSize, ok := params["font_size"].(float64); ok {
		textFormat["fontSize"] = int(fontSize)
		fields = append(fields, "userEnteredFormat.textFormat.fontSize")
	}
	if fontColor, ok := params["font_color"].(map[string]interface{}); ok {
		textFormat["foregroundColor"] = fontColor
		fields = append(fields, "userEnteredFormat.textFormat.foregroundColor")
	}
	if len(textFormat) > 0 {
		cellFormat["textFormat"] = textFormat
	}

	// Alignment
	if hAlign, ok := params["h_align"].(string); ok {
		cellFormat["horizontalAlignment"] = hAlign
		fields = append(fields, "userEnteredFormat.horizontalAlignment")
	}
	if vAlign, ok := params["v_align"].(string); ok {
		cellFormat["verticalAlignment"] = vAlign
		fields = append(fields, "userEnteredFormat.verticalAlignment")
	}

	// Number format
	if numFormat, ok := params["number_format"].(string); ok {
		cellFormat["numberFormat"] = map[string]interface{}{
			"type":    "NUMBER",
			"pattern": numFormat,
		}
		fields = append(fields, "userEnteredFormat.numberFormat")
	}

	// Wrap strategy
	if wrapStrategy, ok := params["wrap_strategy"].(string); ok {
		cellFormat["wrapStrategy"] = wrapStrategy
		fields = append(fields, "userEnteredFormat.wrapStrategy")
	}

	if len(fields) == 0 {
		return "", fmt.Errorf("no formatting options specified")
	}

	request := map[string]interface{}{
		"repeatCell": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId":          int(sheetID),
				"startRowIndex":    int(startRow),
				"endRowIndex":      int(endRow),
				"startColumnIndex": int(startColumn),
				"endColumnIndex":   int(endColumn),
			},
			"cell": map[string]interface{}{
				"userEnteredFormat": cellFormat,
			},
			"fields": strings.Join(fields, ","),
		},
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

func mergeCells(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	startRow, _ := params["start_row"].(float64)
	endRow, _ := params["end_row"].(float64)
	startColumn, _ := params["start_column"].(float64)
	endColumn, _ := params["end_column"].(float64)

	mergeType := "MERGE_ALL"
	if mt, ok := params["merge_type"].(string); ok && mt != "" {
		mergeType = mt
	}

	request := map[string]interface{}{
		"mergeCells": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId":          int(sheetID),
				"startRowIndex":    int(startRow),
				"endRowIndex":      int(endRow),
				"startColumnIndex": int(startColumn),
				"endColumnIndex":   int(endColumn),
			},
			"mergeType": mergeType,
		},
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

func unmergeCells(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	startRow, _ := params["start_row"].(float64)
	endRow, _ := params["end_row"].(float64)
	startColumn, _ := params["start_column"].(float64)
	endColumn, _ := params["end_column"].(float64)

	request := map[string]interface{}{
		"unmergeCells": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId":          int(sheetID),
				"startRowIndex":    int(startRow),
				"endRowIndex":      int(endRow),
				"startColumnIndex": int(startColumn),
				"endColumnIndex":   int(endColumn),
			},
		},
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

func setBorders(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	startRow, _ := params["start_row"].(float64)
	endRow, _ := params["end_row"].(float64)
	startColumn, _ := params["start_column"].(float64)
	endColumn, _ := params["end_column"].(float64)

	style := "SOLID"
	if s, ok := params["style"].(string); ok && s != "" {
		style = s
	}

	color := map[string]interface{}{"red": 0, "green": 0, "blue": 0}
	if c, ok := params["color"].(map[string]interface{}); ok {
		color = c
	}

	border := map[string]interface{}{
		"style": style,
		"color": color,
	}

	updateBorders := map[string]interface{}{
		"range": map[string]interface{}{
			"sheetId":          int(sheetID),
			"startRowIndex":    int(startRow),
			"endRowIndex":      int(endRow),
			"startColumnIndex": int(startColumn),
			"endColumnIndex":   int(endColumn),
		},
	}

	if top, ok := params["top"].(bool); ok && top {
		updateBorders["top"] = border
	}
	if bottom, ok := params["bottom"].(bool); ok && bottom {
		updateBorders["bottom"] = border
	}
	if left, ok := params["left"].(bool); ok && left {
		updateBorders["left"] = border
	}
	if right, ok := params["right"].(bool); ok && right {
		updateBorders["right"] = border
	}
	if innerH, ok := params["inner_h"].(bool); ok && innerH {
		updateBorders["innerHorizontal"] = border
	}
	if innerV, ok := params["inner_v"].(bool); ok && innerV {
		updateBorders["innerVertical"] = border
	}

	request := map[string]interface{}{
		"updateBorders": updateBorders,
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

func autoResize(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	dimension, _ := params["dimension"].(string)
	startIndex, _ := params["start_index"].(float64)
	endIndex, _ := params["end_index"].(float64)

	request := map[string]interface{}{
		"autoResizeDimensions": map[string]interface{}{
			"dimensions": map[string]interface{}{
				"sheetId":    int(sheetID),
				"dimension":  dimension,
				"startIndex": int(startIndex),
				"endIndex":   int(endIndex),
			},
		},
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

// =============================================================================
// Find & Replace
// =============================================================================

func findReplace(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	find, _ := params["find"].(string)
	replacement, _ := params["replacement"].(string)

	findReplace := map[string]interface{}{
		"find":        find,
		"replacement": replacement,
		"allSheets":   true,
	}

	if matchCase, ok := params["match_case"].(bool); ok {
		findReplace["matchCase"] = matchCase
	}
	if matchEntire, ok := params["match_entire"].(bool); ok {
		findReplace["matchEntireCell"] = matchEntire
	}
	if useRegex, ok := params["use_regex"].(bool); ok {
		findReplace["searchByRegex"] = useRegex
	}

	// Limit to specific sheet
	if sheetID, ok := params["sheet_id"].(float64); ok {
		findReplace["allSheets"] = false
		findReplace["sheetId"] = int(sheetID)
	}

	// Limit to specific range
	if rangeStr, ok := params["range"].(string); ok && rangeStr != "" {
		// Parse A1 notation to GridRange (simplified - assumes Sheet1 if not specified)
		// For full implementation, would need to parse the range string
		findReplace["allSheets"] = false
	}

	request := map[string]interface{}{
		"findReplace": findReplace,
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

// =============================================================================
// Protection
// =============================================================================

func protectRange(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)

	protectedRange := map[string]interface{}{
		"range": map[string]interface{}{
			"sheetId": int(sheetID),
		},
	}

	// If row/column specified, protect specific range; otherwise protect entire sheet
	if startRow, ok := params["start_row"].(float64); ok {
		protectedRange["range"].(map[string]interface{})["startRowIndex"] = int(startRow)
	}
	if endRow, ok := params["end_row"].(float64); ok {
		protectedRange["range"].(map[string]interface{})["endRowIndex"] = int(endRow)
	}
	if startColumn, ok := params["start_column"].(float64); ok {
		protectedRange["range"].(map[string]interface{})["startColumnIndex"] = int(startColumn)
	}
	if endColumn, ok := params["end_column"].(float64); ok {
		protectedRange["range"].(map[string]interface{})["endColumnIndex"] = int(endColumn)
	}

	if desc, ok := params["description"].(string); ok {
		protectedRange["description"] = desc
	}

	if warningOnly, ok := params["warning_only"].(bool); ok && warningOnly {
		protectedRange["warningOnly"] = true
	}

	request := map[string]interface{}{
		"addProtectedRange": map[string]interface{}{
			"protectedRange": protectedRange,
		},
	}

	return batchUpdate(ctx, spreadsheetID, []interface{}{request})
}

// =============================================================================
// Helper Functions
// =============================================================================

// batchUpdate sends a batch update request to the Sheets API
func batchUpdate(ctx context.Context, spreadsheetID string, requests []interface{}) (string, error) {
	body := map[string]interface{}{
		"requests": requests,
	}

	endpoint := fmt.Sprintf("%s/spreadsheets/%s:batchUpdate",
		googleSheetsAPIBase, url.PathEscape(spreadsheetID))

	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}
