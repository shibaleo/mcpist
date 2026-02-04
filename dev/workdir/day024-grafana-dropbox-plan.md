# Grafana & Dropbox MCPモジュール実装計画

## 日付

2026-02-04

---

## 概要

新規MCPモジュールとして **Grafana** と **Dropbox** を実装する。既存モジュール（Jira, Google Drive等）のパターンに準拠。

---

## 1. Grafana モジュール

### 1.1 基本情報

| 項目 | 値 |
|------|-----|
| パッケージ | `apps/server/internal/modules/grafana/` |
| ファイル | `module.go` (単一ファイル) |
| モジュール名 | `grafana` |
| APIバージョン | `v1` |
| 認証方式 | API Key (Service Account Token) + Basic Auth |
| ベースURL | ユーザー設定可能 (`creds.Metadata["base_url"]`) |

### 1.2 認証パターン

Jiraモジュールと同じ switch ベースのデュアル認証:

- **API Key** (`store.AuthTypeAPIKey`): `Authorization: Bearer <creds.AccessToken>`
- **Basic Auth** (`store.AuthTypeBasic`): `Authorization: Basic <base64(user:pass)>`
- **ベースURL**: `creds.Metadata["base_url"]` から取得 (例: `https://grafana.example.com`)

```go
func baseURL(ctx context.Context) string {
    creds := getCredentials(ctx)
    if creds == nil { return "" }
    base, _ := creds.Metadata["base_url"].(string)
    return strings.TrimRight(base, "/")
}

func headers(ctx context.Context) map[string]string {
    creds := getCredentials(ctx)
    h := map[string]string{"Accept": "application/json", "Content-Type": "application/json"}
    switch creds.AuthType {
    case store.AuthTypeAPIKey:
        h["Authorization"] = "Bearer " + creds.AccessToken
    case store.AuthTypeBasic:
        auth := base64.StdEncoding.EncodeToString([]byte(creds.Username + ":" + creds.Password))
        h["Authorization"] = "Basic " + auth
    }
    return h
}
```

### 1.3 ツール定義 (15ツール)

#### Read ツール (8)

| # | Name | 説明 | エンドポイント | Annotation |
|---|------|------|---------------|------------|
| 1 | `search` | ダッシュボード・フォルダ検索 | `GET /api/search` | ReadOnly |
| 2 | `get_dashboard` | UID指定でダッシュボード取得 | `GET /api/dashboards/uid/:uid` | ReadOnly |
| 3 | `list_datasources` | 全データソース一覧 | `GET /api/datasources` | ReadOnly |
| 4 | `get_datasource` | UID指定でデータソース取得 | `GET /api/datasources/uid/:uid` | ReadOnly |
| 5 | `list_alerts` | アラートルール一覧 | `GET /api/v1/provisioning/alert-rules` | ReadOnly |
| 6 | `get_alert` | UID指定でアラートルール取得 | `GET /api/v1/provisioning/alert-rules/:uid` | ReadOnly |
| 7 | `query_annotations` | アノテーション検索 | `GET /api/annotations` | ReadOnly |
| 8 | `list_folders` | フォルダ一覧 | `GET /api/folders` | ReadOnly |

#### Write ツール (7)

| # | Name | 説明 | エンドポイント | Annotation |
|---|------|------|---------------|------------|
| 9 | `create_update_dashboard` | ダッシュボード作成/更新 | `POST /api/dashboards/db` | Update |
| 10 | `delete_dashboard` | ダッシュボード削除 | `DELETE /api/dashboards/uid/:uid` | Delete |
| 11 | `create_annotation` | アノテーション作成 | `POST /api/annotations` | Create |
| 12 | `delete_annotation` | アノテーション削除 | `DELETE /api/annotations/:id` | Delete |
| 13 | `create_folder` | フォルダ作成 | `POST /api/folders` | Create |
| 14 | `delete_folder` | フォルダ削除 | `DELETE /api/folders/:uid` | Delete |
| 15 | `create_alert_rule` | アラートルール作成 | `POST /api/v1/provisioning/alert-rules` | Create |

