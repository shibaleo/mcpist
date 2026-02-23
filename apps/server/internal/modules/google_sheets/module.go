package google_sheets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/go-faster/jx"

	"mcpist/server/internal/broker"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/pkg/googledriveapi"
	driveGen "mcpist/server/pkg/googledriveapi/gen"
	"mcpist/server/pkg/googlesheetsapi"
	gen "mcpist/server/pkg/googlesheetsapi/gen"
)

const (
	googleSheetsVersion = "v4"
)

var toJSON = modules.ToJSON

// GoogleSheetsModule implements the Module interface for Google Sheets API
type GoogleSheetsModule struct{}

func New() *GoogleSheetsModule { return &GoogleSheetsModule{} }

var moduleDescriptions = modules.LocalizedText{
	"en-US": "Google Sheets API - Read, create, and edit Google Spreadsheets",
	"ja-JP": "Google Sheets API - Google スプレッドシートの読み取り、作成、編集",
}

func (m *GoogleSheetsModule) Name() string                        { return "google_sheets" }
func (m *GoogleSheetsModule) Descriptions() modules.LocalizedText { return moduleDescriptions }
func (m *GoogleSheetsModule) Description() string {
	return moduleDescriptions["en-US"]
}
func (m *GoogleSheetsModule) APIVersion() string            { return googleSheetsVersion }
func (m *GoogleSheetsModule) Tools() []modules.Tool         { return toolDefinitions }
func (m *GoogleSheetsModule) Resources() []modules.Resource { return nil }
func (m *GoogleSheetsModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

func (m *GoogleSheetsModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format.
func (m *GoogleSheetsModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// =============================================================================
// Token and Client
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[google_sheets] No auth context")
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "google_sheets")
	if err != nil {
		log.Printf("[google_sheets] GetModuleToken error: %v", err)
		return nil
	}
	return credentials
}

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return googlesheetsapi.NewClient(creds.AccessToken)
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
	// Spreadsheet Operations
	{ID: "google_sheets:get_spreadsheet", Name: "get_spreadsheet", Descriptions: modules.LocalizedText{"en-US": "Get spreadsheet metadata including title, sheets, and properties.", "ja-JP": "スプレッドシートのメタデータ（タイトル、シート、プロパティ）を取得します。"}, Annotations: modules.AnnotateReadOnly, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}}, Required: []string{"spreadsheet_id"}}},
	{ID: "google_sheets:create_spreadsheet", Name: "create_spreadsheet", Descriptions: modules.LocalizedText{"en-US": "Create a new spreadsheet.", "ja-JP": "新しいスプレッドシートを作成します。"}, Annotations: modules.AnnotateCreate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"title": {Type: "string", Description: "Spreadsheet title"}, "sheet_names": {Type: "array", Description: "Initial sheet names (optional). Default: ['Sheet1']"}}, Required: []string{"title"}}},
	{ID: "google_sheets:search_spreadsheets", Name: "search_spreadsheets", Descriptions: modules.LocalizedText{"en-US": "Search for spreadsheets in Google Drive.", "ja-JP": "Google Drive内のスプレッドシートを検索します。"}, Annotations: modules.AnnotateReadOnly, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"query": {Type: "string", Description: "Search query (searches in file name)"}, "page_size": {Type: "number", Description: "Maximum results (1-100). Default: 20"}}, Required: nil}},
	// Sheet (Tab) Operations
	{ID: "google_sheets:list_sheets", Name: "list_sheets", Descriptions: modules.LocalizedText{"en-US": "List all sheets (tabs) in a spreadsheet.", "ja-JP": "スプレッドシート内のすべてのシート（タブ）を一覧表示します。"}, Annotations: modules.AnnotateReadOnly, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}}, Required: []string{"spreadsheet_id"}}},
	{ID: "google_sheets:create_sheet", Name: "create_sheet", Descriptions: modules.LocalizedText{"en-US": "Add a new sheet (tab) to a spreadsheet.", "ja-JP": "スプレッドシートに新しいシート（タブ）を追加します。"}, Annotations: modules.AnnotateCreate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "title": {Type: "string", Description: "New sheet title"}, "index": {Type: "number", Description: "Position to insert (0-based). Default: append at end"}}, Required: []string{"spreadsheet_id", "title"}}},
	{ID: "google_sheets:delete_sheet", Name: "delete_sheet", Descriptions: modules.LocalizedText{"en-US": "Delete a sheet (tab) from a spreadsheet.", "ja-JP": "スプレッドシートからシート（タブ）を削除します。"}, Annotations: modules.AnnotateDelete, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID to delete"}}, Required: []string{"spreadsheet_id", "sheet_id"}}},
	{ID: "google_sheets:rename_sheet", Name: "rename_sheet", Descriptions: modules.LocalizedText{"en-US": "Rename a sheet (tab) in a spreadsheet.", "ja-JP": "スプレッドシート内のシート（タブ）の名前を変更します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID to rename"}, "title": {Type: "string", Description: "New sheet title"}}, Required: []string{"spreadsheet_id", "sheet_id", "title"}}},
	{ID: "google_sheets:duplicate_sheet", Name: "duplicate_sheet", Descriptions: modules.LocalizedText{"en-US": "Duplicate a sheet within the same spreadsheet.", "ja-JP": "同じスプレッドシート内でシートを複製します。"}, Annotations: modules.AnnotateCreate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID to duplicate"}, "new_title": {Type: "string", Description: "Title for the new sheet"}, "insert_index": {Type: "number", Description: "Position to insert (0-based). Default: after source"}}, Required: []string{"spreadsheet_id", "sheet_id"}}},
	{ID: "google_sheets:copy_sheet_to", Name: "copy_sheet_to", Descriptions: modules.LocalizedText{"en-US": "Copy a sheet to another spreadsheet.", "ja-JP": "シートを別のスプレッドシートにコピーします。"}, Annotations: modules.AnnotateCreate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"source_spreadsheet_id": {Type: "string", Description: "Source spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID to copy"}, "dest_spreadsheet_id": {Type: "string", Description: "Destination spreadsheet ID"}}, Required: []string{"source_spreadsheet_id", "sheet_id", "dest_spreadsheet_id"}}},
	// Data Read
	{ID: "google_sheets:get_values", Name: "get_values", Descriptions: modules.LocalizedText{"en-US": "Get cell values from a range (e.g., 'Sheet1!A1:C10').", "ja-JP": "指定範囲のセル値を取得します（例: 'Sheet1!A1:C10'）。"}, Annotations: modules.AnnotateReadOnly, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "range": {Type: "string", Description: "A1 notation range (e.g., 'Sheet1!A1:C10', 'A1:C10')"}, "value_render": {Type: "string", Description: "How values should be rendered: 'FORMATTED_VALUE' (default), 'UNFORMATTED_VALUE', or 'FORMULA'"}, "date_time_render": {Type: "string", Description: "How dates should be rendered: 'SERIAL_NUMBER' or 'FORMATTED_STRING' (default)"}}, Required: []string{"spreadsheet_id", "range"}}},
	{ID: "google_sheets:batch_get_values", Name: "batch_get_values", Descriptions: modules.LocalizedText{"en-US": "Get cell values from multiple ranges at once.", "ja-JP": "複数の範囲からセル値を一度に取得します。"}, Annotations: modules.AnnotateReadOnly, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "ranges": {Type: "array", Description: "Array of A1 notation ranges"}, "value_render": {Type: "string", Description: "How values should be rendered"}, "date_time_render": {Type: "string", Description: "How dates should be rendered"}}, Required: []string{"spreadsheet_id", "ranges"}}},
	{ID: "google_sheets:get_formulas", Name: "get_formulas", Descriptions: modules.LocalizedText{"en-US": "Get formulas from a range.", "ja-JP": "指定範囲の数式を取得します。"}, Annotations: modules.AnnotateReadOnly, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "range": {Type: "string", Description: "A1 notation range"}}, Required: []string{"spreadsheet_id", "range"}}},
	// Data Write
	{ID: "google_sheets:update_values", Name: "update_values", Descriptions: modules.LocalizedText{"en-US": "Update cell values in a range.", "ja-JP": "指定範囲のセル値を更新します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "range": {Type: "string", Description: "A1 notation range (e.g., 'Sheet1!A1:C3')"}, "values": {Type: "array", Description: "2D array of values [[row1], [row2], ...]"}, "value_input": {Type: "string", Description: "How input should be interpreted: 'RAW' or 'USER_ENTERED' (default)"}}, Required: []string{"spreadsheet_id", "range", "values"}}},
	{ID: "google_sheets:batch_update_values", Name: "batch_update_values", Descriptions: modules.LocalizedText{"en-US": "Update cell values in multiple ranges at once.", "ja-JP": "複数の範囲のセル値を一度に更新します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "data": {Type: "array", Description: "Array of {range, values} objects. Example: [{\"range\": \"A1:B2\", \"values\": [[1,2],[3,4]]}]"}, "value_input": {Type: "string", Description: "How input should be interpreted: 'RAW' or 'USER_ENTERED' (default)"}}, Required: []string{"spreadsheet_id", "data"}}},
	{ID: "google_sheets:append_values", Name: "append_values", Descriptions: modules.LocalizedText{"en-US": "Append rows to a table (finds the last row and appends data).", "ja-JP": "テーブルに行を追加します（最後の行を見つけてデータを追加）。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "range": {Type: "string", Description: "A1 notation range to search for table (e.g., 'Sheet1!A:C')"}, "values": {Type: "array", Description: "2D array of values to append [[row1], [row2], ...]"}, "value_input": {Type: "string", Description: "How input should be interpreted: 'RAW' or 'USER_ENTERED' (default)"}, "insert_data": {Type: "string", Description: "How to insert: 'OVERWRITE' or 'INSERT_ROWS' (default)"}}, Required: []string{"spreadsheet_id", "range", "values"}}},
	{ID: "google_sheets:clear_values", Name: "clear_values", Descriptions: modules.LocalizedText{"en-US": "Clear cell contents in a range (keeps formatting).", "ja-JP": "指定範囲のセル内容をクリアします（書式は保持）。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "range": {Type: "string", Description: "A1 notation range to clear"}}, Required: []string{"spreadsheet_id", "range"}}},
	// Row/Column Operations
	{ID: "google_sheets:insert_rows", Name: "insert_rows", Descriptions: modules.LocalizedText{"en-US": "Insert empty rows at a specific position.", "ja-JP": "指定位置に空の行を挿入します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID"}, "start_index": {Type: "number", Description: "Row index to start inserting (0-based)"}, "num_rows": {Type: "number", Description: "Number of rows to insert"}}, Required: []string{"spreadsheet_id", "sheet_id", "start_index", "num_rows"}}},
	{ID: "google_sheets:delete_rows", Name: "delete_rows", Descriptions: modules.LocalizedText{"en-US": "Delete rows from a sheet.", "ja-JP": "シートから行を削除します。"}, Annotations: modules.AnnotateDelete, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID"}, "start_index": {Type: "number", Description: "Starting row index (0-based)"}, "end_index": {Type: "number", Description: "Ending row index (exclusive)"}}, Required: []string{"spreadsheet_id", "sheet_id", "start_index", "end_index"}}},
	{ID: "google_sheets:insert_columns", Name: "insert_columns", Descriptions: modules.LocalizedText{"en-US": "Insert empty columns at a specific position.", "ja-JP": "指定位置に空の列を挿入します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID"}, "start_index": {Type: "number", Description: "Column index to start inserting (0-based)"}, "num_columns": {Type: "number", Description: "Number of columns to insert"}}, Required: []string{"spreadsheet_id", "sheet_id", "start_index", "num_columns"}}},
	{ID: "google_sheets:delete_columns", Name: "delete_columns", Descriptions: modules.LocalizedText{"en-US": "Delete columns from a sheet.", "ja-JP": "シートから列を削除します。"}, Annotations: modules.AnnotateDelete, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID"}, "start_index": {Type: "number", Description: "Starting column index (0-based)"}, "end_index": {Type: "number", Description: "Ending column index (exclusive)"}}, Required: []string{"spreadsheet_id", "sheet_id", "start_index", "end_index"}}},
	// Formatting
	{ID: "google_sheets:format_cells", Name: "format_cells", Descriptions: modules.LocalizedText{"en-US": "Format cells (background color, text format, alignment, number format).", "ja-JP": "セルの書式を設定します（背景色、テキスト書式、配置、数値形式）。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID"}, "start_row": {Type: "number", Description: "Start row index (0-based)"}, "end_row": {Type: "number", Description: "End row index (exclusive)"}, "start_column": {Type: "number", Description: "Start column index (0-based)"}, "end_column": {Type: "number", Description: "End column index (exclusive)"}, "background_color": {Type: "object", Description: "Background color {red, green, blue, alpha} (0-1 floats)"}, "bold": {Type: "boolean", Description: "Make text bold"}, "italic": {Type: "boolean", Description: "Make text italic"}, "font_size": {Type: "number", Description: "Font size in points"}, "font_color": {Type: "object", Description: "Font color {red, green, blue, alpha} (0-1 floats)"}, "h_align": {Type: "string", Description: "Horizontal alignment: 'LEFT', 'CENTER', 'RIGHT'"}, "v_align": {Type: "string", Description: "Vertical alignment: 'TOP', 'MIDDLE', 'BOTTOM'"}, "number_format": {Type: "string", Description: "Number format pattern (e.g., '#,##0.00', '0%', 'yyyy-mm-dd')"}, "wrap_strategy": {Type: "string", Description: "Text wrap: 'OVERFLOW_CELL', 'LEGACY_WRAP', 'CLIP', 'WRAP'"}}, Required: []string{"spreadsheet_id", "sheet_id", "start_row", "end_row", "start_column", "end_column"}}},
	{ID: "google_sheets:merge_cells", Name: "merge_cells", Descriptions: modules.LocalizedText{"en-US": "Merge cells in a range.", "ja-JP": "指定範囲のセルを結合します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID"}, "start_row": {Type: "number", Description: "Start row index (0-based)"}, "end_row": {Type: "number", Description: "End row index (exclusive)"}, "start_column": {Type: "number", Description: "Start column index (0-based)"}, "end_column": {Type: "number", Description: "End column index (exclusive)"}, "merge_type": {Type: "string", Description: "Merge type: 'MERGE_ALL' (default), 'MERGE_COLUMNS', 'MERGE_ROWS'"}}, Required: []string{"spreadsheet_id", "sheet_id", "start_row", "end_row", "start_column", "end_column"}}},
	{ID: "google_sheets:unmerge_cells", Name: "unmerge_cells", Descriptions: modules.LocalizedText{"en-US": "Unmerge previously merged cells.", "ja-JP": "結合されたセルを解除します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID"}, "start_row": {Type: "number", Description: "Start row index (0-based)"}, "end_row": {Type: "number", Description: "End row index (exclusive)"}, "start_column": {Type: "number", Description: "Start column index (0-based)"}, "end_column": {Type: "number", Description: "End column index (exclusive)"}}, Required: []string{"spreadsheet_id", "sheet_id", "start_row", "end_row", "start_column", "end_column"}}},
	{ID: "google_sheets:set_borders", Name: "set_borders", Descriptions: modules.LocalizedText{"en-US": "Set borders for a range of cells.", "ja-JP": "セル範囲に罫線を設定します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID"}, "start_row": {Type: "number", Description: "Start row index (0-based)"}, "end_row": {Type: "number", Description: "End row index (exclusive)"}, "start_column": {Type: "number", Description: "Start column index (0-based)"}, "end_column": {Type: "number", Description: "End column index (exclusive)"}, "style": {Type: "string", Description: "Border style: 'SOLID', 'SOLID_MEDIUM', 'SOLID_THICK', 'DASHED', 'DOTTED', 'DOUBLE'"}, "color": {Type: "object", Description: "Border color {red, green, blue, alpha} (0-1 floats)"}, "top": {Type: "boolean", Description: "Apply to top border"}, "bottom": {Type: "boolean", Description: "Apply to bottom border"}, "left": {Type: "boolean", Description: "Apply to left border"}, "right": {Type: "boolean", Description: "Apply to right border"}, "inner_h": {Type: "boolean", Description: "Apply to inner horizontal borders"}, "inner_v": {Type: "boolean", Description: "Apply to inner vertical borders"}}, Required: []string{"spreadsheet_id", "sheet_id", "start_row", "end_row", "start_column", "end_column"}}},
	{ID: "google_sheets:auto_resize", Name: "auto_resize", Descriptions: modules.LocalizedText{"en-US": "Auto-resize columns or rows to fit content.", "ja-JP": "列または行のサイズをコンテンツに合わせて自動調整します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID"}, "dimension": {Type: "string", Description: "Dimension: 'ROWS' or 'COLUMNS'"}, "start_index": {Type: "number", Description: "Start index (0-based)"}, "end_index": {Type: "number", Description: "End index (exclusive)"}}, Required: []string{"spreadsheet_id", "sheet_id", "dimension", "start_index", "end_index"}}},
	// Find & Replace
	{ID: "google_sheets:find_replace", Name: "find_replace", Descriptions: modules.LocalizedText{"en-US": "Find and replace text in a spreadsheet.", "ja-JP": "スプレッドシート内のテキストを検索・置換します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "find": {Type: "string", Description: "Text to find"}, "replacement": {Type: "string", Description: "Replacement text"}, "match_case": {Type: "boolean", Description: "Match case. Default: false"}, "match_entire": {Type: "boolean", Description: "Match entire cell content. Default: false"}, "use_regex": {Type: "boolean", Description: "Use regular expressions. Default: false"}, "sheet_id": {Type: "number", Description: "Limit to specific sheet (optional)"}, "range": {Type: "string", Description: "Limit to specific range in A1 notation (optional)"}}, Required: []string{"spreadsheet_id", "find", "replacement"}}},
	// Protection
	{ID: "google_sheets:protect_range", Name: "protect_range", Descriptions: modules.LocalizedText{"en-US": "Protect a range or sheet from editing.", "ja-JP": "範囲またはシートを編集から保護します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"}, "sheet_id": {Type: "number", Description: "Sheet ID"}, "description": {Type: "string", Description: "Description of the protected range"}, "start_row": {Type: "number", Description: "Start row index (0-based). Omit to protect entire sheet"}, "end_row": {Type: "number", Description: "End row index (exclusive)"}, "start_column": {Type: "number", Description: "Start column index (0-based)"}, "end_column": {Type: "number", Description: "End column index (exclusive)"}, "warning_only": {Type: "boolean", Description: "Show warning instead of blocking. Default: false"}}, Required: []string{"spreadsheet_id", "sheet_id"}}},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"get_spreadsheet":     getSpreadsheet,
	"create_spreadsheet":  createSpreadsheet,
	"search_spreadsheets": searchSpreadsheets,
	"list_sheets":         listSheets,
	"create_sheet":        createSheet,
	"delete_sheet":        deleteSheet,
	"rename_sheet":        renameSheet,
	"duplicate_sheet":     duplicateSheet,
	"copy_sheet_to":       copySheetTo,
	"get_values":          getValues,
	"batch_get_values":    batchGetValues,
	"get_formulas":        getFormulas,
	"update_values":       updateValues,
	"batch_update_values": batchUpdateValues,
	"append_values":       appendValues,
	"clear_values":        clearValues,
	"insert_rows":         insertRows,
	"delete_rows":         deleteRows,
	"insert_columns":      insertColumns,
	"delete_columns":      deleteColumns,
	"format_cells":        formatCells,
	"merge_cells":         mergeCells,
	"unmerge_cells":       unmergeCells,
	"set_borders":         setBorders,
	"auto_resize":         autoResize,
	"find_replace":        findReplace,
	"protect_range":       protectRange,
}

