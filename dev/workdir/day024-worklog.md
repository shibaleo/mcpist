# DAY024 作業ログ

## 日付

2026-02-04

---

## 完了タスク

### Grafana & Dropbox MCP モジュール実装・テスト ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D24-001 | Grafana モジュール実装 (15ツール) | ✅ | API Key + Basic Auth デュアル認証、可変ベースURL |
| D24-002 | Dropbox モジュール実装 (15ツール) | ✅ | OAuth 2.0 + トークンリフレッシュ、Content エンドポイント対応 |
| D24-003 | main.go / tools-export に登録 | ✅ | server, tools-export 両方 |
| D24-004 | Dropbox OAuth authorize/callback ルート作成 | ✅ | POST body パラメータ方式でトークン交換 |
| D24-005 | Dropbox OAuth 接続テスト | ✅ | redirect_uri 登録、ユーザー制限解除、スコープ追加 |
| D24-006 | Dropbox 全15ツール動作確認 | ✅ | 15/15 全ツール成功 |
| D24-007 | Grafana コンソール接続対応 | ✅ | metadata 保存修正、トークン検証追加 |
| D24-008 | Grafana 全15ツール動作確認 | ✅ | 14/15 テスト成功（create_alert_rule は構成複雑のため未テスト） |

---

## 作業詳細

### 1. Grafana モジュール

| 項目 | 内容 |
|------|------|
| パッケージ | `apps/server/internal/modules/grafana/module.go` |
| 認証方式 | API Key (Service Account Token) + Basic Auth |
| ベースURL | `creds.Metadata["base_url"]` から動的取得 |
| ツール数 | 15 (Read 8 + Write 7) |

#### 実装ツール

| カテゴリ | ツール | 説明 | readOnlyHint | destructiveHint |
|----------|--------|------|--------------|-----------------|
| **検索** | `search` | ダッシュボード・フォルダ検索 | true | - |
| **ダッシュボード** | `get_dashboard` | UID指定でダッシュボード取得 | true | - |
| | `create_update_dashboard` | ダッシュボード作成/更新 | false | false |
| | `delete_dashboard` | ダッシュボード削除 | false | **true** |
| **データソース** | `list_datasources` | 全データソース一覧 | true | - |
| | `get_datasource` | UID指定でデータソース取得 | true | - |
| **アラート** | `list_alerts` | アラートルール一覧 | true | - |
| | `get_alert` | UID指定でアラートルール取得 | true | - |
| | `create_alert_rule` | アラートルール作成 | false | false |
| **アノテーション** | `query_annotations` | アノテーション検索 | true | - |
| | `create_annotation` | アノテーション作成 | false | false |
| | `delete_annotation` | アノテーション削除 | false | **true** |
| **フォルダ** | `list_folders` | フォルダ一覧 | true | - |
| | `create_folder` | フォルダ作成 | false | false |
| | `delete_folder` | フォルダ削除 | false | **true** |

#### テスト結果

| # | ツール | 結果 | 備考 |
|---|--------|------|------|
| 1 | `search` | ✅ | ダッシュボード・フォルダ検索 |
| 2 | `get_dashboard` | ✅ | UID指定でダッシュボードJSON取得 |
| 3 | `list_datasources` | ✅ | データソース一覧 |
| 4 | `get_datasource` | ✅ | 個別データソース取得 |
| 5 | `list_alerts` | ✅ | アラートルール一覧 |
| 6 | `get_alert` | ✅ | 個別アラートルール取得 |
| 7 | `query_annotations` | ✅ | アノテーション検索 |
| 8 | `list_folders` | ✅ | フォルダ一覧 |
| 9 | `create_update_dashboard` | ✅ | ダッシュボード作成・クエリ書き換え |
| 10 | `delete_dashboard` | ✅ | ダッシュボード削除 |
| 11 | `create_annotation` | ✅ | アノテーション作成 |
| 12 | `delete_annotation` | ✅ | アノテーション削除 |
| 13 | `create_folder` | ✅ | フォルダ作成 |
| 14 | `delete_folder` | ✅ | フォルダ削除 |
| 15 | `create_alert_rule` | ⏭️ | 構成が複雑なためスキップ |

**14/15 テスト完了**

