# DAY030 作業ログ

## 日付

2026-02-15

---

## コミット一覧

| # | ハッシュ | メッセージ |
|---|---------|-----------|
| 1 | 29f787a | refactor(server): extract SSE/Inline transport from handler into middleware |
| 2 | 20ed596 | chore: remove unused Dockerfiles and .dockerignore files |
| 3 | 1b0c50b | refactor(worker): decouple PostgREST RPC from Supabase-specific headers |
| 4 | 6e87e26 | refactor(server): decouple PostgREST RPC from Supabase-specific headers |
| 5 | 6f26d41 | refactor(server): decouple PostgREST RPC from Supabase-specific headers |
| 6 | 4dc3703 | fix(server): remove duplicate lines in user.go causing build failure |
| 7 | cd70a3c | refactor(server): consolidate RPC design and remove sync_modules |
| 8 | 6f89327 | fix(server): add apikey header to all PostgREST RPC calls |
| 9 | (未コミット) | feat(console): replace static tools.json with dynamic DB fetch via list_modules_with_tools RPC |

---

## 計画との差分

当初計画は Phase 1a (OAuth 2.1 Server) だったが、Claude 認可フローが原因不明で復旧したため Phase 1 を sprint010-backlog に移動。代わりにシステム構成図の更新と Go Server のリファクタリングを実施。

---

## 実施内容

### 1. Sprint 010 バックログ作成

Sprint 009 バックログを引き継ぎ、Sprint 010 の状況を反映した `sprint010-backlog.md` を新規作成。

- Phase 1 (OAuth Server) をバックログに格下げ
- Claude 認可フロー一時障害の原因調査を優先度低で追加

### 2. システム構成図 (grh-componet-interactions.canvas) 更新

#### 第1回（リファクタリング前）

| 変更 | 内容 |
|------|------|
| MOD | モジュール数の記述を削除（変動が激しいため） |
| CON | "クレジット課金" → "プラン管理" |
| AMW | "クレジット消費" → "日次制限"、SSE/Inline トランスポート + セッション管理を追加 |
| HDL | エンドポイント記述 → 責務記述に変更（JSON-RPC ルーティング、ツールフィルタリング、フォーマット変換、バッチ検証） |
| EXT | 個別サービス列挙 → "各種SaaS API" に簡略化 |
| OBS | Grafana Loki を明記 |
| DST | Token Vault の記述を DST に統合 |
| BRK | Broker コンポーネントを新規追加 |
| グループ | "MCP サーバー" グループで AMW/HDL/BRK/MOD を囲む |

#### 第2回（実コードとの照合・修正）

実装コードを全調査し、構成図の記述を実態に合わせて修正。

| 変更 | 内容 |
|------|------|
| GWY | "Rate Limit / Burst制限" 削除（Worker から削除済み、サーバー側に移行）、"Loki ログ送信" 追加 |
| AUS | "OAuth 2.1準拠" → "OAuth 2.0 (Supabase Auth)" |
| AMW | "パニックリカバリ"、"レートリミット（per-user）" 追加（recovery.go, ratelimit.go を反映） |
| MOD | "トークン取得" → "パラメータバリデーション"（トークン管理は BRK の責務） |
| DST | "課金情報 / クレジット残高" → "プラン・日次使用量"、"pgsodium TCE" 明記、"プロンプトテンプレート" 追加 |
| CON | "プロンプトテンプレート管理"、"MCP接続情報表示" 追加 |

### 3. トランスポート層リファクタリング (Go Server)

handler.go に混在していた SSE/Inline トランスポート層とビジネスロジックを分離。

#### 新規ファイル

| ファイル | 内容 |
|----------|------|
| `internal/jsonrpc/types.go` | JSON-RPC 2.0 型 (Request, Response, Error) とエラーコード定数を共通パッケージに切り出し |
| `internal/middleware/transport.go` | SSE/Inline トランスポート。`RequestProcessor` interface で handler に委譲 |

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `internal/mcp/types.go` | JSON-RPC 型を `jsonrpc` パッケージから type alias で再エクスポート |
| `internal/mcp/handler.go` | Session, sessions, mu, ServeHTTP, handleSSE, handleMessage 等を削除。`processRequest` → `ProcessRequest` に公開 (568 → 396 行, -172 行) |
| `cmd/server/main.go` | ミドルウェアチェーンに Transport を追加 |

#### 設計判断