### 1.4 パラメータ詳細

**search**:
- `query` (string): 検索文字列
- `tag` (array[string]): タグフィルタ
- `type` (string): `dash-folder` / `dash-db`
- `folder_uids` (array[string]): フォルダUID指定
- `limit` (number): 最大件数 (default: 100)
- `page` (number): ページ番号

**get_dashboard**: `uid` (string, required)

**get_datasource**: `uid` (string, required)

**get_alert**: `uid` (string, required)

**query_annotations**:
- `from` (number): 開始 epoch ms
- `to` (number): 終了 epoch ms
- `dashboard_uid` (string): ダッシュボードUID
- `panel_id` (number): パネルID
- `tags` (array[string]): タグフィルタ
- `type` (string): `annotation` / `alert`
- `limit` (number): 最大件数

**list_folders**: `limit` (number), `page` (number)

**create_update_dashboard**:
- `dashboard` (object, required): ダッシュボードJSONモデル
- `folder_uid` (string): 保存先フォルダUID
- `message` (string): コミットメッセージ
- `overwrite` (boolean): 上書き許可

**delete_dashboard**: `uid` (string, required)

**create_annotation**:
- `dashboard_uid` (string): ダッシュボードUID
- `panel_id` (number): パネルID
- `time` (number): 開始時刻 epoch ms
- `time_end` (number): 終了時刻 epoch ms
- `text` (string, required): テキスト
- `tags` (array[string]): タグ

**delete_annotation**: `annotation_id` (number, required)

**create_folder**: `title` (string, required), `uid` (string)

**delete_folder**: `uid` (string, required)

**create_alert_rule**:
- `title` (string, required)
- `rule_group` (string, required)
- `folder_uid` (string, required)
- `condition` (string, required)
- `data` (array, required): クエリ/式オブジェクト
- `no_data_state` (string): `NoData` / `Alerting` / `OK`
- `exec_err_state` (string): `Alerting` / `Error` / `OK`
- `for_duration` (string): 発火待ち時間 (例: `5m`)
- `annotations` (object): サマリー等
- `labels` (object): ルーティングラベル

---

## 2. Dropbox モジュール

### 2.1 基本情報

| 項目 | 値 |
|------|-----|
| パッケージ | `apps/server/internal/modules/dropbox/` |
| ファイル | `module.go` (単一ファイル) |
| モジュール名 | `dropbox` |
| APIバージョン | `v2` |
| 認証方式 | OAuth 2.0 + PKCE (トークンリフレッシュ対応) |
| RPC ベースURL | `https://api.dropboxapi.com/2` |
| Content ベースURL | `https://content.dropboxapi.com/2` |
| Token URL | `https://api.dropboxapi.com/oauth2/token` |

### 2.2 認証パターン

Google Driveモジュールと同一パターンのOAuth 2.0トークンリフレッシュ:

```go
const tokenRefreshBuffer = 5 * 60 // 有効期限5分前にリフレッシュ

func getCredentials(ctx context.Context) *store.Credentials {
    // ... GetModuleToken(ctx, userID, "dropbox")
    if credentials.AuthType == store.AuthTypeOAuth2 && credentials.RefreshToken != "" {
        if needsRefresh(credentials) {
            refreshed, err := refreshToken(ctx, authCtx.UserID, credentials)
            ...
        }
    }
}

func refreshToken(ctx context.Context, userID string, creds *store.Credentials) {
    oauthApp, _ := store.GetTokenStore().GetOAuthAppCredentials(ctx, "dropbox")
    // POST https://api.dropboxapi.com/oauth2/token
    // grant_type=refresh_token, refresh_token=..., client_id=..., client_secret=...
    store.GetTokenStore().UpdateModuleToken(ctx, userID, "dropbox", newCreds)
}
```

