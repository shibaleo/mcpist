# DAY023 作業ログ

## 日付

2026-02-03

---

## 完了タスク

### Airtable OAuth 2.0 + PKCE 対応 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D23-010 | Airtable OAuth App 登録 | ✅ | https://airtable.com/create/oauth |
| D23-011 | authorize ルート作成 | ✅ | OAuth 2.0 + PKCE (S256) |
| D23-012 | callback ルート作成 | ✅ | Basic Auth でトークン交換 |
| D23-013 | oauth-apps.ts 更新 | ✅ | OAUTH_PROVIDERS, OAUTH_CONFIGS に追加 |
| D23-014 | services/page.tsx 更新 | ✅ | OAuth + PAT alternativeAuth パターン |
| D23-015 | Go モジュール OAuth 対応 | ✅ | トークンリフレッシュ（ローテーション）実装 |
| D23-016 | 全11ツール動作確認 | ✅ | list_bases, list_records 等すべて成功 |

---

## 作業詳細

### 1. Airtable OAuth 2.0 の特徴

| 項目 | 内容 |
|------|------|
| Authorization URL | `https://airtable.com/oauth2/v1/authorize` |
| Token URL | `https://airtable.com/oauth2/v1/token` |
| PKCE | **必須** (code_challenge_method: S256) |
| トークン有効期限 | 2ヶ月 |
| リフレッシュトークン | あり（使用時に新しいペア発行 = ローテーション） |
| トークン交換 | Basic Auth (`client_id:client_secret` を Base64) |

### 2. PKCE 実装

- `code_verifier` をサーバー側で生成
- `code_challenge` (S256) を認可リクエストに付与
- `code_verifier` を httpOnly Cookie に保存（10分TTL、path `/api/oauth/airtable`）
- callback で Cookie から読み取りトークン交換に使用

### 3. リフレッシュトークンローテーション

Airtable はリフレッシュ時に **新しいリフレッシュトークン** を返す。古いトークンは無効化される。

```go
func refreshToken(ctx context.Context, cred *store.ModuleCredential) error {
    // Basic Auth ヘッダー
    basicAuth := base64.StdEncoding.EncodeToString(
        []byte(clientID + ":" + clientSecret))

    // リフレッシュリクエスト
    resp, _ := http.PostForm(tokenURL, url.Values{
        "grant_type":    {"refresh_token"},
        "refresh_token": {cred.RefreshToken},
    })

    // 新しい access_token AND refresh_token を保存
    store.UpdateModuleToken(ctx, userID, "airtable",
        newAccessToken, newRefreshToken, expiresAt)
}
```

### 4. スコープ

```
data.records:read
data.records:write
schema.bases:read
schema.bases:write
```

### 5. Airtable テスト結果

| ツール | 結果 | 備考 |
|--------|------|------|
| `list_bases` | ✅ | ベース一覧取得 |
| `get_base_schema` | ✅ | テーブル・フィールド定義取得 |
| `list_records` | ✅ | レコード一覧（ページネーション対応） |
| `get_record` | ✅ | 単一レコード取得 |
| `create_records` | ✅ | レコード作成 |
| `update_records` | ✅ | レコード更新 |
| `delete_records` | ✅ | レコード削除 |
| `create_table` | ✅ | テーブル作成 |
| `update_table` | ✅ | テーブル名変更 |
| `create_field` | ✅ | フィールド追加 |
| `update_field` | ✅ | フィールド名変更 |

**備考:** Airtable API にはテーブル削除エンドポイントが存在しない

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/api/oauth/airtable/authorize/route.ts` | 新規作成（PKCE + S256） |
| `apps/console/src/app/api/oauth/airtable/callback/route.ts` | 新規作成（Basic Auth トークン交換） |
| `apps/console/src/lib/oauth-apps.ts` | Airtable プロバイダー追加 |
| `apps/console/src/app/(console)/services/page.tsx` | OAuth + PAT alternativeAuth 追加 |
| `apps/server/internal/modules/airtable/module.go` | リフレッシュトークンローテーション実装 |

### コミット

| コミット | 内容 |
|----------|------|
| `7c3c90c` | feat(airtable): add OAuth 2.0 + PKCE authentication |

---

## TickTick MCP モジュール実装 ✅

### 完了タスク

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D23-020 | TickTick Open API 調査 | ✅ | v1 API、11エンドポイント |
| D23-021 | `modules/ticktick/module.go` 作成 | ✅ | 11ツール実装 |
| D23-022 | `main.go` / `tools-export` に RegisterModule 追加 | ✅ | server, tools-export 両方 |
| D23-023 | OAuth authorize/callback ルート作成 | ✅ | OAuth 2.0、Basic Auth トークン交換 |
| D23-024 | `oauth-apps.ts` に TickTick プロバイダー追加 | ✅ | OAUTH_PROVIDERS, OAUTH_CONFIGS |
| D23-025 | `services/page.tsx` に authConfig 追加 | ✅ | OAuth 方式 |
| D23-026 | `module-data.ts` にアイコン追加 | ✅ | check-circle-2 |
| D23-027 | tools.json 再生成 | ✅ | 11ツール確認 |
| D23-028 | 全11ツール動作確認 | ✅ | 全ツール成功 |

### TickTick Open API v1 の特徴

| 項目 | 内容 |
|------|------|
| Base URL | `https://api.ticktick.com/open/v1` |
| Authorization URL | `https://ticktick.com/oauth/authorize` |
| Token URL | `https://ticktick.com/oauth/token` |
| トークン交換 | Basic Auth (`client_id:client_secret` を Base64) |
| スコープ | `tasks:read tasks:write` |
| Developer Portal | https://developer.ticktick.com/manage |