---

### 2. Dropbox モジュール

| 項目 | 内容 |
|------|------|
| パッケージ | `apps/server/internal/modules/dropbox/module.go` |
| 認証方式 | OAuth 2.0 + トークンリフレッシュ |
| RPC ベースURL | `https://api.dropboxapi.com/2` |
| Content ベースURL | `https://content.dropboxapi.com/2` |
| ツール数 | 15 (Read 8 + Write 7) |

#### Dropbox API の特殊仕様

| 項目 | 内容 |
|------|------|
| RPC エンドポイント | 通常のJSON POST (`Content-Type: application/json`) |
| Content エンドポイント | `Dropbox-API-Arg` ヘッダにパラメータ、ボディはファイル内容 |
| パラメータなしPOST | `null` をJSONボディとして送信する必要あり |
| トークン交換 | POST body パラメータ方式（Basic Authヘッダは非対応） |

#### 実装ツール

| カテゴリ | ツール | 説明 | 種別 | readOnlyHint | destructiveHint |
|----------|--------|------|------|--------------|-----------------|
| **ユーザー** | `get_current_account` | ユーザー情報取得 | RPC | true | - |
| | `get_space_usage` | ストレージ使用量 | RPC | true | - |
| **ファイル** | `list_folder` | フォルダ内一覧 | RPC | true | - |
| | `list_folder_continue` | 一覧続き（ページング） | RPC | true | - |
| | `get_metadata` | メタデータ取得 | RPC | true | - |
| | `search_files` | ファイル検索 | RPC | true | - |
| | `download_file` | ファイルダウンロード | Content | true | - |
| | `upload_file` | ファイルアップロード | Content | false | false |
| | `create_folder` | フォルダ作成 | RPC | false | false |
| | `copy_file` | コピー | RPC | false | false |
| | `move_file` | 移動 | RPC | false | false |
| | `delete_file` | 削除 | RPC | false | **true** |
| **共有** | `list_shared_links` | 共有リンク一覧 | RPC | true | - |
| | `create_shared_link` | 共有リンク作成 | RPC | false | false |
| **リビジョン** | `list_revisions` | リビジョン一覧 | RPC | true | - |

#### テスト結果

| # | ツール | 結果 | 備考 |
|---|--------|------|------|
| 1 | `get_current_account` | ✅ | ユーザー情報取得 |
| 2 | `get_space_usage` | ✅ | ストレージ使用量表示 |
| 3 | `list_folder` | ✅ | ルート・サブフォルダ一覧 |
| 4 | `list_folder_continue` | ✅ | ページネーション継続 |
| 5 | `get_metadata` | ✅ | ファイル/フォルダメタデータ |
| 6 | `search_files` | ✅ | ファイル検索 |
| 7 | `download_file` | ✅ | ファイルダウンロード（Content エンドポイント） |
| 8 | `list_shared_links` | ✅ | 共有リンク一覧 |
| 9 | `upload_file` | ✅ | ファイルアップロード（Content エンドポイント） |
| 10 | `create_folder` | ✅ | フォルダ作成 |
| 11 | `copy_file` | ✅ | ファイルコピー |
| 12 | `move_file` | ✅ | ファイル移動 |
| 13 | `delete_file` | ✅ | ファイル/フォルダ削除 |
| 14 | `create_shared_link` | ✅ | 共有リンク作成 |
| 15 | `list_revisions` | ✅ | リビジョン一覧 |

**15/15 全ツール正常動作確認済み**

---

### 3. Dropbox OAuth 実装

#### OAuth フロー

| 項目 | 内容 |
|------|------|
| Authorization URL | `https://www.dropbox.com/oauth2/authorize` |
| Token URL | `https://api.dropboxapi.com/oauth2/token` |
| token_access_type | `offline`（リフレッシュトークン取得） |
| トークン交換方式 | POST body パラメータ（client_id, client_secret） |

#### トラブルシューティング

