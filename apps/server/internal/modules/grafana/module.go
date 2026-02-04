package grafana

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/store"
)

const (
	grafanaAPIVersion = "v1"
)

var client = httpclient.New()

// GrafanaModule implements the Module interface for Grafana API
type GrafanaModule struct{}

// New creates a new GrafanaModule instance
func New() *GrafanaModule {
	return &GrafanaModule{}
}

// Module descriptions in multiple languages
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Grafana API - Dashboard, data source, alert, annotation, and folder operations",
	"ja-JP": "Grafana API - ダッシュボード、データソース、アラート、アノテーション、フォルダ操作",
}

// Name returns the module name
func (m *GrafanaModule) Name() string {
	return "grafana"
}

// Descriptions returns the module descriptions in all languages
func (m *GrafanaModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description for a specific language
func (m *GrafanaModule) Description(lang string) string {
	return modules.GetLocalizedText(moduleDescriptions, lang)
}

// APIVersion returns the Grafana API version
func (m *GrafanaModule) APIVersion() string {
	return grafanaAPIVersion
}

// Tools returns all available tools
func (m *GrafanaModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *GrafanaModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for Grafana)
func (m *GrafanaModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *GrafanaModule) ReadResource(ctx context.Context, uri string) (string, error) {
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
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "grafana")
	if err != nil {
		return nil
	}
	return credentials
}

func baseURL(ctx context.Context) string {
	creds := getCredentials(ctx)
	if creds == nil {
		return ""
	}
	base, _ := creds.Metadata["base_url"].(string)
	if base == "" {
		return ""
	}
	return strings.TrimRight(base, "/")
}

