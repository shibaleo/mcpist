package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/broker"
	"mcpist/server/pkg/grafanaapi"
	gen "mcpist/server/pkg/grafanaapi/gen"

	"github.com/go-faster/jx"
)

// GrafanaModule implements the Module interface for Grafana API
type GrafanaModule struct{}

// New creates a new GrafanaModule instance
func New() *GrafanaModule {
	return &GrafanaModule{}
}

// Module descriptions
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

// Description returns the module description (English)
func (m *GrafanaModule) Description() string {
	return moduleDescriptions["en-US"]
}

// APIVersion returns the Grafana API version
func (m *GrafanaModule) APIVersion() string {
	return "v1"
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

// ToCompact converts JSON result to compact format (MD or CSV)
// Implements modules.CompactConverter interface
func (m *GrafanaModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
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
// ogen client helper
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "grafana")
	if err != nil {
		return nil
	}
	return credentials
}

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}

	base, _ := creds.Metadata["base_url"].(string)
	if base == "" {
		return nil, fmt.Errorf("grafana base_url not configured")
	}
	serverURL := strings.TrimRight(base, "/")

	switch creds.AuthType {
	case broker.AuthTypeBasic:
		return grafanaapi.NewBasicClient(serverURL, creds.Username, creds.Password)
	default:
		return grafanaapi.NewBearerClient(serverURL, creds.AccessToken)
	}
}

var toJSON = modules.ToJSON

// toRaw converts any value to jx.Raw (JSON bytes).
func toRaw(v any) (jx.Raw, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return jx.Raw(b), nil
}