// =============================================================================
// Spreadsheet Operations
// =============================================================================

func getSpreadsheet(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	spreadsheetID, _ := params["spreadsheet_id"].(string)

	resp, err := cli.GetSpreadsheet(ctx, gen.GetSpreadsheetParams{
		SpreadsheetId: spreadsheetID,
		Fields:        gen.NewOptString("spreadsheetId,properties,sheets.properties"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get spreadsheet: %w", err)
	}
	return toJSON(resp)
}

func createSpreadsheet(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	title, _ := params["title"].(string)

	props := gen.CreateSpreadsheetRequestProperties{}
	raw, _ := json.Marshal(title)
	props["title"] = jx.Raw(raw)

	req := &gen.CreateSpreadsheetRequest{
		Properties: gen.NewOptNilCreateSpreadsheetRequestProperties(props),
	}

	if sheetNames, ok := params["sheet_names"].([]interface{}); ok && len(sheetNames) > 0 {
		sheets := make([]gen.CreateSpreadsheetRequestSheetsItem, len(sheetNames))
		for i, name := range sheetNames {
			item := gen.CreateSpreadsheetRequestSheetsItem{}
			propRaw, _ := json.Marshal(map[string]interface{}{"title": name})
			item["properties"] = jx.Raw(propRaw)
			sheets[i] = item
		}
		req.Sheets = gen.NewOptNilCreateSpreadsheetRequestSheetsItemArray(sheets)
	}

	resp, err := cli.CreateSpreadsheet(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create spreadsheet: %w", err)
	}
	return toJSON(resp)
}

func searchSpreadsheets(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newDriveClient(ctx)
	if err != nil {
		return "", err
	}

	q := "mimeType='application/vnd.google-apps.spreadsheet'"
	if query, ok := params["query"].(string); ok && query != "" {
		q = fmt.Sprintf("mimeType='application/vnd.google-apps.spreadsheet' and name contains '%s'", query)
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
		return "", fmt.Errorf("failed to search spreadsheets: %w", err)
	}
	return toJSON(resp)
}

// =============================================================================
// Sheet (Tab) Operations
// =============================================================================

func listSheets(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	spreadsheetID, _ := params["spreadsheet_id"].(string)

	resp, err := cli.GetSpreadsheet(ctx, gen.GetSpreadsheetParams{
		SpreadsheetId: spreadsheetID,
		Fields:        gen.NewOptString("sheets.properties"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list sheets: %w", err)
	}
	return toJSON(resp)
}

func createSheet(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	title, _ := params["title"].(string)

	props := map[string]interface{}{"title": title}
	if idx, ok := params["index"].(float64); ok {
		props["index"] = int(idx)
	}

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"addSheet": map[string]interface{}{"properties": props}},
	})
}

func deleteSheet(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"deleteSheet": map[string]interface{}{"sheetId": int(sheetID)}},
	})
}

