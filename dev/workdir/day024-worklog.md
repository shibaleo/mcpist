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

---

### S7-025: Credentials リファクタリング ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D24-021 | 仕様書更新 (dtl-itr-MOD-TVL.md v2.2) | ✅ | OAuth 1.0a フィールド名標準化、モジュール別形式更新 |
| D24-022 | 仕様書更新 (dtl-itr-CON-TVL.md) | ✅ | MOD-TVL v2.2 準拠 |
| D24-023 | 統合計画書作成 | ✅ | day024-credentials-refactor-plan.md |
| D24-024 | Phase 1: Console validator 分割 | ✅ | `/api/credentials/validate` + validators/ ディレクトリ |
| D24-025 | Phase 2: Go Server credentials 構造変更 | ✅ | APIKey フィールド追加、レガシーフィールド削除 |
| D24-026 | Phase 3: Console token-vault.ts 変更 | ✅ | Trello の api_key フィールド出力 |
| D24-027 | データマイグレーション作成 | ✅ | Trello username → api_key |
| D24-028 | デプロイ・マイグレーション実行 | ✅ | supabase db push + Vercel/Render デプロイ |
| D24-029 | 統合テスト | ✅ | 全モジュール動作確認 |

---

### 11. Credentials リファクタリング詳細

#### Phase 1: Console API リファクタリング

| Before | After |
|--------|-------|
| `/api/validate-token` | `/api/credentials/validate` (REST スタイル) |
| `/api/token-vault` | 削除（未使用） |
| 1ファイルに全 validator | `validators/` ディレクトリに分割 |

**新規ファイル:**
- `apps/console/src/app/api/credentials/validate/route.ts` (42行)
- `apps/console/src/app/api/credentials/validate/validators/types.ts`
- `apps/console/src/app/api/credentials/validate/validators/index.ts`
- `apps/console/src/app/api/credentials/validate/validators/notion.ts`
- `apps/console/src/app/api/credentials/validate/validators/github.ts`
- `apps/console/src/app/api/credentials/validate/validators/supabase.ts`
- `apps/console/src/app/api/credentials/validate/validators/jira.ts`
- `apps/console/src/app/api/credentials/validate/validators/confluence.ts`
- `apps/console/src/app/api/credentials/validate/validators/trello.ts`
- `apps/console/src/app/api/credentials/validate/validators/grafana.ts`

#### Phase 2: Go Server Credentials 構造変更

| Before | After |
|--------|-------|
| `creds.Username` で api_key (Trello) | `creds.APIKey` フィールド |
| `AuthType2`, `Metadata2` レガシーフィールド | 削除 |
| `GetAuthType()`, `GetMetadata()` ヘルパー | 削除（直接フィールド参照） |

**変更ファイル:**
- `apps/server/internal/store/token.go` - Credentials 構造体リファクタリング
- `apps/server/internal/modules/notion/client.go` - `creds.AuthType` 直接参照
- `apps/server/internal/modules/trello/module.go` - `creds.APIKey` 使用

#### Phase 3: Console token-vault.ts 変更

```typescript
// Trello の場合は api_key フィールドを出力
if (isTrello) {
  credentials.api_key = params.username  // api_key は username パラメータで渡される
  credentials.access_token = params.accessToken
}
```

#### データマイグレーション

**ファイル:** `supabase/migrations/20260205000000_migrate_trello_credentials.sql`

```sql
-- Trello credentials: username → api_key に変換
UPDATE mcpist.user_credentials
SET credentials = jsonb_build_object(
    'auth_type', 'api_key',
    'api_key', credentials::jsonb->>'username',
    'access_token', credentials::jsonb->>'access_token'
  )::text
WHERE module = 'trello'
  AND credentials::jsonb ? 'username'
  AND NOT (credentials::jsonb ? 'api_key');
```

#### Validator 設計方針

| タイプ | Validator | 理由 |
|--------|-----------|------|
| OAuth 2.0 | 不要 | 認証プロバイダがトークン発行 |
| API Key / Basic | 必要 | 手動入力トークンの事前検証 |

**Validator 実装モジュール (7種):**
- notion, github, supabase (api_key)
- jira, confluence (basic)
- trello, grafana (api_key)

#### 統合テスト結果