### スコープについて

TickTick Open API v1 は `tasks:read` と `tasks:write` の2スコープのみ提供。
これでプロジェクト・タスクの全操作をカバー。
Habit（習慣）関連のエンドポイントは Open API v1 に存在しない。

### 実装したツール（11ツール）

| カテゴリ | ツール | 説明 | readOnlyHint | destructiveHint |
|----------|--------|------|--------------|-----------------|
| **プロジェクト** | `list_projects` | プロジェクト一覧 | true | - |
| | `get_project` | プロジェクト詳細 | true | - |
| | `get_project_data` | タスク含むプロジェクトデータ | true | - |
| | `create_project` | プロジェクト作成 | false | false |
| | `update_project` | プロジェクト更新 | false | false |
| | `delete_project` | プロジェクト削除 | false | **true** |
| **タスク** | `get_task` | タスク詳細 | true | - |
| | `create_task` | タスク作成 | false | false |
| | `update_task` | タスク更新 | false | false |
| | `complete_task` | タスク完了 | false | false |
| | `delete_task` | タスク削除 | false | **true** |

### API パラメータマッピング

TickTick API は camelCase、mcpist は snake_case:

| mcpist パラメータ | TickTick API |
|-------------------|-------------|
| `project_id` | `projectId` |
| `task_id` | `id` |
| `start_date` | `startDate` |
| `due_date` | `dueDate` |
| `is_all_day` | `isAllDay` |
| `time_zone` | `timeZone` |
| `repeat_flag` | `repeatFlag` |
| `sort_order` | `sortOrder` |
| `view_mode` | `viewMode` |

### ビルドエラー修正

初回ビルドで `httpclient.DoJSON` が未定義エラー。
`DoJSON` はパッケージレベル関数ではなく `*Client` のメソッドだった。

```go
// 誤: httpclient.DoJSON(ctx, client, "GET", url, headers(ctx), nil)
// 正: client.DoJSON("GET", url, headers(ctx), nil)
```

戻り値も `(string, error)` ではなく `([]byte, error)` のため、
`httpclient.PrettyJSON(respBody)` で文字列変換が必要。

### テスト結果

| # | ツール | 結果 | 備考 |
|---|--------|------|------|
| 1 | `list_projects` | ✅ | 3プロジェクト（Daily Routine, Routine Tasks, Ad-hoc Tasks） |
| 2 | `get_project` | ✅ | Daily Routine 詳細取得 |
| 3 | `get_project_data` | ✅ | タスク・カラム含むデータ取得 |
| 4 | `create_project` | ✅ | "MCP Test Project" 作成 |
| 5 | `update_project` | ✅ | 名前・色変更反映 |
| 6 | `create_task` | ✅ | priority, due_date, content 含むタスク作成 |
| 7 | `get_task` | ✅ | タスク詳細取得 |
| 8 | `update_task` | ✅ | タイトル・優先度更新 |
| 9 | `complete_task` | ✅ | タスク完了 |
| 10 | `delete_task` | ✅ | タスク削除 |
| 11 | `delete_project` | ✅ | プロジェクト削除 |