### 2.3 Content エンドポイント特殊処理

Dropbox APIには2種類のエンドポイントがある:

1. **RPC エンドポイント** (`api.dropboxapi.com`): 通常のJSON POST → `client.DoJSON()` 使用
2. **Content エンドポイント** (`content.dropboxapi.com`): パラメータを `Dropbox-API-Arg` ヘッダで送信、ボディはファイル内容

```go
// ダウンロード用ヘルパー
func doContentDownload(ctx context.Context, path string, apiArg interface{}) (string, error) {
    // Dropbox-API-Arg ヘッダにJSON化したパラメータ設定
    // レスポンスボディ = ファイル内容 (最大1MB読み取り)
    // メタデータは Dropbox-API-Result レスポンスヘッダから取得
}

// アップロード用ヘルパー
func doContentUpload(ctx context.Context, path string, apiArg interface{}, content string) (string, error) {
    // Dropbox-API-Arg ヘッダにJSON化したパラメータ設定
    // Content-Type: application/octet-stream
    // リクエストボディ = ファイル内容
}
```

### 2.4 ツール定義 (15ツール)

#### Read ツール (8)

| # | Name | 説明 | エンドポイント | 種別 | Annotation |
|---|------|------|---------------|------|------------|
| 1 | `get_current_account` | ユーザー情報取得 | `POST /users/get_current_account` | RPC | ReadOnly |
| 2 | `get_space_usage` | ストレージ使用量 | `POST /users/get_space_usage` | RPC | ReadOnly |
| 3 | `list_folder` | フォルダ内一覧 | `POST /files/list_folder` | RPC | ReadOnly |
| 4 | `list_folder_continue` | 一覧続き (ページング) | `POST /files/list_folder/continue` | RPC | ReadOnly |
| 5 | `get_metadata` | ファイル/フォルダメタデータ | `POST /files/get_metadata` | RPC | ReadOnly |
| 6 | `search_files` | ファイル検索 | `POST /files/search_v2` | RPC | ReadOnly |
| 7 | `download_file` | ファイルダウンロード | `POST /files/download` | Content | ReadOnly |
| 8 | `list_shared_links` | 共有リンク一覧 | `POST /sharing/list_shared_links` | RPC | ReadOnly |

#### Write ツール (7)

| # | Name | 説明 | エンドポイント | 種別 | Annotation |
|---|------|------|---------------|------|------------|
| 9 | `upload_file` | ファイルアップロード | `POST /files/upload` | Content | Create |
| 10 | `create_folder` | フォルダ作成 | `POST /files/create_folder_v2` | RPC | Create |
| 11 | `copy_file` | ファイル/フォルダコピー | `POST /files/copy_v2` | RPC | Create |
| 12 | `move_file` | ファイル/フォルダ移動 | `POST /files/move_v2` | RPC | Create |
| 13 | `delete_file` | ファイル/フォルダ削除 | `POST /files/delete_v2` | RPC | Delete |
| 14 | `create_shared_link` | 共有リンク作成 | `POST /sharing/create_shared_link_with_settings` | RPC | Create |
| 15 | `list_revisions` | リビジョン一覧 | `POST /files/list_revisions` | RPC | ReadOnly |

### 2.5 パラメータ詳細

**list_folder**:
- `path` (string, required): フォルダパス (`""` = root, `/Documents` 等)
- `recursive` (boolean): 再帰取得
- `include_deleted` (boolean): 削除済み含む
- `include_media_info` (boolean): メディア情報含む
- `limit` (number): 最大件数 (1-2000)

**list_folder_continue**: `cursor` (string, required)

**get_metadata**:
- `path` (string, required)
- `include_media_info` (boolean)
- `include_deleted` (boolean)