// toRawSlice converts []interface{} to []jx.Raw.
func toRawSlice(v []interface{}) ([]jx.Raw, error) {
	result := make([]jx.Raw, 0, len(v))
	for _, item := range v {
		raw, err := toRaw(item)
		if err != nil {
			return nil, err
		}
		result = append(result, raw)
	}
	return result, nil
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
		ID:   "grafana:list_contact_points",
		Name: "list_contact_points",
		Descriptions: modules.LocalizedText{
			"en-US": "List all contact points.",
			"ja-JP": "すべてのコンタクトポイントを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "grafana:create_contact_point",
		Name: "create_contact_point",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new contact point (e.g., email, Slack, webhook).",
			"ja-JP": "新しいコンタクトポイント（メール、Slack、Webhook等）を作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"name":                    {Type: "string", Description: "Contact point name"},
				"type":                    {Type: "string", Description: "Integration type: email, slack, webhook, discord, pagerduty, etc."},
				"settings":               {Type: "object", Description: "Integration-specific settings (e.g., {\"addresses\": \"user@example.com\"} for email)"},
				"disable_resolve_message": {Type: "boolean", Description: "Disable notifications on alert resolution (default: false)"},
			},
			Required: []string{"name", "type", "settings"},
		},
	},
	{
		ID:   "grafana:update_contact_point",
		Name: "update_contact_point",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing contact point.",
			"ja-JP": "既存のコンタクトポイントを更新します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"uid":                     {Type: "string", Description: "Contact point UID to update"},
				"name":                    {Type: "string", Description: "Contact point name"},
				"type":                    {Type: "string", Description: "Integration type: email, slack, webhook, discord, pagerduty, etc."},
				"settings":               {Type: "object", Description: "Integration-specific settings"},
				"disable_resolve_message": {Type: "boolean", Description: "Disable notifications on alert resolution"},
			},
			Required: []string{"uid", "name", "type", "settings"},
		},
	},
	{
		ID:   "grafana:delete_contact_point",
		Name: "delete_contact_point",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete a contact point by its UID.",
			"ja-JP": "UIDでコンタクトポイントを削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"uid": {Type: "string", Description: "Contact point UID to delete"},
			},
			Required: []string{"uid"},
		},
	},
	{
		ID:   "grafana:get_notification_policy",
		Name: "get_notification_policy",
		Descriptions: modules.LocalizedText{
			"en-US": "Get the notification policy tree (routing rules for alerts).",
			"ja-JP": "通知ポリシーツリー（アラートのルーティングルール）を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "grafana:update_notification_policy",
		Name: "update_notification_policy",
		Descriptions: modules.LocalizedText{
			"en-US": "Update the notification policy tree. This overwrites the entire policy tree.",
			"ja-JP": "通知ポリシーツリーを更新します。ツリー全体が上書きされます。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"receiver":        {Type: "string", Description: "Default receiver (contact point name)"},
				"group_by":        {Type: "array", Description: "Labels to group alerts by", Items: &modules.Property{Type: "string"}},
				"group_wait":      {Type: "string", Description: "Wait time before sending grouped notifications (e.g., '5m')"},
				"group_interval":  {Type: "string", Description: "Interval between grouped notifications (e.g., '5m')"},
				"repeat_interval": {Type: "string", Description: "Repeat interval for notifications (e.g., '4h')"},
				"routes":          {Type: "array", Description: "Array of routing rules with receiver, matchers, group_by, etc."},
			},
			Required: []string{"receiver"},
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
	{
		ID:   "grafana:query_datasource",
		Name: "query_datasource",
		Descriptions: modules.LocalizedText{
			"en-US": "Query a data source (Loki, Prometheus, etc.) via Grafana proxy",
			"ja-JP": "Grafanaプロキシ経由でデータソース（Loki、Prometheus等）にクエリを実行",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"datasource_uid": {Type: "string", Description: "Data source UID (use list_datasources to find)"},
				"expr":           {Type: "string", Description: "Query expression (LogQL for Loki, PromQL for Prometheus, etc.)"},
				"from":           {Type: "string", Description: "Start time (epoch ms or relative e.g., 'now-1h')"},
				"to":             {Type: "string", Description: "End time (epoch ms or relative e.g., 'now')"},
				"max_lines":      {Type: "number", Description: "Maximum number of log lines to return (Loki only, default: 100)"},
			},
			Required: []string{"datasource_uid", "expr"},
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
	"create_alert_rule":          createAlertRule,
	"list_contact_points":        listContactPoints,
	"create_contact_point":       createContactPoint,
	"update_contact_point":       updateContactPoint,
	"delete_contact_point":       deleteContactPoint,
	"get_notification_policy":    getNotificationPolicy,
	"update_notification_policy": updateNotificationPolicy,
	"query_datasource":           queryDatasource,
}

// =============================================================================
// Read Handlers
// =============================================================================

func search(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	p := gen.SearchParams{}
	if q, ok := params["query"].(string); ok && q != "" {
		p.Query.SetTo(q)
	}
	if t, ok := params["type"].(string); ok && t != "" {
		p.Type.SetTo(t)
	}
	if tags, ok := params["tag"].([]interface{}); ok {
		for _, tag := range tags {
			if ts, ok := tag.(string); ok {
				p.Tag = append(p.Tag, ts)
			}
		}
	}
	if uids, ok := params["folder_uids"].([]interface{}); ok {
		for _, uid := range uids {
			if us, ok := uid.(string); ok {
				p.FolderUIDs = append(p.FolderUIDs, us)
			}
		}
	}
	if l, ok := params["limit"].(float64); ok {
		p.Limit.SetTo(int(l))
	}
	if pg, ok := params["page"].(float64); ok {
		p.Page.SetTo(int(pg))
	}

	res, err := c.Search(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getDashboard(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	uid, _ := params["uid"].(string)
	res, err := c.GetDashboardByUid(ctx, gen.GetDashboardByUidParams{UID: uid})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func listDatasources(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListDatasources(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getDatasource(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	uid, _ := params["uid"].(string)
	res, err := c.GetDatasourceByUid(ctx, gen.GetDatasourceByUidParams{UID: uid})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func listAlerts(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListAlertRules(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getAlert(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	uid, _ := params["uid"].(string)
	res, err := c.GetAlertRule(ctx, gen.GetAlertRuleParams{UID: uid})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func queryAnnotations(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	p := gen.QueryAnnotationsParams{}
	if from, ok := params["from"].(float64); ok {
		p.From.SetTo(int64(from))
	}
	if to, ok := params["to"].(float64); ok {
		p.To.SetTo(int64(to))
	}
	if uid, ok := params["dashboard_uid"].(string); ok && uid != "" {
		p.DashboardUID.SetTo(uid)
	}
	if pid, ok := params["panel_id"].(float64); ok {
		p.PanelId.SetTo(int(pid))
	}
	if tags, ok := params["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if ts, ok := tag.(string); ok {
				p.Tags = append(p.Tags, ts)
			}
		}
	}
	if t, ok := params["type"].(string); ok && t != "" {
		p.Type.SetTo(t)
	}
	if l, ok := params["limit"].(float64); ok {
		p.Limit.SetTo(int(l))
	}

	res, err := c.QueryAnnotations(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func listFolders(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	p := gen.ListFoldersParams{}
	if l, ok := params["limit"].(float64); ok {
		p.Limit.SetTo(int(l))
	}
	if pg, ok := params["page"].(float64); ok {
		p.Page.SetTo(int(pg))
	}

	res, err := c.ListFolders(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Write Handlers
// =============================================================================

func createUpdateDashboard(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	dashboard, ok := params["dashboard"]
	if !ok {
		return "", fmt.Errorf("dashboard is required")
	}

	dashRaw, err := toRaw(dashboard)
	if err != nil {
		return "", fmt.Errorf("failed to encode dashboard: %w", err)
	}

	req := &gen.SaveDashboardRequest{
		Dashboard: dashRaw,
	}
	if folderUID, ok := params["folder_uid"].(string); ok && folderUID != "" {
		req.FolderUid.SetTo(folderUID)
	}
	if message, ok := params["message"].(string); ok && message != "" {
		req.Message.SetTo(message)
	}
	if overwrite, ok := params["overwrite"].(bool); ok {
		req.Overwrite.SetTo(overwrite)
	}

	res, err := c.CreateOrUpdateDashboard(ctx, req)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func deleteDashboard(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	uid, _ := params["uid"].(string)
	res, err := c.DeleteDashboardByUid(ctx, gen.DeleteDashboardByUidParams{UID: uid})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func createAnnotation(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	text, _ := params["text"].(string)
	req := &gen.CreateAnnotationRequest{
		Text: text,
	}
	if uid, ok := params["dashboard_uid"].(string); ok && uid != "" {
		req.DashboardUID.SetTo(uid)
	}
	if pid, ok := params["panel_id"].(float64); ok {
		req.PanelId.SetTo(int(pid))
	}
	if t, ok := params["time"].(float64); ok {
		req.Time.SetTo(int64(t))
	}
	if te, ok := params["time_end"].(float64); ok {
		req.TimeEnd.SetTo(int64(te))
	}
	if tags, ok := params["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if ts, ok := tag.(string); ok {
				req.Tags = append(req.Tags, ts)
			}
		}
	}

	res, err := c.CreateAnnotation(ctx, req)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func deleteAnnotation(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	id, _ := params["annotation_id"].(float64)
	res, err := c.DeleteAnnotation(ctx, gen.DeleteAnnotationParams{ID: int(id)})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func createFolder(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	title, _ := params["title"].(string)
	req := &gen.CreateFolderRequest{
		Title: title,
	}
	if uid, ok := params["uid"].(string); ok && uid != "" {
		req.UID.SetTo(uid)
	}

	res, err := c.CreateFolder(ctx, req)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func deleteFolder(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	uid, _ := params["uid"].(string)
	res, err := c.DeleteFolderByUid(ctx, gen.DeleteFolderByUidParams{UID: uid})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func createAlertRule(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	title, _ := params["title"].(string)
	ruleGroup, _ := params["rule_group"].(string)
	folderUID, _ := params["folder_uid"].(string)
	condition, _ := params["condition"].(string)

	data, ok := params["data"].([]interface{})
	if !ok {
		return "", fmt.Errorf("data is required and must be an array")
	}
	dataRaw, err := toRawSlice(data)
	if err != nil {
		return "", fmt.Errorf("failed to encode data: %w", err)
	}

	req := &gen.CreateAlertRuleRequest{
		Title:     title,
		RuleGroup: ruleGroup,
		FolderUID: folderUID,
		Condition: condition,
		Data:      dataRaw,
	}

	if nds, ok := params["no_data_state"].(string); ok && nds != "" {
		req.NoDataState.SetTo(nds)
	}
	if ees, ok := params["exec_err_state"].(string); ok && ees != "" {
		req.ExecErrState.SetTo(ees)
	}
	if fd, ok := params["for_duration"].(string); ok && fd != "" {
		req.For.SetTo(fd)
	}
	if ann, ok := params["annotations"]; ok {
		raw, err := toRaw(ann)
		if err != nil {
			return "", fmt.Errorf("failed to encode annotations: %w", err)
		}
		req.Annotations = raw
	}
	if lbl, ok := params["labels"]; ok {
		raw, err := toRaw(lbl)
		if err != nil {
			return "", fmt.Errorf("failed to encode labels: %w", err)
		}
		req.Labels = raw
	}

	res, err := c.CreateAlertRule(ctx, req)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func listContactPoints(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListContactPoints(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func createContactPoint(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	name, _ := params["name"].(string)
	cpType, _ := params["type"].(string)

	settings, ok := params["settings"]
	if !ok {
		return "", fmt.Errorf("settings is required")
	}
	settingsRaw, err := toRaw(settings)
	if err != nil {
		return "", fmt.Errorf("failed to encode settings: %w", err)
	}

	req := &gen.CreateContactPointRequest{
		Name:     name,
		Type:     cpType,
		Settings: settingsRaw,
	}
	if drm, ok := params["disable_resolve_message"].(bool); ok {
		req.DisableResolveMessage.SetTo(drm)
	}

	res, err := c.CreateContactPoint(ctx, req)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func updateContactPoint(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	uid, _ := params["uid"].(string)
	name, _ := params["name"].(string)
	cpType, _ := params["type"].(string)

	settings, ok := params["settings"]
	if !ok {
		return "", fmt.Errorf("settings is required")
	}
	settingsRaw, err := toRaw(settings)
	if err != nil {
		return "", fmt.Errorf("failed to encode settings: %w", err)
	}

	req := &gen.UpdateContactPointRequest{
		Name:     name,
		Type:     cpType,
		Settings: settingsRaw,
	}
	if drm, ok := params["disable_resolve_message"].(bool); ok {
		req.DisableResolveMessage.SetTo(drm)
	}

	res, err := c.UpdateContactPoint(ctx, req, gen.UpdateContactPointParams{UID: uid})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func deleteContactPoint(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	uid, _ := params["uid"].(string)
	err = c.DeleteContactPoint(ctx, gen.DeleteContactPointParams{UID: uid})
	if err != nil {
		return "", err
	}
	return toJSON(map[string]string{"message": "contact point deleted"})
}

func getNotificationPolicy(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetNotificationPolicy(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func updateNotificationPolicy(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	receiver, _ := params["receiver"].(string)
	req := &gen.NotificationPolicy{}
	req.Receiver.SetTo(receiver)

	if groupBy, ok := params["group_by"].([]interface{}); ok {
		for _, g := range groupBy {
			if s, ok := g.(string); ok {
				req.GroupBy = append(req.GroupBy, s)
			}
		}
	}
	if gw, ok := params["group_wait"].(string); ok && gw != "" {
		req.GroupWait.SetTo(gw)
	}
	if gi, ok := params["group_interval"].(string); ok && gi != "" {
		req.GroupInterval.SetTo(gi)
	}
	if ri, ok := params["repeat_interval"].(string); ok && ri != "" {
		req.RepeatInterval.SetTo(ri)
	}
	if routes, ok := params["routes"].([]interface{}); ok {
		routesRaw, err := toRawSlice(routes)
		if err != nil {
			return "", fmt.Errorf("failed to encode routes: %w", err)
		}
		req.Routes = routesRaw
	}

	res, err := c.UpdateNotificationPolicy(ctx, req)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func queryDatasource(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	dsUID, _ := params["datasource_uid"].(string)
	expr, _ := params["expr"].(string)

	from := "now-1h"
	if f, ok := params["from"].(string); ok && f != "" {
		from = f
	}
	to := "now"
	if t, ok := params["to"].(string); ok && t != "" {
		to = t
	}

	query := map[string]any{
		"refId": "A",
		"datasource": map[string]any{
			"uid": dsUID,
		},
		"expr": expr,
	}

	if maxLines, ok := params["max_lines"].(float64); ok && maxLines > 0 {
		query["maxLines"] = int(maxLines)
	}

	queryRaw, err := toRaw(query)
	if err != nil {
		return "", fmt.Errorf("failed to encode query: %w", err)
	}

	req := &gen.DsQueryRequest{
		Queries: []jx.Raw{queryRaw},
	}
	req.From.SetTo(from)
	req.To.SetTo(to)

	res, err := c.QueryDatasource(ctx, req)
	if err != nil {
		return "", err
	}

	// res is jx.Raw (free-form), pretty-print it
	var parsed any
	if json.Unmarshal(res, &parsed) == nil {
		return toJSON(parsed)
	}
	return string(res), nil
}