| 問題 | 原因 | 対処 |
|------|------|------|
| `Invalid redirect_uri` | Developer Console に未登録 | `https://dev.mcpist.app/api/oauth/dropbox/callback` を登録 |
| `This app has reached its user limit` | 開発アプリのユーザー制限 | Developer Console で追加ユーザーを有効化 |
| `Failed to exchange token` | Basic Auth ヘッダ方式が非対応 | POST body パラメータ方式に変更 |
| `could not decode input as JSON` | パラメータなしPOSTで nil 送信 | `json.RawMessage("null")` に変更 |
| `files.content.read` スコープ不足 | Permissions 未有効化 | Developer Console で有効化 + 再認証 |

---

### 4. コンソール側バグ修正

#### metadata 保存の修正

`token-vault.ts` の `buildCredentials()` が `basic` 認証タイプのみ metadata を保存していた。
Grafana は `api_key` 認証を使用するため、`base_url` が保存されなかった。

```typescript
// Before: basic auth 条件の中にあった
if (params.authType === 'basic') {
  credentials.metadata = params.metadata
}

// After: 認証方式に関わらず保存
if (params.metadata) {
  credentials.metadata = params.metadata
}
```

#### Grafana トークン検証の追加

`validate-token/route.ts` に Grafana 用のバリデーション関数が未実装で、`default` ケース (`{ valid: true }`) にフォールスルーしていた。
無効なトークンでも「接続成功」と表示される問題。

- `validateGrafanaToken()` 関数追加（`GET /api/org` で検証）
- `grafana` case を switch に追加
- `token-vault.ts` で Grafana の `base_url` を `validationExtra` に渡すよう修正
- `token-validator.ts` の型定義に `base_url` 追加