func renameSheet(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	title, _ := params["title"].(string)

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"updateSheetProperties": map[string]interface{}{
			"properties": map[string]interface{}{"sheetId": int(sheetID), "title": title},
			"fields":     "title",
		}},
	})
}

func duplicateSheet(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)

	dup := map[string]interface{}{"sourceSheetId": int(sheetID)}
	if newTitle, ok := params["new_title"].(string); ok && newTitle != "" {
		dup["newSheetName"] = newTitle
	}
	if insertIndex, ok := params["insert_index"].(float64); ok {
		dup["insertSheetIndex"] = int(insertIndex)
	}

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"duplicateSheet": dup},
	})
}

func copySheetTo(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	sourceSpreadsheetID, _ := params["source_spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	destSpreadsheetID, _ := params["dest_spreadsheet_id"].(string)

	resp, err := cli.CopySheetTo(ctx, &gen.CopySheetToRequest{
		DestinationSpreadsheetId: gen.NewOptString(destSpreadsheetID),
	}, gen.CopySheetToParams{
		SpreadsheetId: sourceSpreadsheetID,
		SheetId:       int(sheetID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to copy sheet: %w", err)
	}
	return toJSON(resp)
}

// =============================================================================
// Data Operations - Read
// =============================================================================

func getValues(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	rangeStr, _ := params["range"].(string)

	p := gen.GetValuesParams{
		SpreadsheetId: spreadsheetID,
		Range:         rangeStr,
	}
	if vr, ok := params["value_render"].(string); ok && vr != "" {
		p.ValueRenderOption = gen.NewOptString(vr)
	}
	if dtr, ok := params["date_time_render"].(string); ok && dtr != "" {
		p.DateTimeRenderOption = gen.NewOptString(dtr)
	}

	resp, err := cli.GetValues(ctx, p)
	if err != nil {
		return "", fmt.Errorf("failed to get values: %w", err)
	}
	return toJSON(resp)
}

func batchGetValues(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	ranges, _ := params["ranges"].([]interface{})

	var rangeStrs []string
	for _, r := range ranges {
		if s, ok := r.(string); ok {
			rangeStrs = append(rangeStrs, s)
		}
	}

	p := gen.BatchGetValuesParams{
		SpreadsheetId: spreadsheetID,
		Ranges:        rangeStrs,
	}
	if vr, ok := params["value_render"].(string); ok && vr != "" {
		p.ValueRenderOption = gen.NewOptString(vr)
	}
	if dtr, ok := params["date_time_render"].(string); ok && dtr != "" {
		p.DateTimeRenderOption = gen.NewOptString(dtr)
	}

	resp, err := cli.BatchGetValues(ctx, p)
	if err != nil {
		return "", fmt.Errorf("failed to batch get values: %w", err)
	}
	return toJSON(resp)
}

func getFormulas(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	rangeStr, _ := params["range"].(string)

	resp, err := cli.GetValues(ctx, gen.GetValuesParams{
		SpreadsheetId:     spreadsheetID,
		Range:             rangeStr,
		ValueRenderOption: gen.NewOptString("FORMULA"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get formulas: %w", err)
	}
	return toJSON(resp)
}

// =============================================================================
// Data Operations - Write
// =============================================================================

func updateValues(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	rangeStr, _ := params["range"].(string)
	values, _ := params["values"].([]interface{})

	valueInput := "USER_ENTERED"
	if vi, ok := params["value_input"].(string); ok && vi != "" {
		valueInput = vi
	}

	resp, err := cli.UpdateValues(ctx, &gen.ValueRange{
		Values: toRaw2D(values),
	}, gen.UpdateValuesParams{
		SpreadsheetId:    spreadsheetID,
		Range:            rangeStr,
		ValueInputOption: valueInput,
	})
	if err != nil {
		return "", fmt.Errorf("failed to update values: %w", err)
	}
	return toJSON(resp)
}

func batchUpdateValues(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	data, _ := params["data"].([]interface{})

	valueInput := "USER_ENTERED"
	if vi, ok := params["value_input"].(string); ok && vi != "" {
		valueInput = vi
	}

	var vrs []gen.ValueRange
	for _, item := range data {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		vr := gen.ValueRange{}
		if r, ok := m["range"].(string); ok {
			vr.Range = gen.NewOptNilString(r)
		}
		if vals, ok := m["values"].([]interface{}); ok {
			vr.Values = toRaw2D(vals)
		}
		vrs = append(vrs, vr)
	}

	resp, err := cli.BatchUpdateValues(ctx, &gen.BatchUpdateValuesRequest{
		ValueInputOption: gen.NewOptString(valueInput),
		Data:             vrs,
	}, gen.BatchUpdateValuesParams{
		SpreadsheetId: spreadsheetID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to batch update values: %w", err)
	}
	return toJSON(resp)
}

func appendValues(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	rangeStr, _ := params["range"].(string)
	values, _ := params["values"].([]interface{})

	valueInput := "USER_ENTERED"
	if vi, ok := params["value_input"].(string); ok && vi != "" {
		valueInput = vi
	}

	p := gen.AppendValuesParams{
		SpreadsheetId:    spreadsheetID,
		Range:            rangeStr,
		ValueInputOption: valueInput,
	}
	if id, ok := params["insert_data"].(string); ok && id != "" {
		p.InsertDataOption = gen.NewOptString(id)
	} else {
		p.InsertDataOption = gen.NewOptString("INSERT_ROWS")
	}

	resp, err := cli.AppendValues(ctx, &gen.ValueRange{
		Values: toRaw2D(values),
	}, p)
	if err != nil {
		return "", fmt.Errorf("failed to append values: %w", err)
	}
	return toJSON(resp)
}

func clearValues(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	rangeStr, _ := params["range"].(string)

	resp, err := cli.ClearValues(ctx, &gen.ClearValuesReq{}, gen.ClearValuesParams{
		SpreadsheetId: spreadsheetID,
		Range:         rangeStr,
	})
	if err != nil {
		return "", fmt.Errorf("failed to clear values: %w", err)
	}
	return toJSON(resp)
}

// =============================================================================
// Row/Column Operations
// =============================================================================

func insertRows(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	startIndex, _ := params["start_index"].(float64)
	numRows, _ := params["num_rows"].(float64)

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"insertDimension": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId": int(sheetID), "dimension": "ROWS",
				"startIndex": int(startIndex), "endIndex": int(startIndex) + int(numRows),
			},
			"inheritFromBefore": startIndex > 0,
		}},
	})
}

func deleteRows(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	startIndex, _ := params["start_index"].(float64)
	endIndex, _ := params["end_index"].(float64)

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"deleteDimension": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId": int(sheetID), "dimension": "ROWS",
				"startIndex": int(startIndex), "endIndex": int(endIndex),
			},
		}},
	})
}