| モジュール | auth_type | 結果 |
|-----------|-----------|------|
| notion | oauth2 | ✅ |
| todoist | oauth2 | ✅ |
| google_calendar | oauth2 | ✅ |
| airtable | oauth2 | ✅ |
| supabase | api_key | ✅ |
| trello | api_key | ✅ (マイグレーション成功) |
| jira | basic | ✅ |
| grafana | api_key | ✅ |

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
| **Credentials リファクタリング** | **API 分割、構造体変更、データマイグレーション** |
| コミット数 | 10 |
| 累計モジュール数 | 20 (既存18 + Grafana, Dropbox) |

---

### S7-030: credit_transactions スキーマ改善 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D24-030 | meta_tool/details カラム追加 | ✅ | run/batch 統一トラッキング |
| D24-031 | consume_user_credits RPC 更新 | ✅ | 新シグネチャ (meta_tool, details) |
| D24-032 | Go Server 対応 (run/batch) | ✅ | handler.go, user.go, ToolDetail 構造体 |
| D24-033 | get_my_usage RPC 更新 | ✅ | details JSONB からモジュール別集計 |
| D24-034 | ダッシュボード日付ピッカー追加 | ✅ | 期間指定利用量表示 |
| D24-035 | レガシーカラム削除マイグレーション | ✅ | module, tool, task_id DROP |
| D24-036 | データマイグレーション（既存レコード変換） | ✅ | batch レコード削除、non-consume にデフォルト値設定 |

---

### S7-031: ランニングバランス方式実装 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D24-037 | 実装計画書作成 | ✅ | running-balance-implementation-plan.md |
| D24-038 | running_free/running_paid カラム追加 | ✅ | credit_transactions にランニングバランス列 |
| D24-039 | 既存データバックフィル | ✅ | credits テーブルから逆算してランニングバランス設定 |
| D24-040 | consume_user_credits RPC 更新 | ✅ | ランニングバランス読み書き対応 |
| D24-041 | add_user_credits RPC 更新 | ✅ | 購入/付与時もランニングバランス記録 |
| D24-042 | get_user_context RPC 更新 | ✅ | ランニングバランスから残高読み取り |
| D24-043 | NOT NULL 制約追加 | ✅ | running_free, running_paid |
| D24-044 | check_credit_integrity 関数作成 | ✅ | credits テーブルとの整合性チェック |
| D24-045 | 本番 DB 適用・動作確認 | ✅ | バックフィル正常、新規取引も正常 |

---

### S7-032: credits テーブル廃止 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D24-046 | consume_user_credits から credits 参照削除 | ✅ | users テーブルで存在確認に変更 |
| D24-047 | add_user_credits から credits 参照削除 | ✅ | ランニングバランスのみで動作 |
| D24-048 | get_user_context から credits 参照削除 | ✅ | フォールバックを 0/0 に変更 |
| D24-049 | handle_new_user トリガー更新 | ✅ | credits INSERT 削除 |
| D24-050 | check_credit_integrity 関数削除 | ✅ | 比較対象消滅 |
| D24-051 | credits テーブル DROP | ✅ | CASCADE で RLS・トリガーも削除 |
| D24-052 | 本番 DB 適用・動作確認 | ✅ | get_user_context, add_user_credits 正常動作確認 |

---

### 12. ランニングバランス方式 詳細

#### 設計

| 項目 | Before | After |
|------|--------|-------|
| 残高の保存先 | `credits` テーブル (キャッシュ) | `credit_transactions.running_free/running_paid` |
| 残高取得 | `SELECT * FROM credits` | 最新トランザクションの `running_free/running_paid` |
| パフォーマンス | O(1) | O(1)（インデックス `user_id, created_at DESC`） |
| データ整合性 | credits と transactions が乖離する可能性 | 各レコードが正確な残高を保持（イベントソーシング準拠） |
| credits テーブル | 必須 | 廃止（DROP 済み） |

#### バックフィル戦略

1. 各ユーザーの最新トランザクションに `credits` テーブルの現在値をセット
2. それ以前のレコードは「最新残高 - 以降の累積 amount」で逆算
3. 残りの NULL は 0 で埋め
4. NOT NULL 制約追加

#### 検証結果

