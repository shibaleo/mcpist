# DAY034 作業ログ

## 日付

2026-02-21

---

## コミット一覧 (8件)

| # | ハッシュ | メッセージ |
|---|---------|-----------|
| 1 | e84f67e | docs: add day033 worklog, day034 plan, security report, and worker migration plan |
| 2 | 8760c08 | fix: use find-then-update for UpsertOAuthApp to avoid duplicate key violation |
| 3 | a5cd582 | fix: return per-tool rows from GetModuleConfig so console toggles reflect DB state |
| 4 | 3931055 | fix: align UpsertToolSettingsBody field names with Worker spec (enabled_tools/disabled_tools) |
| 5 | 8043210 | refactor: unify OpenAPI specs — Go Server spec as single source of truth |
| 6 | 5b0c65f | fix: add [build] command to wrangler.toml so Cloudflare runs generate:openapi before deploy |
| 7 | bc5c485 | fix: restore tools field in /v1/modules response by adding tools back to required |
| 8 | a0f0377 | fix: unmarshal module tools via json.RawMessage instead of jx.Raw |

### 前セッション (day034 後半)

| # | ハッシュ | メッセージ |
|---|---------|-----------|
| 9 | 2f5e4b9 | fix: force GORM to include enabled column in tool_settings UPSERT |
| 10 | b6354ce | fix: use raw SQL for tool_settings UPSERT to bypass GORM bool zero-value bug |
| 11 | afc9b66 | refactor: remove Console-side default tool settings logic, delegate to server |
| 12 | 6e2b4c1 | refactor: unify MCP schema language to English, remove server-side i18n logic |
| 13 | 1217b73 | chore: simplify Worker config, remove env.dev section |

### 本セッション (day034 続き)

| # | ハッシュ | メッセージ |
|---|---------|-----------|
| 14 | f9f870e | fix: use OAuth 1.0a standard field names for Trello credentials |
| 15 | c938136 | feat: redesign OAuth apps page as card grid, add Atlassian OAuth 2.0 support |
| 16 | 495a643 | fix: switch Confluence OAuth scopes from Classic to Granular |
| 17 | fbaacfa | fix: separate Jira and Confluence OAuth flows to prevent cross-saving |

---

## 実施内容

### 1. OpenAPI spec 統一 (8043210)

**課題:** Go Server spec と Worker spec が別々に管理されており、フィールド名の不一致（`enabled`/`disabled` vs `enabled_tools`/`disabled_tools`）でバグが発生。

**対応:**
- Go Server spec を single source of truth に決定
- Console の型生成ソースを Worker spec → Go Server spec に切り替え (`package.json` の `generate:api`)
- Worker spec を認証 + 独自エンドポイントのみに縮小
- ogen 再生成 + handler.go のフィールドマッピング修正

### 2. ツール設定トグルの修正 (a5cd582, 3931055, 2f5e4b9, b6354ce)

**課題:** Console のツール ON/OFF トグルが保存されない。

**原因の連鎖:**
1. `GetModuleConfig` がモジュール単位の1行しか返さず、個別ツールの設定が反映されない → per-tool rows を返すように修正
2. `UpsertToolSettingsBody` のフィールド名が spec と不一致 → Worker spec に合わせて修正
3. GORM の `enabled` カラム (bool) がゼロ値 (`false`) だと UPDATE 文に含まれない → `Select("enabled")` で明示指定
4. それでも GORM が bool ゼロ値を無視 → raw SQL (`INSERT ... ON CONFLICT DO UPDATE SET enabled = ?`) で解決

### 3. /v1/modules レスポンスの tools フィールド修復 (bc5c485, a0f0377)

**課題:** `/v1/modules` レスポンスに `tools` フィールドが含まれず、Console でツール一覧が表示されない。

**対応:**
- ogen spec で `tools` を required に復元
- `jx.Raw` → `json.RawMessage` に変更（jx が中間変換でデータを落としていた）

### 4. Console 側デフォルトツール設定ロジックの削除 (afc9b66)