**11/11 全ツール正常動作確認済み**

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/internal/modules/ticktick/module.go` | 新規作成（11ツール） |
| `apps/server/cmd/server/main.go` | RegisterModule(ticktick.New()) 追加 |
| `apps/server/cmd/tools-export/main.go` | RegisterModule(ticktick.New()) + displayName 追加 |
| `apps/console/src/app/api/oauth/ticktick/authorize/route.ts` | 新規作成（OAuth 2.0 認可） |
| `apps/console/src/app/api/oauth/ticktick/callback/route.ts` | 新規作成（Basic Auth トークン交換） |
| `apps/console/src/lib/oauth-apps.ts` | TickTick プロバイダー追加 |
| `apps/console/src/app/(console)/services/page.tsx` | ticktick authConfig 追加 |
| `apps/console/src/lib/module-data.ts` | ticktick アイコン追加 |
| `apps/console/src/lib/tools.json` | TickTick 11ツール追加 |

### コミット

| コミット | 内容 |
|----------|------|
| `2ab4c12` | feat(ticktick): add TickTick MCP module with OAuth 2.0 authentication |

---

## Observability 設計書作成 ✅

Sprint-007 Phase 1 の設計書作成（S7-001, S7-006）。

### 完了タスク

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S7-001 | Observability 設計書作成 | ✅ | `dsn-observability.md` v1.0 |
| S7-006 | Grafana ダッシュボード設計 | ✅ | Overview + Module Performance の2構成 |

### 設計書の主要決定事項

| 項目 | 決定 |
|------|------|
| 優先度 | 可用性監視 > 障害検知 > 運用可視化 > トレーサビリティ > 異常検知 |
| ログ基盤 | Grafana Loki (Cloud Free Tier) への HTTP Push のみ |
| 標準出力 | 起動・初期化・Loki送信失敗に限定（二重ログなし） |
| ラベル設計 | `module`, `status` をラベル、`tool`, `event` はデータフィールド |
| 外形監視 | Grafana Synthetic Monitoring (Free: 5 checks) で Worker + Console |
| ダッシュボード | Overview（可用性 + サマリ）、Module Performance の2つ |
| アラート | Worker ダウン (Critical)、Console ダウン (High)、エラー率 >50% (High) |

### レビュー指摘と対応

| 指摘 | 対応 |
|------|------|
| `tool`, `event` ラベルはカーディナリティが高い | データフィールドに移動 |
| `session_id` は不要（公開 API サービス） | セクションごと削除 |
| 二重ログ（slog + Loki）は冗長 | Loki Push のみに一本化 |
| 「ツール実行なし」アラートは不要 | 削除 |
| 攻撃対策よりも可用性が重要 | 優先度を再構成、Synthetic Monitoring 追加 |

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `docs/003_design/observability/dsn-observability.md` | 新規作成 |

---

## セキュリティ設計書作成 ✅

Sprint-007 Phase 2 の設計書作成（S7-010〜S7-015）。既存 v1.0 を全面改訂し v2.0 に。

### 完了タスク

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S7-010 | セキュリティ設計書作成 | ✅ | v1.0 → v2.0 全面改訂 |
| S7-011 | 認証・認可フロー整理 | ✅ | JWT 3段階フォールバック、API Key SHA-256、Gateway Secret |
| S7-012 | OAuth セキュリティ整理 | ✅ | 10プロバイダー、PKCE、state、トークンローテーション |
| S7-013 | データ保護整理 | ✅ | pgsodium TCE、RLS ポリシー一覧 |
| S7-014 | SSRF 対策整理 | ✅ | PostgreSQL localhost 禁止、SQLi 対策 |
| S7-015 | セキュリティチェックリスト作成 | ✅ | 新規モジュール追加時の確認項目 |

### v1.0 → v2.0 の主な変更

| 項目 | v1.0 | v2.0 |
|------|------|------|
| インフラ | Fly.io | Render / Koyeb |
| 認証仲介 | Token Broker | Supabase 直接 |
| API Key | 未記載 | SHA-256 ハッシュ保存、KV キャッシュ |
| OAuth | 基本的な記載のみ | 10プロバイダー詳細、PKCE、ローテーション |
| 権限ゲート | 未記載 | Filter → Gate → Detect 3層防御 |
| リスク許容 | 未記載 | JWT 漏洩、Rate Limit 誤差、DDoS を明記 |

### 文書間の整合性

Observability と Security の重複を解消:
- `invalid_gateway_secret` の実装コードを Observability から削除
- Observability → ログ仕様（フィールド・LogQL クエリ）に集中
- Security → 検証ロジック・発生原因を記載
- Observability から Security への参照リンクを設置

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `docs/003_design/security/dsn-security.md` | v1.0 → v2.0 全面改訂 |

---

## DAY023 サマリ

| 項目 | 内容 |
|------|------|
| 新規モジュール | TickTick（11ツール） |
| OAuth 対応 | Airtable (PKCE + Basic Auth), TickTick (Basic Auth) |
| テスト結果 | Airtable 11/11 ✅, TickTick 11/11 ✅ |
| 設計書 | Observability 設計書 (dsn-observability.md) 新規作成 |
| 設計書 | セキュリティ設計書 (dsn-security.md) v2.0 全面改訂 |
| Sprint-007 進捗 | Phase 1: S7-001,006 完了 / Phase 2: S7-010〜015 全完了 |
| 累計モジュール数 | 19 (Notion, GitHub, Jira, Confluence, Supabase, Airtable, Google Calendar, Google Docs, Google Drive, Google Sheets, Google Apps Script, Google Tasks, Microsoft To Do, PostgreSQL, Todoist, Trello, Asana, TickTick) |

---

## 次回の作業

1. S7-002〜005: Go Server の構造化ログ実装（設計書に基づく）
2. S7-006: Grafana Synthetic Monitoring・ダッシュボード設定
3. Sprint 007 Phase 3: 仕様書の実装追従更新 (S7-020〜026)