| 確認項目 | 結果 |
|----------|------|
| バックフィル（既存データ） | running_paid が正しく逆算されている |
| 新規消費（マイグレーション後） | running_free/running_paid が正しく書き込まれている |
| credits テーブルとの整合性 | check_credit_integrity() = 0件（完全一致） |
| credits テーブル削除後の RPC | get_user_context, add_user_credits 正常動作 |
| フォールバック（取引なしユーザー） | users テーブルで存在確認 → 0/0 返却 |

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `supabase/migrations/20260205000002_credit_transactions_details.sql` | meta_tool/details 追加、consume RPC 更新、get_my_usage RPC 更新 |
| `supabase/migrations/20260205000003_credit_transactions_cleanup.sql` | レガシーカラム削除、NOT NULL 制約、データマイグレーション |
| `supabase/migrations/20260205000004_running_balance.sql` | running balance カラム追加、バックフィル、全 RPC 更新 |
| `supabase/migrations/20260205000005_drop_credits_table.sql` | credits テーブル参照削除、credits テーブル DROP |
| `apps/server/internal/store/user.go` | ToolDetail 構造体追加、ConsumeCredit シグネチャ変更 |
| `apps/server/internal/mcp/handler.go` | run/batch で details 配列を渡す |
| `apps/console/src/app/(console)/dashboard/page.tsx` | 日付ピッカー追加、getMyUsage 呼び出し |
| `apps/console/src/lib/credits.ts` | getMyUsage, UsageStats 型追加 |
| `dev/workdir/running-balance-implementation-plan.md` | 実装計画書 |

---

## DAY024 サマリ（追記）

| 項目 | 内容 |
|------|------|
| **credit_transactions スキーマ改善** | **meta_tool/details 統一、レガシーカラム削除** |
| **ランニングバランス方式** | **running_free/running_paid 追加、全 RPC 更新** |
| **credits テーブル廃止** | **DROP TABLE、全参照削除** |
| アーキテクチャ変更 | デュアルテーブル → イベントソーシング（Running Balance） |
| データ整合性 | 各取引レコードが正確な残高を保持 |

---

### S7-033: Per-user Burst 制限 & Batch サイズ上限 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D24-053 | Rate Limit 方式検討 | ✅ | Worker KV → Go Server インメモリに決定 |
| D24-054 | per-user burst 制限ミドルウェア実装 | ✅ | スライディングウィンドウ 10 req/sec |
| D24-055 | ミドルウェアチェーンへの組み込み | ✅ | Authorize → RateLimit → MCPHandler |
| D24-056 | batch サイズ上限追加 | ✅ | 最大 10 コマンド/バッチ |
| D24-057 | metatool 記述更新 | ✅ | batch 説明に制限事項を日英追記 |
| D24-058 | デプロイ・動作確認 | ✅ | 11コマンド → 拒否、10コマンド → 成功 |

---

### 13. Per-user Burst 制限 詳細

#### 背景

以前 Worker に実装していた Rate Limit（IP 単位 1000 req/分、ユーザー単位 5 req/秒）は KV コスト削減のため削除済み（commit `ec055b7`、2026-01-23）。burst 制限なしではリソース保護ができないため、KV に依存しない方式で再実装。

#### 方式比較

| 方式 | KV コスト | 精度 | 採用 |
|------|-----------|------|------|
| Worker + KV | +2 ops/req | 高 | ❌ 無料枠超過リスク |
| Worker + Cache API | 0 | 中（PoP 単位） | ❌ 精度不十分 |
| **Go Server インメモリ** | **0** | **高** | **✅ 採用** |

#### 実装

| 項目 | 内容 |
|------|------|
| ファイル | `apps/server/internal/middleware/ratelimit.go` (新規) |
| アルゴリズム | スライディングウィンドウ（直近 1 秒間のタイムスタンプ配列） |
| 制限値 | 10 req/sec（per-user） |
| ロック | `sync.Mutex`（全ユーザー共有 1 ロック） |
| メモリ | ユーザーあたり数十バイト（`[]time.Time` + lastAccess） |
| クリーンアップ | 60 秒間隔で 5 分間アクセスなしのエントリを削除 |
| 超過時レスポンス | HTTP 429 + `Retry-After: 1` + batch 利用案内メッセージ |