func headers(ctx context.Context) map[string]string {
	creds := getCredentials(ctx)
	if creds == nil {
		return map[string]string{}
	}

	h := map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}

	switch creds.AuthType {
	case store.AuthTypeAPIKey:
		h["Authorization"] = "Bearer " + creds.AccessToken
	case store.AuthTypeBasic:
		auth := base64.StdEncoding.EncodeToString([]byte(creds.Username + ":" + creds.Password))
		h["Authorization"] = "Basic " + auth
	}

	return h
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
		ID:   "grafana:search",
		Name: "search",
		Descriptions: modules.LocalizedText{
			"en-US": "Search for dashboards and folders in Grafana.",
			"ja-JP": "Grafanaでダッシュボードとフォルダを検索します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"query":       {Type: "string", Description: "Search query string"},
				"tag":         {Type: "array", Description: "List of tags to filter by", Items: &modules.Property{Type: "string"}},
				"type":        {Type: "string", Description: "Type to search for: dash-folder, dash-db"},
				"folder_uids": {Type: "array", Description: "List of folder UIDs to search within", Items: &modules.Property{Type: "string"}},
				"limit":       {Type: "number", Description: "Maximum results (default: 100)"},
				"page":        {Type: "number", Description: "Page number (default: 1)"},
			},
		},
	},
	{
		ID:   "grafana:get_dashboard",
		Name: "get_dashboard",
		Descriptions: modules.LocalizedText{
			"en-US": "Get a dashboard by its UID, including the full JSON model and metadata.",
			"ja-JP": "UIDでダッシュボードを取得します（完全なJSONモデルとメタデータを含む）。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"uid": {Type: "string", Description: "Dashboard UID"},
			},
			Required: []string{"uid"},
		},
	},
	{
		ID:   "grafana:list_datasources",
		Name: "list_datasources",
		Descriptions: modules.LocalizedText{
			"en-US": "List all configured data sources.",
			"ja-JP": "設定済みの全データソースを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "grafana:get_datasource",
		Name: "get_datasource",
		Descriptions: modules.LocalizedText{
			"en-US": "Get a data source by its UID.",
			"ja-JP": "UIDでデータソースを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"uid": {Type: "string", Description: "Data source UID"},
			},
			Required: []string{"uid"},
		},
	},
	{
		ID:   "grafana:list_alerts",
		Name: "list_alerts",
		Descriptions: modules.LocalizedText{
			"en-US": "List all alert rules.",
			"ja-JP": "すべてのアラートルールを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "grafana:get_alert",
		Name: "get_alert",
		Descriptions: modules.LocalizedText{
			"en-US": "Get an alert rule by its UID.",
			"ja-JP": "UIDでアラートルールを取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"uid": {Type: "string", Description: "Alert rule UID"},
			},
			Required: []string{"uid"},
		},
	},
	{
		ID:   "grafana:query_annotations",
		Name: "query_annotations",
		Descriptions: modules.LocalizedText{
			"en-US": "Query annotations with optional filters for dashboard, panel, time range, or tags.",
			"ja-JP": "ダッシュボード、パネル、時間範囲、タグでフィルタリングしてアノテーションを検索します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"from":          {Type: "number", Description: "Epoch timestamp in milliseconds for start time"},
				"to":            {Type: "number", Description: "Epoch timestamp in milliseconds for end time"},
				"dashboard_uid": {Type: "string", Description: "Dashboard UID to filter by"},
				"panel_id":      {Type: "number", Description: "Panel ID to filter by"},
				"tags":          {Type: "array", Description: "Tags to filter by", Items: &modules.Property{Type: "string"}},
				"type":          {Type: "string", Description: "Annotation type: annotation or alert"},
				"limit":         {Type: "number", Description: "Maximum results (default: 100)"},
			},
		},
	},
	{
		ID:   "grafana:list_folders",
		Name: "list_folders",
		Descriptions: modules.LocalizedText{
			"en-US": "List all folders.",
			"ja-JP": "すべてのフォルダを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"limit": {Type: "number", Description: "Maximum results (default: 1000)"},
				"page":  {Type: "number", Description: "Page number (default: 1)"},
			},
		},
	},
	// =========================================================================
	// Write Tools
	// =========================================================================
	{
		ID:   "grafana:create_update_dashboard",
		Name: "create_update_dashboard",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new dashboard or update an existing one. Provide the full dashboard JSON model.",
			"ja-JP": "新しいダッシュボードを作成するか、既存のダッシュボードを更新します。完全なダッシュボードJSONモデルを指定します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"dashboard":  {Type: "object", Description: "Full dashboard JSON model. Set id to null for new dashboards."},
				"folder_uid": {Type: "string", Description: "UID of the folder to save the dashboard in"},
				"message":    {Type: "string", Description: "Commit message for the change"},
				"overwrite":  {Type: "boolean", Description: "Overwrite existing dashboard with the same title (default: false)"},
			},
			Required: []string{"dashboard"},
		},
	},
	{
		ID:   "grafana:delete_dashboard",
		Name: "delete_dashboard",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a dashboard by its UID.",
			"ja-JP": "UIDでダッシュボードを削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"uid": {Type: "string", Description: "Dashboard UID to delete"},
			},
			Required: []string{"uid"},
		},
	},
	{
		ID:   "grafana:create_annotation",
		Name: "create_annotation",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new annotation on a dashboard.",
			"ja-JP": "ダッシュボードに新しいアノテーションを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"dashboard_uid": {Type: "string", Description: "Dashboard UID to annotate"},
				"panel_id":      {Type: "number", Description: "Panel ID to annotate"},
				"time":          {Type: "number", Description: "Epoch timestamp in milliseconds for annotation start"},
				"time_end":      {Type: "number", Description: "Epoch timestamp in milliseconds for annotation end (for region annotations)"},
				"text":          {Type: "string", Description: "Annotation text"},
				"tags":          {Type: "array", Description: "Tags for the annotation", Items: &modules.Property{Type: "string"}},
			},
			Required: []string{"text"},
		},
	},
	{
		ID:   "grafana:delete_annotation",
		Name: "delete_annotation",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete an annotation by its ID.",
			"ja-JP": "IDでアノテーションを削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"annotation_id": {Type: "number", Description: "Annotation ID to delete"},
			},
			Required: []string{"annotation_id"},
		},
	},
	{
		ID:   "grafana:create_folder",
		Name: "create_folder",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new folder.",
			"ja-JP": "新しいフォルダを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"title": {Type: "string", Description: "Folder title"},
				"uid":   {Type: "string", Description: "Optional custom UID for the folder"},
			},
			Required: []string{"title"},
		},
	},
	{
		ID:   "grafana:delete_folder",
		Name: "delete_folder",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a folder by its UID. This also deletes all dashboards within the folder.",
			"ja-JP": "UIDでフォルダを削除します。フォルダ内のすべてのダッシュボードも削除されます。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"uid": {Type: "string", Description: "Folder UID to delete"},
			},
			Required: []string{"uid"},
		},
	},
	{
		ID:   "grafana:create_alert_rule",
		Name: "create_alert_rule",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new alert rule.",
			"ja-JP": "新しいアラートルールを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"title":          {Type: "string", Description: "Alert rule title"},
				"rule_group":     {Type: "string", Description: "Rule group name"},
				"folder_uid":     {Type: "string", Description: "Folder UID to create the rule in"},
				"condition":      {Type: "string", Description: "Condition reference ID"},
				"data":           {Type: "array", Description: "Array of query/expression objects defining the alert conditions"},
				"no_data_state":  {Type: "string", Description: "State when no data: NoData, Alerting, OK (default: NoData)"},
				"exec_err_state": {Type: "string", Description: "State on execution error: Alerting, Error, OK (default: Alerting)"},
				"for_duration":   {Type: "string", Description: "Duration before alert fires (e.g., '5m', '1h', default: '5m')"},
				"annotations":    {Type: "object", Description: "Annotations map (e.g., summary, description)"},
				"labels":         {Type: "object", Description: "Labels map for routing"},
			},
			Required: []string{"title", "rule_group", "folder_uid", "condition", "data"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

var toolHandlers = map[string]toolHandler{
	// Read
	"search":            search,
	"get_dashboard":     getDashboard,
	"list_datasources":  listDatasources,
	"get_datasource":    getDatasource,
	"list_alerts":       listAlerts,
	"get_alert":         getAlert,
	"query_annotations": queryAnnotations,
	"list_folders":      listFolders,
	// Write
	"create_update_dashboard": createUpdateDashboard,
	"delete_dashboard":        deleteDashboard,
	"create_annotation":       createAnnotation,
	"delete_annotation":       deleteAnnotation,
	"create_folder":           createFolder,
	"delete_folder":           deleteFolder,
	"create_alert_rule":       createAlertRule,
}

// =============================================================================
// Read Handlers
// =============================================================================

func search(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}

	query := url.Values{}
	if q, ok := params["query"].(string); ok && q != "" {
		query.Set("query", q)
	}
	if t, ok := params["type"].(string); ok && t != "" {
		query.Set("type", t)
	}
	if tags, ok := params["tag"].([]interface{}); ok {
		for _, tag := range tags {
			if ts, ok := tag.(string); ok {
				query.Add("tag", ts)
			}
		}
	}
	if uids, ok := params["folder_uids"].([]interface{}); ok {
		for _, uid := range uids {
			if us, ok := uid.(string); ok {
				query.Add("folderUIDs", us)
			}
		}
	}
	if l, ok := params["limit"].(float64); ok {
		query.Set("limit", fmt.Sprintf("%d", int(l)))
	}
	if p, ok := params["page"].(float64); ok {
		query.Set("page", fmt.Sprintf("%d", int(p)))
	}

	endpoint := fmt.Sprintf("%s/api/search?%s", base, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getDashboard(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}
	uid, _ := params["uid"].(string)
	if uid == "" {
		return "", fmt.Errorf("uid is required")
	}

	endpoint := fmt.Sprintf("%s/api/dashboards/uid/%s", base, url.PathEscape(uid))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func listDatasources(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}

	endpoint := fmt.Sprintf("%s/api/datasources", base)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getDatasource(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}
	uid, _ := params["uid"].(string)
	if uid == "" {
		return "", fmt.Errorf("uid is required")
	}

	endpoint := fmt.Sprintf("%s/api/datasources/uid/%s", base, url.PathEscape(uid))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func listAlerts(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}

	endpoint := fmt.Sprintf("%s/api/v1/provisioning/alert-rules", base)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getAlert(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}
	uid, _ := params["uid"].(string)
	if uid == "" {
		return "", fmt.Errorf("uid is required")
	}

	endpoint := fmt.Sprintf("%s/api/v1/provisioning/alert-rules/%s", base, url.PathEscape(uid))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func queryAnnotations(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}

	query := url.Values{}
	if from, ok := params["from"].(float64); ok {
		query.Set("from", fmt.Sprintf("%d", int64(from)))
	}
	if to, ok := params["to"].(float64); ok {
		query.Set("to", fmt.Sprintf("%d", int64(to)))
	}
	if uid, ok := params["dashboard_uid"].(string); ok && uid != "" {
		query.Set("dashboardUID", uid)
	}
	if pid, ok := params["panel_id"].(float64); ok {
		query.Set("panelId", fmt.Sprintf("%d", int(pid)))
	}
	if tags, ok := params["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if ts, ok := tag.(string); ok {
				query.Add("tags", ts)
			}
		}
	}
	if t, ok := params["type"].(string); ok && t != "" {
		query.Set("type", t)
	}
	if l, ok := params["limit"].(float64); ok {
		query.Set("limit", fmt.Sprintf("%d", int(l)))
	}

	endpoint := fmt.Sprintf("%s/api/annotations?%s", base, query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func listFolders(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}

	query := url.Values{}
	if l, ok := params["limit"].(float64); ok {
		query.Set("limit", fmt.Sprintf("%d", int(l)))
	}
	if p, ok := params["page"].(float64); ok {
		query.Set("page", fmt.Sprintf("%d", int(p)))
	}

	endpoint := fmt.Sprintf("%s/api/folders", base)
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Write Handlers
// =============================================================================

func createUpdateDashboard(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}

	dashboard, ok := params["dashboard"]
	if !ok {
		return "", fmt.Errorf("dashboard is required")
	}

	body := map[string]any{
		"dashboard": dashboard,
	}
	if folderUID, ok := params["folder_uid"].(string); ok && folderUID != "" {
		body["folderUid"] = folderUID
	}
	if message, ok := params["message"].(string); ok && message != "" {
		body["message"] = message
	}
	if overwrite, ok := params["overwrite"].(bool); ok {
		body["overwrite"] = overwrite
	}

	endpoint := fmt.Sprintf("%s/api/dashboards/db", base)
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteDashboard(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}
	uid, _ := params["uid"].(string)
	if uid == "" {
		return "", fmt.Errorf("uid is required")
	}

	endpoint := fmt.Sprintf("%s/api/dashboards/uid/%s", base, url.PathEscape(uid))
	respBody, err := client.DoJSON("DELETE", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createAnnotation(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}

	text, _ := params["text"].(string)
	if text == "" {
		return "", fmt.Errorf("text is required")
	}

	body := map[string]any{
		"text": text,
	}
	if uid, ok := params["dashboard_uid"].(string); ok && uid != "" {
		body["dashboardUID"] = uid
	}
	if pid, ok := params["panel_id"].(float64); ok {
		body["panelId"] = int(pid)
	}
	if t, ok := params["time"].(float64); ok {
		body["time"] = int64(t)
	}
	if te, ok := params["time_end"].(float64); ok {
		body["timeEnd"] = int64(te)
	}
	if tags, ok := params["tags"].([]interface{}); ok {
		body["tags"] = tags
	}

	endpoint := fmt.Sprintf("%s/api/annotations", base)
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteAnnotation(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}
	id, ok := params["annotation_id"].(float64)
	if !ok {
		return "", fmt.Errorf("annotation_id is required")
	}

	endpoint := fmt.Sprintf("%s/api/annotations/%d", base, int(id))
	respBody, err := client.DoJSON("DELETE", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createFolder(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}

	title, _ := params["title"].(string)
	if title == "" {
		return "", fmt.Errorf("title is required")
	}

	body := map[string]any{
		"title": title,
	}
	if uid, ok := params["uid"].(string); ok && uid != "" {
		body["uid"] = uid
	}

	endpoint := fmt.Sprintf("%s/api/folders", base)
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteFolder(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}
	uid, _ := params["uid"].(string)
	if uid == "" {
		return "", fmt.Errorf("uid is required")
	}

	endpoint := fmt.Sprintf("%s/api/folders/%s", base, url.PathEscape(uid))
	respBody, err := client.DoJSON("DELETE", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createAlertRule(ctx context.Context, params map[string]any) (string, error) {
	base := baseURL(ctx)
	if base == "" {
		return "", fmt.Errorf("grafana base_url not configured")
	}

	title, _ := params["title"].(string)
	if title == "" {
		return "", fmt.Errorf("title is required")
	}
	ruleGroup, _ := params["rule_group"].(string)
	if ruleGroup == "" {
		return "", fmt.Errorf("rule_group is required")
	}
	folderUID, _ := params["folder_uid"].(string)
	if folderUID == "" {
		return "", fmt.Errorf("folder_uid is required")
	}
	condition, _ := params["condition"].(string)
	if condition == "" {
		return "", fmt.Errorf("condition is required")
	}
	data, ok := params["data"]
	if !ok {
		return "", fmt.Errorf("data is required")
	}

	body := map[string]any{
		"title":     title,
		"ruleGroup": ruleGroup,
		"folderUID": folderUID,
		"condition": condition,
		"data":      data,
	}

	if nds, ok := params["no_data_state"].(string); ok && nds != "" {
		body["noDataState"] = nds
	}
	if ees, ok := params["exec_err_state"].(string); ok && ees != "" {
		body["execErrState"] = ees
	}
	if fd, ok := params["for_duration"].(string); ok && fd != "" {
		body["for"] = fd
	}
	if ann, ok := params["annotations"]; ok {
		body["annotations"] = ann
	}
	if lbl, ok := params["labels"]; ok {
		body["labels"] = lbl
	}

	endpoint := fmt.Sprintf("%s/api/v1/provisioning/alert-rules", base)
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}