func insertColumns(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	startIndex, _ := params["start_index"].(float64)
	numColumns, _ := params["num_columns"].(float64)

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"insertDimension": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId": int(sheetID), "dimension": "COLUMNS",
				"startIndex": int(startIndex), "endIndex": int(startIndex) + int(numColumns),
			},
			"inheritFromBefore": startIndex > 0,
		}},
	})
}

func deleteColumns(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	startIndex, _ := params["start_index"].(float64)
	endIndex, _ := params["end_index"].(float64)

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"deleteDimension": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId": int(sheetID), "dimension": "COLUMNS",
				"startIndex": int(startIndex), "endIndex": int(endIndex),
			},
		}},
	})
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

	if bgColor, ok := params["background_color"].(map[string]interface{}); ok {
		cellFormat["backgroundColor"] = bgColor
		fields = append(fields, "userEnteredFormat.backgroundColor")
	}

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

	if hAlign, ok := params["h_align"].(string); ok {
		cellFormat["horizontalAlignment"] = hAlign
		fields = append(fields, "userEnteredFormat.horizontalAlignment")
	}
	if vAlign, ok := params["v_align"].(string); ok {
		cellFormat["verticalAlignment"] = vAlign
		fields = append(fields, "userEnteredFormat.verticalAlignment")
	}

	if numFormat, ok := params["number_format"].(string); ok {
		cellFormat["numberFormat"] = map[string]interface{}{"type": "NUMBER", "pattern": numFormat}
		fields = append(fields, "userEnteredFormat.numberFormat")
	}
	if wrapStrategy, ok := params["wrap_strategy"].(string); ok {
		cellFormat["wrapStrategy"] = wrapStrategy
		fields = append(fields, "userEnteredFormat.wrapStrategy")
	}

	if len(fields) == 0 {
		return "", fmt.Errorf("no formatting options specified")
	}

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"repeatCell": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId": int(sheetID), "startRowIndex": int(startRow), "endRowIndex": int(endRow),
				"startColumnIndex": int(startColumn), "endColumnIndex": int(endColumn),
			},
			"cell":   map[string]interface{}{"userEnteredFormat": cellFormat},
			"fields": strings.Join(fields, ","),
		}},
	})
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

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"mergeCells": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId": int(sheetID), "startRowIndex": int(startRow), "endRowIndex": int(endRow),
				"startColumnIndex": int(startColumn), "endColumnIndex": int(endColumn),
			},
			"mergeType": mergeType,
		}},
	})
}