#### ミドルウェアチェーン

```
/mcp → Authorize(Gateway Secret + UserContext) → RateLimit(10 req/sec) → MCPHandler
```

- Rate Limiter は Authorize の**後**に配置
- Gateway Secret 検証を通過した正規リクエストのみカウント
- context から `AuthContext.UserID` を取得（ヘッダ直読みではない）

#### Batch サイズ上限

| 項目 | 内容 |
|------|------|
| 上限 | 10 コマンド/バッチ |
| チェック箇所 | `checkBatchPermissions()` 内 |
| 超過時エラー | `batch too large: N commands (max 10)` |

#### Nano インスタンスへの影響

| リソース | 影響 |
|----------|------|
| CPU | 外部 API 待ちで I/O ブロック中は CPU 未使用。0.1 vCPU でも問題なし |
| メモリ | 1000 ユーザー分保持しても数十 KB。goroutine は ~8KB/本 |
| 最悪ケース | 10 req/sec × 10 tools/batch = 100 ツール/秒（外部 API が先にボトルネック） |

#### 制限の全体像

| 制限 | 値 | 場所 |
|------|-----|------|
| Per-user burst | 10 req/sec | Go Server ミドルウェア |
| Batch サイズ | 最大 10 コマンド | Go Server handler |
| 月間クォータ | 1000 credits (free) | Supabase RPC |
| リクエストタイムアウト | 30 秒 | Worker |

#### テスト結果

| テスト | 結果 |
|--------|------|
| 11 コマンド batch | `batch too large: 11 commands (max 10)` で拒否 ✅ |
| 10 コマンド batch | 全 10 タスク成功 ✅ |
| 15 並列 curl → burst 制限 | 200×10, 429×5 ✅ |

#### Burst 制限テスト詳細

本番 Render サーバーに対して 15 リクエストを並列送信:

```bash
for i in $(seq 1 15); do
  curl -s -o /dev/null -w "req $i: %{http_code}\n" \
    -X POST "REDACTED_PRIMARY_API_URL/mcp" \
    -H "Content-Type: application/json" \
    -H "X-Gateway-Secret: ***" \
    -H "X-User-ID: ***" \
    -H "X-Request-ID: burst-test-$i" \
    -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' &
done
wait
```

結果（到着順）:

| req | status | 備考 |
|-----|--------|------|
| 1 | 200 | |
| 2 | 429 | 11番目以降に到着 |
| 3 | 200 | |
| 4 | 429 | |
| 5 | 429 | |
| 6 | 200 | |
| 7 | 200 | |
| 8 | 200 | |
| 9 | 200 | |
| 10 | 200 | |
| 11 | 429 | |
| 12 | 200 | |
| 13 | 200 | |
| 14 | 200 | |
| 15 | 429 | |

**200: 10件、429: 5件** — 10 req/sec の制限が正確に機能。並列送信のため番号順と到着順は異なるが、先着10件が許可され残り5件が拒否された。

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/internal/middleware/ratelimit.go` | 新規: per-user スライディングウィンドウ burst 制限 |
| `apps/server/cmd/server/main.go` | ミドルウェアチェーンに RateLimiter 追加 |
| `apps/server/internal/mcp/handler.go` | batch サイズ上限 (10) 追加 |
| `apps/server/internal/modules/modules.go` | batch metatool 記述に制限事項を日英追記 |

#### コミット

| コミット | 内容 |
|----------|------|
| `5d95526` | feat(server): add per-user burst limit and batch size cap |

---

### S7-034: Console UI 改善 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D24-059 | localhost フォールバック除去 | ✅ | `getMcpServerUrl()` ヘルパー作成、4ファイルに適用 |
| D24-060 | ダッシュボード サービスカードリンク修正 | ✅ | `/tools` → `/services` |
| D24-061 | ダッシュボード highlight フラッシュ修正 | ✅ | `!loading &&` 条件追加 |
| D24-062 | ダークテーマ調整 (Obsidian風) | ✅ | 背景・カード・ボーダー色を少し明るく |
| D24-063 | ライトテーマ追加 (アイボリー系) | ✅ | `:root` をアイボリー/クリーム系に変更 |
| D24-064 | フォーム視認性改善 | ✅ | input/textarea に `bg-white` 適用（背景 < カード < フォーム） |
| D24-065 | サイドバーラベル短縮 | ✅ | MCP接続→MCP、サービス接続→サービス、ツール設定→ツール |
| D24-066 | サイドバーリサイズハンドル改善 | ✅ | `w-1.5 bg-sidebar-border` → `w-1 bg-transparent hover:bg-primary/30` |
| D24-067 | OAuth consents セクション削除 | ✅ | connections ページから認可済みクライアント UI を削除 |
| D24-068 | textarea リサイズ有効化 | ✅ | ツール設定のカスタム説明を `resize-y` に変更 |
| D24-069 | .env.local MCP URL 修正 | ✅ | `api.dev.mcpist.app` → `mcp.dev.mcpist.app` (DNS解決失敗の修正) |

---

### 14. Console UI 改善 詳細

#### テーマ変更

| 項目 | Before | After (Light) | After (Dark) |
|------|--------|---------------|--------------|
| 背景 | `#171717` | `#f0ebe3` (アイボリー) | `#1a1a1e` (Obsidian風) |
| カード | `#1f1f1f` | `#f8f5f0` | `#222226` |
| ボーダー | `#2e2e2e` | `#ddd8d0` | `#333338` |
| サイドバー | `#1f1f1f` | `#efe9e0` | `#1e1e22` |
| 前景色 | `#b5b5b5` | `#2c2c2c` | `#b5b5b5` (変更なし) |