- **循環参照回避**: `mcp` → `middleware` の import が既存のため、Request/Response/Error を `internal/jsonrpc` パッケージに切り出し
- **Transport は末端**: `func(next) http.Handler` ではなく、チェーン最後で `RequestProcessor` interface を受け取る `http.Handler`
- **Handler は http.Handler を実装しなくなる**: `ProcessRequest` メソッドのみ公開

#### ミドルウェアチェーン (変更後)

```
Before: Recovery → Authorize → RateLimit → MCPHandler (transport + logic 混在)
After:  Recovery → Authorize → RateLimit → Transport → MCPHandler (logic のみ)
```

### 4. PostgREST RPC の Supabase 依存解消

Worker・Go Server の PostgREST RPC 呼び出しから Supabase 固有ヘッダー (`apikey`) を削除し、汎用的な `Authorization: Bearer` のみの構成に変更。Neon 移行を見据えた設計判断。

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/worker/src/index.ts` | `apikey` ヘッダー削除、`Authorization: Bearer` のみに |
| `apps/server/internal/broker/user.go` | 同上 |
| `apps/server/internal/broker/token.go` | 同上 |

#### 環境変数の変更

| 環境 | 旧 | 新 |
|------|----|----|
| Render / Worker | `SUPABASE_URL`, `SUPABASE_SECRET_KEY` | `POSTGREST_URL`, `POSTGREST_API_KEY` |

### 5. 不要ファイル削除

| ファイル | 理由 |
|----------|------|
| `apps/server/Dockerfile.dev` | 未使用 |
| `apps/server/.dockerignore` | 未使用 |
| `apps/worker/Dockerfile` | Worker は Cloudflare Workers にデプロイ、Docker 不要 |

### 6. RPC 設計見直しと統合

全 9 RPC を精査し、統合・廃止・インターフェース改善を実施。

#### sync_modules 廃止

- モジュール一覧の DB 登録はランタイム Broker の責務ではなく、マイグレーションの責務
- `apps/server/internal/broker/module.go` 削除
- `cmd/server/main.go` から起動時呼び出し削除
- Render で 401 エラーが出ていた（GRANT EXECUTE 不足）が、修正ではなく廃止で対応

#### プロンプト RPC 統合

`list_user_prompts` + `get_user_prompt_by_name` → `get_user_prompts` (統合)

| 変更 | 内容 |
|------|------|
| SQL | `20260215100000_merge_prompt_rpcs.sql` 新規作成。旧関数 DROP + 統合関数 CREATE (optional `p_prompt_name DEFAULT NULL`) |
| Go | `fetchUserPrompts(userID, promptName)` 共通メソッド追加。公開 API は維持 |

#### インターフェース改善

全 RPC 呼び出しの JSON ボディ構築を `fmt.Sprintf` → `json.Marshal` に変更（5 箇所、user.go + token.go）。

### 7. Supabase Kong 認証対応 (apikey ヘッダー復元)

PostgREST RPC の Supabase 依存解消（セクション4）で `apikey` ヘッダーを削除したが、Supabase Kong が `apikey` ヘッダーを必須とすることが判明。Neon 移行前のため `apikey` ヘッダーを復元。

#### 根本原因の調査過程

1. Render デプロイ後に `get_user_context failed: status 401` が継続
2. `sb_secret_` キーを `service_role` JWT (`eyJ...`) に変更 → 改善せず
3. curl で直接 PostgREST を叩いて `No API key found in request` を確認
4. Supabase Kong は **`apikey` + `Authorization: Bearer` の両方が必須**と判明

#### Supabase vs Neon の認証方式

| | Supabase | Neon / セルフホスト PostgREST |
|---|---|---|
| `apikey` ヘッダー | 必須 (Kong が要求) | 不要 |
| `Authorization: Bearer` | 必須 (PostgREST が要求) | 必須 |

#### 環境変数の変更

| 環境 | 変更 |
|------|------|
| Render `POSTGREST_API_KEY` | `sb_secret_...` → `eyJ...` (service_role JWT) |
| Worker `POSTGREST_API_KEY` | 同上 (`wrangler secret put --env dev`) |

### 8. RPC 設計図 (docs/graph/grh-rpc-design.canvas) 作成

Obsidian Canvas 形式で全コンポーネントと 7 RPC の関係を図示。

### 9. Console の tools.json 依存排除 — DB からモジュール+ツール情報を動的取得

Console がビルド時生成の `tools.json` に依存していた構造を廃止し、DB から `list_modules_with_tools` RPC で動的取得する方式に移行。

#### マイグレーション (`20260216100000_list_modules_with_tools.sql`)

| 変更 | 内容 |
|------|------|
| ALTER TABLE | `modules` に `descriptions JSONB DEFAULT '{}'` カラム追加 |
| sync_modules 更新 | `descriptions` も UPSERT するよう拡張 |
| 新 RPC | `list_modules_with_tools()` — `id, name, status, descriptions, tools` を返却 |
| GRANT | `authenticated` + `anon` ロールに EXECUTE 権限 |

#### Go Server 変更

| ファイル | 変更内容 |
|----------|----------|
| `internal/broker/user.go` | `SyncModuleEntry` に `Descriptions map[string]string` フィールド追加 |
| `cmd/server/main.go` | `buildSyncEntries` で `m.Descriptions()` を含めて同期 |

#### Console 変更 (`module-data.ts` 書き換え)

| Before | After |
|--------|-------|
| `import toolsData from "./tools.json"` (静的) | `supabase.rpc("list_modules_with_tools")` (動的 DB フェッチ) |
| `export const modules` (同期) | `export async function getModules()` (非同期、シングルトンキャッシュ) |

- `getModule()`, `getModuleTools()` も async 化
- `ModuleRow` interface + `(supabase.rpc as any)` で未生成型を回避
- `moduleDisplayNames` マップで DB の lowercase ID → UI 表示名に変換

#### コンシューマー変更 (4 ファイル)

| ファイル | 変更内容 |
|----------|----------|
| `(console)/tools/page.tsx` | `modules` を useState + `getModules()` に変更 |
| `(console)/services/page.tsx` | 同上 |
| `page.tsx` (ランディング) | `services` を useState + useEffect に変更 |
| `(onboarding)/onboarding/page.tsx` | `allModules` state + useEffect に変更 |
| `lib/tool-settings.ts` | `getModule()` 呼び出しに `await` 追加 |

#### 削除ファイル

| ファイル | 理由 |
|----------|------|
| `apps/console/src/lib/tools.json` | DB フェッチに置き換え |
| `apps/server/cmd/tools-export/main.go` | tools.json 生成 CLI、不要に |
| `apps/server/cmd/tools-export/tools_test.go` | 同上 |

#### ビルド検証

- `go build ./...` — pass
- `tsc --noEmit` — pass (型キャスト対応後)
- `next build` — pass

#### デプロイ後の DB 検証

| 項目 | 結果 |
|------|------|
| modules テーブル | 20 モジュール、全て `has_descriptions: true` |
| descriptions 内容 | `en-US` / `ja-JP` の2言語が正常格納 |
| `list_modules_with_tools()` RPC | id, name, status, descriptions, tools を正常返却 |
| ツール数 | github: 26, google_sheets: 27, asana: 23 等、正常 |

### 10. 設計図の更新

#### grh-rpc-design.canvas

| 変更 | 内容 |
|------|------|
| summary | RPC 数 7 → 9 (`sync_modules`, `list_modules_with_tools` 追加) |
| modules ノード | `tools (JSONB)`, `descriptions (JSONB)` 追加 |
| 新ノード | `rpc8` (sync_modules), `rpc9` (list_modules_with_tools) |
| 新ノード | `serverboot` (Server 起動), `console` (Console) |
| 新エッジ | serverboot→rpc8, console→rpc9, rpc8→modules, rpc9→modules |

#### grh-table-design.canvas

| 変更 | 内容 |
|------|------|
| modules ノード | `tools (JSONB)`, `descriptions (JSONB)` カラム追加 |

---

## ビルド・テスト・デプロイ結果

- `go build ./...` — pass (全コミットでビルド確認済み)
- `tsc --noEmit` — pass (Console)
- `next build` — pass (Console)
- Render auto-deploy — 計 6 回デプロイ (最終: tools.json 廃止 + descriptions 同期)
- Worker deploy — `wrangler deploy --env dev` 成功

### デプロイ経緯

| # | コミット | 結果 | 備考 |
|---|---------|------|------|
| 1 | 29f787a | live | トランスポート分離 |
| 2 | 6f26d41 | live | apikey ヘッダー削除 → Render 401 発生 |
| 3 | 4dc3703 | live | user.go 重複修正 (Koyeb ビルド復旧) |
| 4 | cd70a3c | live | RPC 統合。401 継続 |
| 5 | 6f89327 | live | apikey ヘッダー復元 → **401 解消** |

### 障害と対応

| 障害 | 原因 | 対応 |
|------|------|------|
| Koyeb Service degraded | user.go の重複行 (amend 操作の副作用) | ファイル全体を書き直し |
| Render `sync_modules` 401 | GRANT EXECUTE 不足 | RPC 自体を廃止 |
| Render `get_user_context` 401 | apikey ヘッダー削除 + sb_secret キーの仕様差 | apikey ヘッダー復元 + service_role JWT に変更 |

### 本番動作検証

| テスト | 結果 | 備考 |
|--------|------|------|
| `/health` | OK | `status: ok`, `db: ok` |
| MCP initialize | OK | Claude Code から `mcpist-dev` connected |
| `tools/list` | OK | ツール一覧正常返却 |
| `prompts/list` | OK | 統合後の `get_user_prompts` RPC 動作確認 |
| `postgresql:list_schemas` | OK | MCP ツール実行成功 (Neon DB 接続確認) |

---

## 変更ファイル一覧

| ファイル | 種別 |
|----------|------|
| `apps/server/cmd/server/main.go` | 変更 |
| `apps/server/internal/broker/module.go` | 削除 |
| `apps/server/internal/broker/user.go` | 変更 |
| `apps/server/internal/broker/token.go` | 変更 |
| `apps/server/internal/jsonrpc/types.go` | 新規 |
| `apps/server/internal/mcp/handler.go` | 変更 |
| `apps/server/internal/mcp/types.go` | 変更 |
| `apps/server/internal/middleware/transport.go` | 新規 |
| `apps/server/Dockerfile.dev` | 削除 |
| `apps/server/.dockerignore` | 削除 |
| `apps/worker/Dockerfile` | 削除 |
| `apps/worker/src/index.ts` | 変更 |
| `supabase/migrations/20260215100000_merge_prompt_rpcs.sql` | 新規 |
| `supabase/migrations/20260216100000_list_modules_with_tools.sql` | 新規 |
| `apps/console/src/lib/module-data.ts` | 変更 |
| `apps/console/src/lib/tool-settings.ts` | 変更 |
| `apps/console/src/lib/tools.json` | 削除 |
| `apps/console/src/app/(console)/tools/page.tsx` | 変更 |
| `apps/console/src/app/(console)/services/page.tsx` | 変更 |
| `apps/console/src/app/page.tsx` | 変更 |
| `apps/console/src/app/(onboarding)/onboarding/page.tsx` | 変更 |
| `apps/server/cmd/tools-export/main.go` | 削除 |
| `apps/server/cmd/tools-export/tools_test.go` | 削除 |

## 未コミット

| ファイル | 内容 |
|----------|------|
| `docs/graph/grh-componet-interactions.canvas` | 構成図更新 |
| `docs/graph/grh-rpc-design.canvas` | RPC 設計図 (sync_modules, list_modules_with_tools 追加) |
| `docs/graph/grh-table-design.canvas` | テーブル設計図 (modules に tools, descriptions 追加) |
| `dev/sprint/sprint010-backlog.md` | 新規 (untracked) |
| `dev/diary/day030-worklog.md` | 本ファイル |
| Console + Server 変更 (12 ファイル) | tools.json 廃止 + DB フェッチ移行 (ステージ済み) |

---

## DAY030 サマリ

| 項目 | 内容 |
|------|------|
| テーマ | トランスポート分離 + PostgREST RPC 設計見直し + tools.json 廃止 |
| 対応スプリント | Sprint 010 (計画変更: OAuth → リファクタリング + DB 移行) |
| コミット数 | 8 (+1 未コミット) |
| RPC 数 | 9 → 7 → 9 (sync_modules 復活 + list_modules_with_tools 追加、プロンプト 2→1 統合) |
| handler.go | 568 → 396 行 (-30%) |
| 新規パッケージ | `internal/jsonrpc` (型の共通化)、`middleware/transport.go` (トランスポート分離) |
| 削除 | `tools.json` (静的ファイル)、`tools-export` CLI (生成ツール) |
| 構成図修正 | 3 回実施 (構成図×2、RPC 設計図、テーブル設計図) |
| デプロイ | Render 6 回、Worker 2 回。最終的に全 MCP サーバー connected |
| 主な成果 | Console の DB 直接取得化、RPC 統合・廃止、json.Marshal 化、apikey/JWT 認証整理 |
| 学び | Supabase Kong は apikey + Authorization 両方必須。sb_secret は JWT ではない |