func unmergeCells(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	startRow, _ := params["start_row"].(float64)
	endRow, _ := params["end_row"].(float64)
	startColumn, _ := params["start_column"].(float64)
	endColumn, _ := params["end_column"].(float64)

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"unmergeCells": map[string]interface{}{
			"range": map[string]interface{}{
				"sheetId": int(sheetID), "startRowIndex": int(startRow), "endRowIndex": int(endRow),
				"startColumnIndex": int(startColumn), "endColumnIndex": int(endColumn),
			},
		}},
	})
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

	border := map[string]interface{}{"style": style, "color": color}

	updateBorders := map[string]interface{}{
		"range": map[string]interface{}{
			"sheetId": int(sheetID), "startRowIndex": int(startRow), "endRowIndex": int(endRow),
			"startColumnIndex": int(startColumn), "endColumnIndex": int(endColumn),
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

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"updateBorders": updateBorders},
	})
}

func autoResize(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)
	dimension, _ := params["dimension"].(string)
	startIndex, _ := params["start_index"].(float64)
	endIndex, _ := params["end_index"].(float64)

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"autoResizeDimensions": map[string]interface{}{
			"dimensions": map[string]interface{}{
				"sheetId": int(sheetID), "dimension": dimension,
				"startIndex": int(startIndex), "endIndex": int(endIndex),
			},
		}},
	})
}