**視覚的階層**: 背景 (最暗) < カード (中間) < フォーム入力 (最明/白)

#### getMcpServerUrl() ヘルパー

`apps/console/src/lib/env.ts` に新規作成。`NEXT_PUBLIC_MCP_SERVER_URL` 未設定時にエラーをスローする。

**適用ファイル:**
- `apps/console/src/app/(admin)/admin/page.tsx`
- `apps/console/src/app/(console)/connections/page.tsx`
- `apps/console/src/app/(console)/dev/mcp-client/page.tsx`
- `apps/console/src/app/oauth/callback/page.tsx`

#### OAuth consents 削除

connections ページから認可済みクライアントの表示・取り消し UI を削除。
削除した RPC 呼び出し: `list_my_oauth_consents`, `revoke_my_oauth_consent`（RPC 自体は残存）。

#### DNS 問題の調査

| 問題 | 原因 | 対処 |
|------|------|------|
| API Key テスト `Failed to fetch` | `.env.local` が `api.dev.mcpist.app` を参照（DNS レコード不在） | `mcp.dev.mcpist.app` に修正 |

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/lib/env.ts` | `getMcpServerUrl()` ヘルパー作成 |
| `apps/console/src/styles/globals.css` | ライトテーマ (アイボリー)、ダークテーマ (Obsidian風)、dot-grid/glass-sidebar のライト/ダーク対応 |
| `apps/console/src/components/sidebar.tsx` | ラベル短縮、リサイズハンドル改善 |
| `apps/console/src/components/ui/input.tsx` | `bg-transparent` → `bg-white dark:bg-input/30` |
| `apps/console/src/components/ui/textarea.tsx` | `bg-transparent` → `bg-white dark:bg-input/30` |
| `apps/console/src/app/(console)/dashboard/page.tsx` | href `/services`、highlight フラッシュ修正 |
| `apps/console/src/app/(console)/connections/page.tsx` | `getMcpServerUrl()`、OAuth consents 削除 |
| `apps/console/src/app/(console)/tools/page.tsx` | textarea `resize-y` |
| `apps/console/src/app/(admin)/admin/page.tsx` | `getMcpServerUrl()` |
| `apps/console/src/app/(console)/dev/mcp-client/page.tsx` | `getMcpServerUrl()` |
| `apps/console/src/app/oauth/callback/page.tsx` | `getMcpServerUrl()` |

---

## 次回の作業

1. Sprint 007 Phase 3: 仕様書の実装追従更新 (S7-020〜026) - 残りのタスク
2. Grafana ダッシュボードの改善・アラート設定
3. S7 残タスクの確認と計画
4. クレジットモデル仕様書 (dtl-spc-credit-model.md) をランニングバランス方式に更新
5. 言語設定が MCP ツールスキーマに反映されない問題の調査継続