**search_files**:
- `query` (string, required)
- `path` (string): 検索スコープ
- `max_results` (number): 最大件数 (1-1000)
- `file_categories` (array[string]): `image`, `document`, `pdf`, `spreadsheet`, `presentation`, `audio`, `video`, `folder`

**download_file**: `path` (string, required)

**list_shared_links**: `path` (string), `cursor` (string)

**upload_file**:
- `path` (string, required): アップロード先パス
- `content` (string, required): ファイル内容 (テキスト)
- `mode` (string): `add` (default) / `overwrite` / `update`
- `autorename` (boolean): 競合時自動リネーム
- `mute` (boolean): 通知抑制

**create_folder**: `path` (string, required), `autorename` (boolean)

**copy_file**: `from_path` (string, required), `to_path` (string, required), `autorename` (boolean)

**move_file**: `from_path` (string, required), `to_path` (string, required), `autorename` (boolean)

**delete_file**: `path` (string, required)

**create_shared_link**: `path` (string, required), `requested_visibility` (string): `public` / `team_only` / `password`

**list_revisions**: `path` (string, required), `mode` (string): `path` / `id`, `limit` (number)

---

## 3. main.go への変更

```go
// import追加
"mcpist/server/internal/modules/dropbox"
"mcpist/server/internal/modules/grafana"

// init() 内に追加
modules.RegisterModule(grafana.New())
modules.RegisterModule(dropbox.New())
```

---

## 4. 実装順序

### Phase 1: Grafana モジュール

| # | タスク | 備考 |
|---|--------|------|
| 1 | `grafana/module.go` 作成 | struct, metadata, auth, baseURL |
| 2 | Read ツール実装 (8) | search, get_dashboard, list_datasources, get_datasource, list_alerts, get_alert, query_annotations, list_folders |
| 3 | Write ツール実装 (7) | create_update_dashboard, delete_dashboard, create_annotation, delete_annotation, create_folder, delete_folder, create_alert_rule |
| 4 | main.go に登録 | import + RegisterModule |

### Phase 2: Dropbox モジュール

| # | タスク | 備考 |
|---|--------|------|
| 5 | `dropbox/module.go` 作成 | struct, metadata, OAuth refresh, content helpers |
| 6 | Read ツール実装 (8) | get_current_account, get_space_usage, list_folder, list_folder_continue, get_metadata, search_files, download_file, list_shared_links |
| 7 | Write ツール実装 (7) | upload_file, create_folder, copy_file, move_file, delete_file, create_shared_link, list_revisions |
| 8 | main.go に登録 | import + RegisterModule |

### Phase 3: ビルド確認

| # | タスク |
|---|--------|
| 9 | `go build ./cmd/server` でビルド成功確認 |

---

## 5. 参照ファイル

| ファイル | 参照理由 |
|---------|---------|
| `internal/modules/jira/module.go` | デュアル認証 + 可変ベースURL パターン |
| `internal/modules/google_drive/module.go` | OAuth 2.0 トークンリフレッシュ パターン |
| `internal/modules/ticktick/module.go` | シンプルモジュール構造の参考 |
| `internal/modules/types.go` | Module インターフェース定義 |
| `cmd/server/main.go` | モジュール登録 |

---

## 6. 完了条件

- [ ] `apps/server/internal/modules/grafana/module.go` が存在し、15ツールが定義されている
- [ ] `apps/server/internal/modules/dropbox/module.go` が存在し、15ツールが定義されている
- [ ] Grafana: API Key + Basic Auth のデュアル認証対応
- [ ] Grafana: ベースURLが `creds.Metadata["base_url"]` から動的に取得される
- [ ] Dropbox: OAuth 2.0 トークンリフレッシュ対応
- [ ] Dropbox: Content エンドポイント (download/upload) が `Dropbox-API-Arg` ヘッダで処理される
- [ ] `main.go` に両モジュールが登録されている
- [ ] `go build ./cmd/server` が成功する