// =============================================================================
// Find & Replace
// =============================================================================

func findReplace(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	find, _ := params["find"].(string)
	replacement, _ := params["replacement"].(string)

	fr := map[string]interface{}{
		"find":        find,
		"replacement": replacement,
		"allSheets":   true,
	}

	if matchCase, ok := params["match_case"].(bool); ok {
		fr["matchCase"] = matchCase
	}
	if matchEntire, ok := params["match_entire"].(bool); ok {
		fr["matchEntireCell"] = matchEntire
	}
	if useRegex, ok := params["use_regex"].(bool); ok {
		fr["searchByRegex"] = useRegex
	}
	if sheetID, ok := params["sheet_id"].(float64); ok {
		fr["allSheets"] = false
		fr["sheetId"] = int(sheetID)
	}
	if rangeStr, ok := params["range"].(string); ok && rangeStr != "" {
		fr["allSheets"] = false
	}

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"findReplace": fr},
	})
}

// =============================================================================
// Protection
// =============================================================================

func protectRange(ctx context.Context, params map[string]any) (string, error) {
	spreadsheetID, _ := params["spreadsheet_id"].(string)
	sheetID, _ := params["sheet_id"].(float64)

	protectedRange := map[string]interface{}{
		"range": map[string]interface{}{"sheetId": int(sheetID)},
	}

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

	return sheetsBatchUpdate(ctx, spreadsheetID, []map[string]interface{}{
		{"addProtectedRange": map[string]interface{}{"protectedRange": protectedRange}},
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

// sheetsBatchUpdate sends a batch update request to the Sheets API via ogen.
func sheetsBatchUpdate(ctx context.Context, spreadsheetID string, requests []map[string]interface{}) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	var items []gen.BatchUpdateRequestRequestsItem
	for _, req := range requests {
		item := gen.BatchUpdateRequestRequestsItem{}
		for k, v := range req {
			raw, err := json.Marshal(v)
			if err != nil {
				continue
			}
			item[k] = jx.Raw(raw)
		}
		items = append(items, item)
	}

	resp, err := cli.BatchUpdate(ctx,
		&gen.BatchUpdateRequest{Requests: items},
		gen.BatchUpdateParams{SpreadsheetId: spreadsheetID},
	)
	if err != nil {
		return "", fmt.Errorf("failed to batch update: %w", err)
	}
	return toJSON(resp)
}

// toRaw2D converts []interface{} (2D array from JSON params) to OptNilAnyArrayArray for ogen ValueRange.
func toRaw2D(values []interface{}) gen.OptNilAnyArrayArray {
	if values == nil {
		return gen.OptNilAnyArrayArray{}
	}
	rows := make([][]jx.Raw, len(values))
	for i, row := range values {
		rowArr, ok := row.([]interface{})
		if !ok {
			continue
		}
		cells := make([]jx.Raw, len(rowArr))
		for j, cell := range rowArr {
			raw, err := json.Marshal(cell)
			if err != nil {
				cells[j] = jx.Raw(`null`)
			} else {
				cells[j] = jx.Raw(raw)
			}
		}
		rows[i] = cells
	}
	return gen.NewOptNilAnyArrayArray(rows)
}
