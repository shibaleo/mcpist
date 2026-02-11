# Grafana Module

## Status

- **Status**: Implemented (ogen)
- **Date**: 2026-02-11
- **API Version**: v1
- **Client**: ogen generated (`pkg/grafanaapi/`)

## Endpoint Catalog

### Search

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `search` | GET | `/api/search` | query, tag, type, folder_uids, limit, page |

### Dashboards

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `get_dashboard` | GET | `/api/dashboards/uid/{uid}` | **uid** |
| `create_update_dashboard` | POST | `/api/dashboards/db` | **dashboard**, folder_uid, message, overwrite |
| `delete_dashboard` | DELETE | `/api/dashboards/uid/{uid}` | **uid** |

### Datasources

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_datasources` | GET | `/api/datasources` | (none) |
| `get_datasource` | GET | `/api/datasources/uid/{uid}` | **uid** |
| `query_datasource` | POST | `/api/ds/query` | **datasource_uid**, **expr**, from, to, max_lines |

### Alert Rules

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_alerts` | GET | `/api/v1/provisioning/alert-rules` | (none) |
| `get_alert` | GET | `/api/v1/provisioning/alert-rules/{uid}` | **uid** |
| `create_alert_rule` | POST | `/api/v1/provisioning/alert-rules` | **title**, **rule_group**, **folder_uid**, **condition**, **data**, no_data_state, exec_err_state, for_duration, annotations, labels |

### Annotations

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `query_annotations` | GET | `/api/annotations` | from, to, dashboard_uid, panel_id, tags, type, limit |
| `create_annotation` | POST | `/api/annotations` | **text**, dashboard_uid, panel_id, time, time_end, tags |
| `delete_annotation` | DELETE | `/api/annotations/{id}` | **annotation_id** |

### Folders

| Tool | Method | Endpoint | Params (required **bold**) |
|------|--------|----------|---------------------------|
| `list_folders` | GET | `/api/folders` | limit, page |
| `create_folder` | POST | `/api/folders` | **title**, uid |
| `delete_folder` | DELETE | `/api/folders/{uid}` | **uid** |

## Summary

- **Total**: 16 tools (GET: 8, POST: 5, DELETE: 3)
- **ogen operations**: 16 (全ツールが ogen 単体)
- **複合ツール**: なし

## Tool Classification

| 分類 | ツール | 説明 |
|------|--------|------|
| **ogen 単体 (Read)** | search, get_dashboard, list_datasources, get_datasource, list_alerts, get_alert, query_annotations, list_folders | ogen client の 1 メソッド呼出 → `toJSON(res)` |
| **ogen 単体 (Write)** | create_update_dashboard, delete_dashboard, create_annotation, delete_annotation, create_folder, delete_folder, create_alert_rule, query_datasource | リクエストボディ構築 → ogen メソッド呼出 → `toJSON(res)` |

## Response Schemas

subset spec で定義しているスキーマ:

| Schema | Used by |
|--------|---------|
| SearchResult | search |
| DashboardFullWithMeta, DashboardMeta | getDashboardByUid |
| SaveDashboardRequest, SaveDashboardResponse | createOrUpdateDashboard |
| DeleteDashboardResponse | deleteDashboardByUid |
| Datasource | listDatasources, getDatasourceByUid |
| DsQueryRequest | queryDatasource |
| AlertRule | listAlertRules, getAlertRule, createAlertRule (response) |
| CreateAlertRuleRequest | createAlertRule (request) |
| Annotation | queryAnnotations |
| CreateAnnotationRequest, CreateAnnotationResponse | createAnnotation |
| DeleteAnnotationResponse | deleteAnnotation |
| Folder | listFolders, createFolder (response) |
| CreateFolderRequest | createFolder (request) |
| DeleteFolderResponse | deleteFolderByUid |

## Notes

### 動的 server URL

GitHub/Supabase は固定 URL (`api.github.com`, `api.supabase.com`) だが、
Grafana は `credentials.Metadata["base_url"]` からインスタンス URL を取得する。
`newOgenClient()` で `strings.TrimRight(base, "/")` してから `gen.NewClient()` に渡す。

### 二重認証 (Bearer + Basic)

Grafana は API Key (Bearer) と Basic auth の両方をサポートする。
ogen の `SecuritySource` インタフェースは両方のメソッドを要求するため、
2 つの実装を用意:

- `bearerSecuritySource`: `BearerAuth()` → トークン返却、`BasicAuth()` → `ogenerrors.ErrSkipClientSecurity`
- `basicSecuritySource`: `BasicAuth()` → username/password 返却、`BearerAuth()` → `ErrSkipClientSecurity`

`newOgenClient()` で `creds.AuthType` に応じて切り替える。

### Free-form JSON フィールド

以下のフィールドは任意の JSON 構造を持つため、spec で `schema: {}` として定義:

| フィールド | 型 | 用途 |
|-----------|------|------|
| `DashboardFullWithMeta.dashboard` | `jx.Raw` | ダッシュボード JSON model (パネル、変数、テンプレート等) |
| `SaveDashboardRequest.dashboard` | `jx.Raw` | 作成/更新時のダッシュボード JSON |
| `DsQueryRequest.queries[*]` | `[]jx.Raw` | データソースクエリ (refId, datasource, expr 等) |
| `queryDatasource` response | `jx.Raw` | クエリ結果 (データソース種別で構造が異なる) |
| `AlertRule.data[*]` | `[]jx.Raw` | アラート条件の query/expression オブジェクト |
| `AlertRule.annotations` | `jx.Raw` | アノテーション map (summary, description 等) |
| `AlertRule.labels` | `jx.Raw` | ラベル map (ルーティング用) |

Go 側では `toRaw()` / `toRawSlice()` ヘルパーで `map[string]any` → `jx.Raw` に変換する。
`queryDatasource` のレスポンスは `jx.Raw` → `json.Unmarshal` → `toJSON` でプリティプリントする。

### queryDatasource のクエリ構築

`query_datasource` ツールはパラメータからクエリオブジェクトを構築する:

```go
query := map[string]any{
    "refId": "A",
    "datasource": map[string]any{"uid": dsUID},
    "expr": expr,
}
// max_lines があれば追加 (Loki 用)
```

このクエリを `toRaw()` → `jx.Raw` にシリアライズし、`DsQueryRequest.Queries` に設定する。
`from`/`to` はデフォルト `"now-1h"` / `"now"` で、Grafana の相対時間記法をそのまま渡す。
