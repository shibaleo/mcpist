package google_docs

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
	"mcpist/server/pkg/googledocsapi"
	gen "mcpist/server/pkg/googledocsapi/gen"
	"mcpist/server/pkg/googledriveapi"
	driveGen "mcpist/server/pkg/googledriveapi/gen"
)

const (
	googleDocsVersion = "v1"
)

var toJSON = modules.ToJSON

// GoogleDocsModule implements the Module interface for Google Docs API
type GoogleDocsModule struct{}

func New() *GoogleDocsModule { return &GoogleDocsModule{} }

var moduleDescriptions = modules.LocalizedText{
	"en-US": "Google Docs API - Read, create, and edit Google Documents",
	"ja-JP": "Google Docs API - Google ドキュメントの読み取り、作成、編集",
}

func (m *GoogleDocsModule) Name() string                        { return "google_docs" }
func (m *GoogleDocsModule) Descriptions() modules.LocalizedText { return moduleDescriptions }
func (m *GoogleDocsModule) Description() string {
	return moduleDescriptions["en-US"]
}
func (m *GoogleDocsModule) APIVersion() string            { return googleDocsVersion }
func (m *GoogleDocsModule) Tools() []modules.Tool         { return toolDefinitions }
func (m *GoogleDocsModule) Resources() []modules.Resource { return nil }
func (m *GoogleDocsModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

func (m *GoogleDocsModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format.
func (m *GoogleDocsModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// =============================================================================
// Token and Client
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[google_docs] No auth context")
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "google_docs")
	if err != nil {
		log.Printf("[google_docs] GetModuleToken error: %v", err)
		return nil
	}
	return credentials
}

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return googledocsapi.NewClient(creds.AccessToken)
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
	{ID: "google_docs:get_document", Name: "get_document", Descriptions: modules.LocalizedText{"en-US": "Get a Google Document's metadata and structure.", "ja-JP": "Google ドキュメントのメタデータと構造を取得します。"}, Annotations: modules.AnnotateReadOnly, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}}, Required: []string{"document_id"}}},
	{ID: "google_docs:read_document", Name: "read_document", Descriptions: modules.LocalizedText{"en-US": "Read a Google Document's content as plain text.", "ja-JP": "Google ドキュメントの内容をプレーンテキストとして読み取ります。"}, Annotations: modules.AnnotateReadOnly, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}}, Required: []string{"document_id"}}},
	{ID: "google_docs:list_tabs", Name: "list_tabs", Descriptions: modules.LocalizedText{"en-US": "List all tabs in a multi-tab document.", "ja-JP": "マルチタブドキュメントの全タブを一覧表示します。"}, Annotations: modules.AnnotateReadOnly, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}}, Required: []string{"document_id"}}},
	{ID: "google_docs:create_document", Name: "create_document", Descriptions: modules.LocalizedText{"en-US": "Create a new Google Document.", "ja-JP": "新しい Google ドキュメントを作成します。"}, Annotations: modules.AnnotateCreate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"title": {Type: "string", Description: "Document title"}}, Required: []string{"title"}}},
	{ID: "google_docs:append_text", Name: "append_text", Descriptions: modules.LocalizedText{"en-US": "Append text to the end of a document.", "ja-JP": "ドキュメントの末尾にテキストを追加します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "text": {Type: "string", Description: "Text to append"}, "tab_id": {Type: "string", Description: "Tab ID for multi-tab documents (optional)"}}, Required: []string{"document_id", "text"}}},
	{ID: "google_docs:insert_text", Name: "insert_text", Descriptions: modules.LocalizedText{"en-US": "Insert text at a specific position in the document.", "ja-JP": "ドキュメントの指定位置にテキストを挿入します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "text": {Type: "string", Description: "Text to insert"}, "index": {Type: "number", Description: "Position index (1-based). Use 1 for document start."}, "tab_id": {Type: "string", Description: "Tab ID for multi-tab documents (optional)"}}, Required: []string{"document_id", "text", "index"}}},
	{ID: "google_docs:delete_range", Name: "delete_range", Descriptions: modules.LocalizedText{"en-US": "Delete content from a specified range in the document.", "ja-JP": "ドキュメントの指定範囲のコンテンツを削除します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "start_index": {Type: "number", Description: "Start position index (1-based)"}, "end_index": {Type: "number", Description: "End position index (1-based, exclusive)"}, "tab_id": {Type: "string", Description: "Tab ID for multi-tab documents (optional)"}}, Required: []string{"document_id", "start_index", "end_index"}}},
	{ID: "google_docs:apply_text_style", Name: "apply_text_style", Descriptions: modules.LocalizedText{"en-US": "Apply text styling (bold, italic, underline, colors) to a range.", "ja-JP": "指定範囲にテキストスタイル（太字、斜体、下線、色）を適用します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "start_index": {Type: "number", Description: "Start position index (1-based)"}, "end_index": {Type: "number", Description: "End position index (1-based, exclusive)"}, "bold": {Type: "boolean", Description: "Apply bold"}, "italic": {Type: "boolean", Description: "Apply italic"}, "underline": {Type: "boolean", Description: "Apply underline"}, "strikethrough": {Type: "boolean", Description: "Apply strikethrough"}, "font_size": {Type: "number", Description: "Font size in points"}, "foreground_color": {Type: "string", Description: "Text color in hex format (e.g., '#FF0000')"}, "background_color": {Type: "string", Description: "Background color in hex format"}, "tab_id": {Type: "string", Description: "Tab ID for multi-tab documents (optional)"}}, Required: []string{"document_id", "start_index", "end_index"}}},
	{ID: "google_docs:apply_paragraph_style", Name: "apply_paragraph_style", Descriptions: modules.LocalizedText{"en-US": "Apply paragraph styling (alignment, spacing, indentation) to a range.", "ja-JP": "指定範囲に段落スタイル（配置、行間、インデント）を適用します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "start_index": {Type: "number", Description: "Start position index (1-based)"}, "end_index": {Type: "number", Description: "End position index (1-based, exclusive)"}, "alignment": {Type: "string", Description: "Alignment: 'START', 'CENTER', 'END', 'JUSTIFIED'"}, "line_spacing": {Type: "number", Description: "Line spacing multiplier (e.g., 1.0, 1.5, 2.0)"}, "indent_start": {Type: "number", Description: "Start indentation in points"}, "indent_end": {Type: "number", Description: "End indentation in points"}, "tab_id": {Type: "string", Description: "Tab ID for multi-tab documents (optional)"}}, Required: []string{"document_id", "start_index", "end_index"}}},
	{ID: "google_docs:insert_table", Name: "insert_table", Descriptions: modules.LocalizedText{"en-US": "Insert a table at a specific position in the document.", "ja-JP": "ドキュメントの指定位置にテーブルを挿入します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "rows": {Type: "number", Description: "Number of rows"}, "columns": {Type: "number", Description: "Number of columns"}, "index": {Type: "number", Description: "Position index (1-based) to insert the table"}, "tab_id": {Type: "string", Description: "Tab ID for multi-tab documents (optional)"}}, Required: []string{"document_id", "rows", "columns", "index"}}},
	{ID: "google_docs:insert_page_break", Name: "insert_page_break", Descriptions: modules.LocalizedText{"en-US": "Insert a page break at a specific position.", "ja-JP": "指定位置に改ページを挿入します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "index": {Type: "number", Description: "Position index (1-based)"}, "tab_id": {Type: "string", Description: "Tab ID for multi-tab documents (optional)"}}, Required: []string{"document_id", "index"}}},
	{ID: "google_docs:insert_image", Name: "insert_image", Descriptions: modules.LocalizedText{"en-US": "Insert an image from a URL at a specific position.", "ja-JP": "URLから画像を指定位置に挿入します。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "image_url": {Type: "string", Description: "Public URL of the image"}, "index": {Type: "number", Description: "Position index (1-based)"}, "width": {Type: "number", Description: "Image width in points (optional)"}, "height": {Type: "number", Description: "Image height in points (optional)"}, "tab_id": {Type: "string", Description: "Tab ID for multi-tab documents (optional)"}}, Required: []string{"document_id", "image_url", "index"}}},
	{ID: "google_docs:list_comments", Name: "list_comments", Descriptions: modules.LocalizedText{"en-US": "List all comments on a document.", "ja-JP": "ドキュメントの全コメントを一覧表示します。"}, Annotations: modules.AnnotateReadOnly, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "page_size": {Type: "number", Description: "Maximum number of comments (1-100). Default: 20"}, "page_token": {Type: "string", Description: "Token for pagination"}}, Required: []string{"document_id"}}},
	{ID: "google_docs:get_comment", Name: "get_comment", Descriptions: modules.LocalizedText{"en-US": "Get a specific comment with its replies.", "ja-JP": "特定のコメントとその返信を取得します。"}, Annotations: modules.AnnotateReadOnly, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "comment_id": {Type: "string", Description: "Comment ID"}}, Required: []string{"document_id", "comment_id"}}},
	{ID: "google_docs:add_comment", Name: "add_comment", Descriptions: modules.LocalizedText{"en-US": "Add a comment anchored to a specific text range.", "ja-JP": "特定のテキスト範囲にアンカーされたコメントを追加します。"}, Annotations: modules.AnnotateCreate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "content": {Type: "string", Description: "Comment content"}, "quoted_text": {Type: "string", Description: "Text to anchor the comment to (optional)"}}, Required: []string{"document_id", "content"}}},
	{ID: "google_docs:reply_to_comment", Name: "reply_to_comment", Descriptions: modules.LocalizedText{"en-US": "Reply to an existing comment.", "ja-JP": "既存のコメントに返信します。"}, Annotations: modules.AnnotateCreate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "comment_id": {Type: "string", Description: "Comment ID to reply to"}, "content": {Type: "string", Description: "Reply content"}}, Required: []string{"document_id", "comment_id", "content"}}},
	{ID: "google_docs:resolve_comment", Name: "resolve_comment", Descriptions: modules.LocalizedText{"en-US": "Mark a comment as resolved.", "ja-JP": "コメントを解決済みとしてマークします。"}, Annotations: modules.AnnotateUpdate, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "comment_id": {Type: "string", Description: "Comment ID to resolve"}}, Required: []string{"document_id", "comment_id"}}},
	{ID: "google_docs:delete_comment", Name: "delete_comment", Descriptions: modules.LocalizedText{"en-US": "Delete a comment from the document.", "ja-JP": "ドキュメントからコメントを削除します。"}, Annotations: modules.AnnotateDelete, InputSchema: modules.InputSchema{Type: "object", Properties: map[string]modules.Property{"document_id": {Type: "string", Description: "Document ID"}, "comment_id": {Type: "string", Description: "Comment ID to delete"}}, Required: []string{"document_id", "comment_id"}}},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"get_document":          getDocument,
	"read_document":         readDocument,
	"list_tabs":             listTabs,
	"create_document":       createDocument,
	"append_text":           appendText,
	"insert_text":           insertText,
	"delete_range":          deleteRange,
	"apply_text_style":      applyTextStyle,
	"apply_paragraph_style": applyParagraphStyle,
	"insert_table":          insertTable,
	"insert_page_break":     insertPageBreak,
	"insert_image":          insertImage,
	"list_comments":         listComments,
	"get_comment":           getComment,
	"add_comment":           addComment,
	"reply_to_comment":      replyToComment,
	"resolve_comment":       resolveComment,
	"delete_comment":        deleteComment,
}

// =============================================================================
// Document Access
// =============================================================================

func getDocument(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	documentID, _ := params["document_id"].(string)

	resp, err := cli.GetDocument(ctx, gen.GetDocumentParams{DocumentId: documentID})
	if err != nil {
		return "", fmt.Errorf("failed to get document: %w", err)
	}
	return toJSON(resp)
}

func readDocument(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	documentID, _ := params["document_id"].(string)

	doc, err := cli.GetDocument(ctx, gen.GetDocumentParams{DocumentId: documentID})
	if err != nil {
		return "", fmt.Errorf("failed to get document: %w", err)
	}

	// Serialize doc to JSON then parse for text extraction
	docJSON, err := toJSON(doc)
	if err != nil {
		return "", fmt.Errorf("failed to serialize document: %w", err)
	}

	var docMap map[string]interface{}
	if err := json.Unmarshal([]byte(docJSON), &docMap); err != nil {
		return "", fmt.Errorf("failed to parse document: %w", err)
	}

	text := extractTextFromDocument(docMap)

	result := map[string]interface{}{
		"document_id": documentID,
		"title":       doc.Title.Value,
		"content":     text,
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}

// extractTextFromDocument extracts plain text from a Google Docs document structure.
func extractTextFromDocument(doc map[string]interface{}) string {
	var text strings.Builder

	body, ok := doc["body"].(map[string]interface{})
	if !ok {
		return ""
	}
	content, ok := body["content"].([]interface{})
	if !ok {
		return ""
	}

	for _, element := range content {
		elem, ok := element.(map[string]interface{})
		if !ok {
			continue
		}
		if para, ok := elem["paragraph"].(map[string]interface{}); ok {
			if elements, ok := para["elements"].([]interface{}); ok {
				for _, e := range elements {
					if textElem, ok := e.(map[string]interface{}); ok {
						if textRun, ok := textElem["textRun"].(map[string]interface{}); ok {
							if c, ok := textRun["content"].(string); ok {
								text.WriteString(c)
							}
						}
					}
				}
			}
		}
		if table, ok := elem["table"].(map[string]interface{}); ok {
			if rows, ok := table["tableRows"].([]interface{}); ok {
				for _, row := range rows {
					if r, ok := row.(map[string]interface{}); ok {
						if cells, ok := r["tableCells"].([]interface{}); ok {
							for _, cell := range cells {
								if c, ok := cell.(map[string]interface{}); ok {
									if cellContent, ok := c["content"].([]interface{}); ok {
										for _, cc := range cellContent {
											if para, ok := cc.(map[string]interface{})["paragraph"].(map[string]interface{}); ok {
												if elements, ok := para["elements"].([]interface{}); ok {
													for _, e := range elements {
														if textElem, ok := e.(map[string]interface{}); ok {
															if textRun, ok := textElem["textRun"].(map[string]interface{}); ok {
																if c, ok := textRun["content"].(string); ok {
																	text.WriteString(c)
																}
															}
														}
													}
												}
											}
										}
									}
								}
							}
							text.WriteString("\t")
						}
					}
					text.WriteString("\n")
				}
			}
		}
	}
	return text.String()
}

func listTabs(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	documentID, _ := params["document_id"].(string)

	doc, err := cli.GetDocument(ctx, gen.GetDocumentParams{DocumentId: documentID})
	if err != nil {
		return "", fmt.Errorf("failed to get document: %w", err)
	}

	if len(doc.Tabs) == 0 {
		result := map[string]interface{}{
			"document_id": documentID,
			"tabs":        []interface{}{},
			"message":     "Document has no tabs (single tab document)",
		}
		b, _ := json.Marshal(result)
		return string(b), nil
	}

	tabsJSON, _ := json.Marshal(doc.Tabs)
	result := fmt.Sprintf(`{"document_id":"%s","tabs":%s}`, documentID, string(tabsJSON))
	return result, nil
}

// =============================================================================
// Document Creation
// =============================================================================

func createDocument(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	title, _ := params["title"].(string)

	resp, err := cli.CreateDocument(ctx, &gen.CreateDocumentRequest{Title: title})
	if err != nil {
		return "", fmt.Errorf("failed to create document: %w", err)
	}
	return toJSON(resp)
}

// =============================================================================
// Content Editing (all via batchUpdate)
// =============================================================================

func batchUpdate(ctx context.Context, documentID string, requests []map[string]interface{}) (string, error) {
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

	resp, err := cli.BatchUpdateDocument(ctx,
		&gen.BatchUpdateRequest{Requests: items},
		gen.BatchUpdateDocumentParams{DocumentId: documentID},
	)
	if err != nil {
		return "", fmt.Errorf("failed to batch update: %w", err)
	}
	return toJSON(resp)
}

func appendText(ctx context.Context, params map[string]any) (string, error) {
	documentID, _ := params["document_id"].(string)
	text, _ := params["text"].(string)

	cli, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	doc, err := cli.GetDocument(ctx, gen.GetDocumentParams{DocumentId: documentID})
	if err != nil {
		return "", fmt.Errorf("failed to get document: %w", err)
	}

	docJSON, _ := toJSON(doc)
	var docMap map[string]interface{}
	json.Unmarshal([]byte(docJSON), &docMap)

	var endIndex int = 1
	if body, ok := docMap["body"].(map[string]interface{}); ok {
		if content, ok := body["content"].([]interface{}); ok && len(content) > 0 {
			lastElem := content[len(content)-1].(map[string]interface{})
			if idx, ok := lastElem["endIndex"].(float64); ok {
				endIndex = int(idx) - 1
			}
		}
	}

	location := map[string]interface{}{"index": endIndex}
	if tabID, ok := params["tab_id"].(string); ok && tabID != "" {
		location["tabId"] = tabID
	}

	return batchUpdate(ctx, documentID, []map[string]interface{}{
		{"insertText": map[string]interface{}{"location": location, "text": text}},
	})
}

func insertText(ctx context.Context, params map[string]any) (string, error) {
	documentID, _ := params["document_id"].(string)
	text, _ := params["text"].(string)
	index := int(params["index"].(float64))

	location := map[string]interface{}{"index": index}
	if tabID, ok := params["tab_id"].(string); ok && tabID != "" {
		location["tabId"] = tabID
	}

	return batchUpdate(ctx, documentID, []map[string]interface{}{
		{"insertText": map[string]interface{}{"location": location, "text": text}},
	})
}

func deleteRange(ctx context.Context, params map[string]any) (string, error) {
	documentID, _ := params["document_id"].(string)
	startIndex := int(params["start_index"].(float64))
	endIndex := int(params["end_index"].(float64))

	rangeSpec := map[string]interface{}{"startIndex": startIndex, "endIndex": endIndex}
	if tabID, ok := params["tab_id"].(string); ok && tabID != "" {
		rangeSpec["tabId"] = tabID
	}

	return batchUpdate(ctx, documentID, []map[string]interface{}{
		{"deleteContentRange": map[string]interface{}{"range": rangeSpec}},
	})
}

// =============================================================================
// Formatting
// =============================================================================

func applyTextStyle(ctx context.Context, params map[string]any) (string, error) {
	documentID, _ := params["document_id"].(string)
	startIndex := int(params["start_index"].(float64))
	endIndex := int(params["end_index"].(float64))

	textStyle := map[string]interface{}{}
	fields := []string{}

	if bold, ok := params["bold"].(bool); ok {
		textStyle["bold"] = bold
		fields = append(fields, "bold")
	}
	if italic, ok := params["italic"].(bool); ok {
		textStyle["italic"] = italic
		fields = append(fields, "italic")
	}
	if underline, ok := params["underline"].(bool); ok {
		textStyle["underline"] = underline
		fields = append(fields, "underline")
	}
	if strikethrough, ok := params["strikethrough"].(bool); ok {
		textStyle["strikethrough"] = strikethrough
		fields = append(fields, "strikethrough")
	}
	if fontSize, ok := params["font_size"].(float64); ok {
		textStyle["fontSize"] = map[string]interface{}{"magnitude": fontSize, "unit": "PT"}
		fields = append(fields, "fontSize")
	}
	if fgColor, ok := params["foreground_color"].(string); ok && fgColor != "" {
		textStyle["foregroundColor"] = parseColor(fgColor)
		fields = append(fields, "foregroundColor")
	}
	if bgColor, ok := params["background_color"].(string); ok && bgColor != "" {
		textStyle["backgroundColor"] = parseColor(bgColor)
		fields = append(fields, "backgroundColor")
	}

	if len(fields) == 0 {
		return "", fmt.Errorf("no style properties specified")
	}

	rangeSpec := map[string]interface{}{"startIndex": startIndex, "endIndex": endIndex}
	if tabID, ok := params["tab_id"].(string); ok && tabID != "" {
		rangeSpec["tabId"] = tabID
	}

	return batchUpdate(ctx, documentID, []map[string]interface{}{
		{"updateTextStyle": map[string]interface{}{"range": rangeSpec, "textStyle": textStyle, "fields": strings.Join(fields, ",")}},
	})
}

func applyParagraphStyle(ctx context.Context, params map[string]any) (string, error) {
	documentID, _ := params["document_id"].(string)
	startIndex := int(params["start_index"].(float64))
	endIndex := int(params["end_index"].(float64))

	paragraphStyle := map[string]interface{}{}
	fields := []string{}

	if alignment, ok := params["alignment"].(string); ok && alignment != "" {
		paragraphStyle["alignment"] = alignment
		fields = append(fields, "alignment")
	}
	if lineSpacing, ok := params["line_spacing"].(float64); ok {
		paragraphStyle["lineSpacing"] = lineSpacing * 100
		fields = append(fields, "lineSpacing")
	}
	if indentStart, ok := params["indent_start"].(float64); ok {
		paragraphStyle["indentStart"] = map[string]interface{}{"magnitude": indentStart, "unit": "PT"}
		fields = append(fields, "indentStart")
	}
	if indentEnd, ok := params["indent_end"].(float64); ok {
		paragraphStyle["indentEnd"] = map[string]interface{}{"magnitude": indentEnd, "unit": "PT"}
		fields = append(fields, "indentEnd")
	}

	if len(fields) == 0 {
		return "", fmt.Errorf("no paragraph style properties specified")
	}

	rangeSpec := map[string]interface{}{"startIndex": startIndex, "endIndex": endIndex}
	if tabID, ok := params["tab_id"].(string); ok && tabID != "" {
		rangeSpec["tabId"] = tabID
	}

	return batchUpdate(ctx, documentID, []map[string]interface{}{
		{"updateParagraphStyle": map[string]interface{}{"range": rangeSpec, "paragraphStyle": paragraphStyle, "fields": strings.Join(fields, ",")}},
	})
}

func parseColor(hex string) map[string]interface{} {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return nil
	}
	r, _ := hexToDec(hex[0:2])
	g, _ := hexToDec(hex[2:4])
	b, _ := hexToDec(hex[4:6])
	return map[string]interface{}{
		"color": map[string]interface{}{
			"rgbColor": map[string]interface{}{
				"red": float64(r) / 255.0, "green": float64(g) / 255.0, "blue": float64(b) / 255.0,
			},
		},
	}
}

func hexToDec(hex string) (int, error) {
	var result int
	for _, c := range hex {
		result *= 16
		switch {
		case c >= '0' && c <= '9':
			result += int(c - '0')
		case c >= 'a' && c <= 'f':
			result += int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			result += int(c-'A') + 10
		}
	}
	return result, nil
}

// =============================================================================
// Structure
// =============================================================================

func insertTable(ctx context.Context, params map[string]any) (string, error) {
	documentID, _ := params["document_id"].(string)
	rows := int(params["rows"].(float64))
	columns := int(params["columns"].(float64))
	index := int(params["index"].(float64))

	location := map[string]interface{}{"index": index}
	if tabID, ok := params["tab_id"].(string); ok && tabID != "" {
		location["tabId"] = tabID
	}

	return batchUpdate(ctx, documentID, []map[string]interface{}{
		{"insertTable": map[string]interface{}{"rows": rows, "columns": columns, "location": location}},
	})
}

func insertPageBreak(ctx context.Context, params map[string]any) (string, error) {
	documentID, _ := params["document_id"].(string)
	index := int(params["index"].(float64))

	location := map[string]interface{}{"index": index}
	if tabID, ok := params["tab_id"].(string); ok && tabID != "" {
		location["tabId"] = tabID
	}

	return batchUpdate(ctx, documentID, []map[string]interface{}{
		{"insertPageBreak": map[string]interface{}{"location": location}},
	})
}

func insertImage(ctx context.Context, params map[string]any) (string, error) {
	documentID, _ := params["document_id"].(string)
	imageURL, _ := params["image_url"].(string)
	index := int(params["index"].(float64))

	location := map[string]interface{}{"index": index}
	if tabID, ok := params["tab_id"].(string); ok && tabID != "" {
		location["tabId"] = tabID
	}

	insertInlineImage := map[string]interface{}{"location": location, "uri": imageURL}
	if width, ok := params["width"].(float64); ok {
		if height, ok := params["height"].(float64); ok {
			insertInlineImage["objectSize"] = map[string]interface{}{
				"width":  map[string]interface{}{"magnitude": width, "unit": "PT"},
				"height": map[string]interface{}{"magnitude": height, "unit": "PT"},
			}
		}
	}

	return batchUpdate(ctx, documentID, []map[string]interface{}{
		{"insertInlineImage": insertInlineImage},
	})
}

// =============================================================================
// Comments (via Drive API)
// =============================================================================

func listComments(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newDriveClient(ctx)
	if err != nil {
		return "", err
	}
	documentID, _ := params["document_id"].(string)

	p := driveGen.ListCommentsParams{
		FileId: documentID,
		Fields: driveGen.NewOptString("comments(id,content,author,createdTime,modifiedTime,resolved,replies)"),
	}

	pageSize := 20
	if ps, ok := params["page_size"].(float64); ok && ps > 0 {
		pageSize = int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
	}
	p.PageSize = driveGen.NewOptInt(pageSize)

	if pt, ok := params["page_token"].(string); ok && pt != "" {
		p.PageToken = driveGen.NewOptString(pt)
	}

	resp, err := cli.ListComments(ctx, p)
	if err != nil {
		return "", fmt.Errorf("failed to list comments: %w", err)
	}
	return toJSON(resp)
}

func getComment(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newDriveClient(ctx)
	if err != nil {
		return "", err
	}
	documentID, _ := params["document_id"].(string)
	commentID, _ := params["comment_id"].(string)

	resp, err := cli.GetComment(ctx, driveGen.GetCommentParams{
		FileId:    documentID,
		CommentId: commentID,
		Fields:    driveGen.NewOptString("id,content,author,createdTime,modifiedTime,resolved,replies"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get comment: %w", err)
	}
	return toJSON(resp)
}

func addComment(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newDriveClient(ctx)
	if err != nil {
		return "", err
	}
	documentID, _ := params["document_id"].(string)
	content, _ := params["content"].(string)

	reqBody := &driveGen.CommentRequest{
		Content: driveGen.NewOptString(content),
	}
	if quotedText, ok := params["quoted_text"].(string); ok && quotedText != "" {
		reqBody.QuotedFileContent = driveGen.NewOptCommentRequestQuotedFileContent(
			driveGen.CommentRequestQuotedFileContent{Value: driveGen.NewOptString(quotedText)},
		)
	}

	resp, err := cli.CreateComment(ctx, reqBody, driveGen.CreateCommentParams{
		FileId: documentID,
		Fields: driveGen.NewOptString("id,content,author,createdTime"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to add comment: %w", err)
	}
	return toJSON(resp)
}

func replyToComment(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newDriveClient(ctx)
	if err != nil {
		return "", err
	}
	documentID, _ := params["document_id"].(string)
	commentID, _ := params["comment_id"].(string)
	content, _ := params["content"].(string)

	resp, err := cli.CreateReply(ctx, &driveGen.ReplyRequest{
		Content: driveGen.NewOptNilString(content),
	}, driveGen.CreateReplyParams{
		FileId:    documentID,
		CommentId: commentID,
		Fields:    driveGen.NewOptString("id,content,author,createdTime"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to reply to comment: %w", err)
	}
	return toJSON(resp)
}

func resolveComment(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newDriveClient(ctx)
	if err != nil {
		return "", err
	}
	documentID, _ := params["document_id"].(string)
	commentID, _ := params["comment_id"].(string)

	resp, err := cli.CreateReply(ctx, &driveGen.ReplyRequest{
		Content: driveGen.NewOptNilString(""),
		Action:  driveGen.NewOptNilString("resolve"),
	}, driveGen.CreateReplyParams{
		FileId:    documentID,
		CommentId: commentID,
		Fields:    driveGen.NewOptString("id,content,author,createdTime"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to resolve comment: %w", err)
	}
	return toJSON(resp)
}

func deleteComment(ctx context.Context, params map[string]any) (string, error) {
	cli, err := newDriveClient(ctx)
	if err != nil {
		return "", err
	}
	documentID, _ := params["document_id"].(string)
	commentID, _ := params["comment_id"].(string)

	err = cli.DeleteComment(ctx, driveGen.DeleteCommentParams{
		FileId:    documentID,
		CommentId: commentID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to delete comment: %w", err)
	}
	return `{"success":true,"message":"Comment deleted successfully"}`, nil
}