**課題:** サーバー側でデフォルト設定を管理するようにしたため、Console の `saveDefaultToolSettings` / `isDefaultEnabled` が不要に。

**対応:**
- `saveDefaultToolSettingsAction` (server action) を削除
- `saveDefaultToolSettings` 関数を削除
- `isDefaultEnabled` 関数を削除
- 「デフォルトに戻す」ボタンを tools/page.tsx から削除
- 11 個の OAuth callback ファイルから `saveDefaultToolSettings` 呼び出しを削除

### 5. MCP スキーマ言語の英語統一 (6e2b4c1)

**課題:** MCP ツールスキーマがユーザーの言語設定に応じて日英で返されていたが、LLM向けスキーマは英語に統一したい。Console UI の多言語対応は維持。

**対応 (26 ファイル):**
- `Module` interface: `Description(lang string) string` → `Description() string`
- `modules.go`: `DynamicMetaTools` / `GetModuleSchemas` から `lang` パラメータ削除、日本語メタツール説明文 (~50行) を削除
- 全 20 モジュール: `Description()` が `moduleDescriptions["en-US"]` を直接返すように変更
- `handler.go`: `authCtx.Language` 参照を削除
- `middleware/authz.go`: `AuthContext` から `Language` フィールド削除
- `broker/user.go`: `UserContext` から `Language` フィールド削除
- `db/repo_user.go`: `MCPContext` から `Language` フィールド削除、settings.language 抽出ロジック削除
- `types.go`: `GetLocalizedText()` ヘルパー関数を削除

### 6. MCP サーバードメイン変更 (1217b73 + Cloudflare/Vercel 設定)

**変更:**
- `mcp.dev.mcpist.app` → `mcp.mcpist.app`
- Cloudflare Worker: `mcpist-gateway-dev` → `mcpist-gateway` にリネーム
- Cloudflare DNS: Worker ルート `mcp.mcpist.app` → `mcpist-gateway` を設定
- Cloudflare Workers Builds: デプロイコマンド `npx wrangler deploy -e dev` → `npx wrangler deploy`
- wrangler.toml: `[env.dev]` / `[env.production]` 削除、トップレベル `[vars]` に `APP_ENV = "mcpist"`
- `.mcp.json`: URL を `mcp.mcpist.app` に更新
- Vercel: 環境変数を `mcp.mcpist.app` に更新

---

## 本番環境ステータス

```
$ curl -s https://mcp.mcpist.app/health
{"status":"ok","backend":{"healthy":true,"statusCode":200,"latencyMs":246}}
```

MCP ツールスキーマが英語で返ることを確認済み（google_calendar モジュールで検証）。

### 7. Trello OAuth 1.0a フィールド名修正 (f9f870e)

**課題:** Trello の OAuth 1.0a 認証で、Console が `apiKey` / `apiToken` フィールドで保存していたが、Go Server 側は `access_token` / `consumer_key` を参照していたため、接続後にツールが動作しない。

**対応:**
- Console の Trello 認証フィールドを OAuth 1.0a 標準名 (`consumer_key`, `access_token`) に統一

### 8. OAuth Apps ページのカードグリッド化 + Atlassian OAuth 2.0 対応 (c938136)

**課題:**
- OAuth Apps 管理ページが単純なフォームリストで使いにくい
- Jira / Confluence が Basic 認証のみで、OAuth 2.0 に対応していない

**対応:**
- **OAuth Apps ページ再設計:** カードグリッドレイアウトに変更。設定済み / 未設定セクションに分離、クリックで Dialog 編集
- **Atlassian OAuth 2.0:** authorize / callback ルート実装済み。Jira + Confluence で OAuth 2.0 ログインが可能に
- **サービスページ:** Jira / Confluence を `authType: "oauth"` に変更、Basic 認証は `alternativeAuth` として残存
- **alternativeAuth 修正:** Dialog で `extraFields`（email, domain）が描画されない問題と、`handleConnectSubmit` が `alternativeAuth.authType` を検出しない問題を修正
- **Token Refresh:** `token.go` に Jira / Confluence の refresh config 追加 (`RotatesRefreshToken: true`)
- **Admin サイドバー:** `/admin` サブルートで両方のナビ項目がハイライトされるバグを修正