---

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/internal/modules/grafana/module.go` | 新規作成（15ツール、デュアル認証） |
| `apps/server/internal/modules/dropbox/module.go` | 新規作成（15ツール、OAuth + Content エンドポイント） |
| `apps/server/cmd/server/main.go` | RegisterModule(grafana, dropbox) 追加 |
| `apps/server/cmd/tools-export/main.go` | RegisterModule(grafana, dropbox) + displayName 追加 |
| `apps/console/src/app/api/oauth/dropbox/authorize/route.ts` | 新規作成（OAuth 認可URL生成） |
| `apps/console/src/app/api/oauth/dropbox/callback/route.ts` | 新規作成（POST body パラメータでトークン交換） |
| `apps/console/src/lib/token-vault.ts` | metadata 保存修正 + Grafana validationExtra 追加 |
| `apps/console/src/lib/token-validator.ts` | `base_url` 型追加 |
| `apps/console/src/app/api/validate-token/route.ts` | Grafana トークン検証関数・ケース追加 |
| `apps/console/src/lib/tools.json` | Grafana 15 + Dropbox 15 ツール追加 |

### コミット

| コミット | 内容 |
|----------|------|
| `8c6d4ba` | feat(grafana,dropbox): add Grafana and Dropbox MCP modules |
| `210f769` | fix(dropbox): add OAuth authorize and callback API routes |
| `66f9f29` | fix(dropbox): send null JSON body for no-param API endpoints |
| `285d58f` | fix(console): add client info in query parameter |
| `7072038` | fix(console): remove basic auth restriction |
| `c3adda9` | feat(console): add Grafana token validation and fix metadata persistence |

---

### Observability: 構造化ログ実装 (S7-002〜S7-005) ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D24-009 | Go Server 構造化ログ実装 (S7-002) | ✅ | log.Printf削除、Loki Push に統一 |
| D24-010 | LogToolCall に userID 追加 (S7-003) | ✅ | AuthContext から取得して渡す |
| D24-011 | セキュリティイベントログ (S7-004) | ✅ | invalid_gateway_secret を LogSecurityEvent で記録 |
| D24-012 | ラベルカーディナリティ修正 (S7-005) | ✅ | tool, event をラベルから削除→データフィールドのみ |
| D24-013 | Grafana query_datasource ツール追加 | ✅ | POST /api/ds/query でLoki/Prometheus照会可能 |
| D24-014 | Go Server に instance/region ラベル追加 | ✅ | INSTANCE_ID, INSTANCE_REGION 環境変数 |
| D24-015 | Worker に Loki Push 実装 | ✅ | waitUntil() でノンブロッキング送信 |
| D24-016 | Worker request_id 統一 | ✅ | Worker → Go Server で同一 request_id を共有 |
| D24-017 | Worker Loki シークレット設定 | ✅ | wrangler secret put (dev環境) |
| D24-018 | Loki データ受信確認 | ✅ | Worker + Go Server 両方のログを Grafana Explore で確認 |
| D24-019 | Instance/Region 自動検出 | ✅ | Render/Koyeb 組み込み環境変数からフォールバック取得 |
| D24-020 | Grafana Observability ダッシュボード作成 | ✅ | 7パネル、API経由で作成 |

---

### 5. Go Server Loki 構造化ログ

#### 変更内容

| 項目 | Before | After |
|------|--------|-------|
| ログ出力 | log.Printf + Loki Push 二重出力 | Loki Push のみ（起動ログ除く） |
| LogToolCall | userID なし | userID パラメータ追加 |
| ラベル: tool | ラベルに含む（250+カーディナリティ） | データフィールドのみ |
| ラベル: event | ラベルに含む | データフィールドのみ |
| app ラベル | ハードコード `"mcpist-dev"` | `APP_ENV` 環境変数から取得 |
| instance ラベル | なし | `INSTANCE_ID` 環境変数 |
| region ラベル | なし | `INSTANCE_REGION` 環境変数 |

#### セキュリティイベント

| イベント | トリガー | 記録先 |
|----------|----------|--------|
| `invalid_gateway_secret` | Gateway Secret 不一致 | Go Server → Loki |
| `auth_failed` | 認証失敗（トークンなし/無効） | Worker → Loki |

---

### 6. Grafana query_datasource ツール

| 項目 | 内容 |
|------|------|
| エンドポイント | `POST /api/ds/query` (Grafana Proxy API) |
| パラメータ | `datasource_uid`, `expr`, `from`, `to`, `max_lines` |
| 用途 | Loki/Prometheus のクエリをMCP経由で実行 |

---

### 7. Worker Loki Push 実装

| 項目 | 内容 |
|------|------|
| 送信方式 | `ctx.waitUntil()` でノンブロッキング |
| 認証 | Basic Auth (GRAFANA_LOKI_USER:GRAFANA_LOKI_API_KEY) |
| ラベル | `app`, `instance="worker"`, `region="cloudflare"`, `type`, `method` |
| データフィールド | `request_id`, `method`, `path`, `status_code`, `duration_ms`, `user_id`, `auth_type`, `backend` |
| request_id | Worker で生成 → Go Server に `X-Request-ID` ヘッダで共有（E2Eトレーシング） |

#### Worker 環境変数

| 変数 | 種別 | 値 |
|------|------|-----|
| `APP_ENV` | vars | `mcpist-dev` |
| `GRAFANA_LOKI_URL` | secret | `https://logs-prod-021.grafana.net` |
| `GRAFANA_LOKI_USER` | secret | `1332294` |
| `GRAFANA_LOKI_API_KEY` | secret | `glc_eyJ...` |

---

### 8. Grafana Explore でのクエリ例

```
{app="mcpist-dev"}                                    # 全ログ
{instance="worker"}                                   # Worker のみ
{instance="local"}                                    # Go Server のみ
{instance="worker"} | json | request_id="UUID..."     # 特定リクエスト追跡
{type="security"}                                     # セキュリティイベント
```

---

### Observability 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/internal/observability/loki.go` | instance/region ラベル追加、ラベルカーディナリティ修正、userID追加、log.Printf削除 |
| `apps/server/internal/modules/modules.go` | Run() から userID を LogToolCall に渡す |
| `apps/server/internal/middleware/authz.go` | LogSecurityEvent 追加、冗長 log.Printf 削除 |
| `apps/server/internal/modules/grafana/module.go` | query_datasource ツール追加 |
| `apps/console/src/lib/tools.json` | query_datasource 追加（tools-export 再生成） |
| `apps/worker/src/index.ts` | Loki Push 実装、request_id 統一 |
| `apps/worker/wrangler.toml` | APP_ENV vars 追加、Loki シークレットコメント追加 |

### 9. Instance/Region 自動検出

#### 変更内容

`loki.go` の `Init()` でプラットフォーム組み込み環境変数からフォールバック取得:

```go
func firstNonEmpty(vals ...string) string {
    for _, v := range vals {
        if v != "" { return v }
    }
    return ""
}

instanceID := firstNonEmpty(
    os.Getenv("INSTANCE_ID"),       // 手動設定（最優先）
    os.Getenv("RENDER_INSTANCE_ID"),// Render 自動提供
    os.Getenv("KOYEB_INSTANCE_ID"), // Koyeb 自動提供
    "local",
)
instanceRegion := firstNonEmpty(
    os.Getenv("INSTANCE_REGION"),   // 手動設定（最優先）
    os.Getenv("RENDER_REGION"),     // Render 手動設定
    os.Getenv("KOYEB_REGION"),      // Koyeb 自動提供
    "local",
)
```

| プラットフォーム | Instance ID | Region |
|------------------|-------------|--------|
| Render | `RENDER_INSTANCE_ID`（自動） | `RENDER_REGION`（手動設定 = `oregon`） |
| Koyeb | `KOYEB_INSTANCE_ID`（自動） | `KOYEB_REGION`（自動） |
| Worker | ハードコード `"worker"` | ハードコード `"cloudflare"` |
| ローカル | フォールバック `"local"` | フォールバック `"local"` |

---

### 10. Grafana Observability ダッシュボード

| 項目 | 内容 |
|------|------|
| フォルダ | MCPist (uid: `mcpist`) |
| ダッシュボード | MCPist Observability (uid: `mcpist-observability`) |
| 作成方法 | Grafana API (`create_update_dashboard`) 経由 |
| 自動更新 | 30秒 |
| デフォルト範囲 | 1時間 |
| テンプレート変数 | `$instance`（インスタンスフィルター） |

#### パネル一覧

| # | パネル | 種類 | クエリ |
|---|--------|------|--------|
| 1 | Request Rate by Instance | timeseries/bars | `sum by (instance) (count_over_time({app="mcpist-dev", type="request"} [1m]))` |
| 2 | Avg Response Time | timeseries/line | `avg_over_time({app="mcpist-dev", type="request"} \| json \| unwrap duration_ms [5m]) by (instance)` |
| 3 | Status Code Distribution | piechart | `sum by (status_code) (count_over_time({app="mcpist-dev", type="request"} \| json \| status_code != "" [$__range]))` |
| 4 | Tool Execution Count | barchart | `topk(10, sum by (module) (count_over_time({app="mcpist-dev", module!=""} [$__range])))` |
| 5 | Error Rate | stat | `sum(count_over_time({app="mcpist-dev", type="request"} \| json \| status_code >= 400 [$__range]))` |
| 6 | Security Events | table | `{app="mcpist-dev", type="security"}` |
| 7 | Recent Logs | logs | `{app="mcpist-dev"}` |

---

### Observability コミット

| コミット | 内容 |
|----------|------|
| `4fb88d2` | feat(server): implement structured Loki logging and add Grafana query tool |
| `256c755` | feat(observability): add Loki logging to Worker and instance labels to Server |
| `0a27e5b` | fix(observability): auto-detect instance ID and region from Render/Koyeb env vars |

---

## DAY024 サマリ

| 項目 | 内容 |
|------|------|
| 新規モジュール | Grafana（16ツール）、Dropbox（15ツール） |
| OAuth 対応 | Dropbox (POST body パラメータ方式) |
| テスト結果 | Grafana 14/15 ✅, Dropbox 15/15 ✅ |
| バグ修正 | metadata 保存制限、トークン検証フォールスルー、null body、トークン交換方式 |
| Observability | Go Server 構造化ログ (S7-002〜005)、Worker Loki Push、E2Eトレーシング |
| Instance 自動検出 | Render/Koyeb 組み込み環境変数からフォールバック取得 |
| Grafana ツール追加 | `query_datasource`（Loki/Prometheus クエリ） |
| Grafana ダッシュボード | MCPist Observability（7パネル、API経由で作成） |
| コミット数 | 9 |
| 累計モジュール数 | 20 (既存18 + Grafana, Dropbox) |

---

## 次回の作業

1. Sprint 007 Phase 3: 仕様書の実装追従更新 (S7-020〜026)
2. Grafana ダッシュボードの改善・アラート設定
3. S7 残タスクの確認と計画