### 9. Confluence OAuth スコープを Classic → Granular に変更 (495a643)

**課題:** Atlassian が Confluence Cloud API の Classic スコープを完全に廃止。Classic スコープ (`read:confluence-content.all` 等) では v1 / v2 いずれの API エンドポイントでも 401 "scope does not match" が返る。v1 `/wiki/rest/api/space` は 410 Gone。

**調査:**
- `curl` で v1 (`/wiki/rest/api/content`, `/wiki/rest/api/search`) と v2 (`/wiki/api/v2/spaces`) を検証 → すべて 401
- Jira の Classic スコープは引き続き動作することを確認

**対応:**
- `authorize/route.ts`: Confluence スコープを Granular に変更 (`read:space:confluence`, `read:page:confluence`, `write:page:confluence`, `read:content-details:confluence`, `write:comment:confluence`, `read:comment:confluence`, `read:label:confluence`, `write:label:confluence`, `search:confluence`)
- `oauth/apps.ts`: `OAUTH_CONFIGS` の `atlassian` / `atlassian-confluence` のスコープも同様に更新
- Jira スコープ (Classic) はそのまま維持

### 10. Atlassian OAuth 同時保存問題の修正 (fbaacfa)

**課題:** Jira を OAuth 接続すると、Confluence にも同じクレデンシャルが保存されていた。ユーザーが Confluence のスコープに認可していないのに接続済みになる問題。

**原因:** `getOAuthProviderForService("jira")` が `OAUTH_CONFIGS` を順番にイテレートし、combined エントリ `atlassian` (`serviceId: "jira"`) に最初にマッチ → `module=atlassian` → callback で `["jira", "confluence"]` 両方に保存。

**対応:**
- `OAUTH_CONFIGS` から combined `atlassian` エントリを削除。`atlassian-jira` / `atlassian-confluence` のみ残す
- `authorize/route.ts`: combined `atlassian` スコープ定義を削除、`module` パラメータを必須化
- `callback/route.ts`: `modulesToSave` ループを削除、`moduleName` に直接保存。module 未指定時はエラー
- `getOAuthAuthorizationUrl`: `atlassian` ケースを削除

### 11. 全サービス動作確認

Granular スコープ適用後、MCP ツール経由で全サービスの動作を確認:

| サービス | 結果 |
|----------|------|
| Jira | OK (`list_projects` → 1件取得) |
| Confluence | OK (`list_spaces` → 1件, `get_pages` → 25件取得) |
| Google Calendar | OK (`list_events` → 2件取得) |
| Google Drive | OK (`list_files` → 多数取得) |
| Google Docs | OK (`list_comments` → 成功) |
| Google Sheets | OK (`search_spreadsheets` → 4件取得) |
| Google Tasks | OK (`list_task_lists` → 2件取得) |
| Google Apps Script | OK (`list_projects` → 1件取得) |

---

## DAY034 サマリ

| 項目 | 内容 |
|------|------|
| テーマ | spec 統一 + ツール設定バグ修正 + i18n 簡素化 + ドメイン整理 + OAuth 改善 + Atlassian OAuth 2.0 |
| コミット数 | 17 |
| 主な成果 | OpenAPI spec を Go Server に一本化、ツールトグル修正 (GORM bool問題)、MCP スキーマ英語統一 (26ファイル)、ドメイン mcp.mcpist.app に移行、OAuth Apps カードUI、Atlassian OAuth 2.0 対応、Confluence Granular スコープ移行、Jira/Confluence OAuth分離、全サービス動作確認完了 |

---

## 未完了・次ステップ

- Worker spec の縮小（共有スキーマの残骸削除）
- `go test ./...` のパス確認
