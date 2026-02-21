# DAY031 バックログ

## 日付

2026-02-17

---

## 完了タスク (Sprint 010 実績)

### 1. ~~Worker API RESTful 再設計~~ ✅ 完了

**実績:** DAY030-031 で実施。

- `POST /v1/rpc/{name}` 単一 RPC エンドポイントを廃止
- Hono ルート定義で 24 の RESTful エンドポイントを個別実装
- OpenAPI 3.1 spec (`openapi.yaml`) を全パス + レスポンス型で更新
- `rpc-proxy.ts` → `postgrest.ts` + `routes/*.ts` に分離

### 2. ~~Console openapi-fetch 導入~~ ✅ 完了

**実績:** DAY031 で実施。

- `openapi-typescript` で `openapi.yaml` → `types.ts` 自動生成
- `openapi-fetch` の `createClient<paths>()` で型安全クライアント作成
- 全 11 consumer ファイルから `as unknown as` キャスト 26 箇所を除去
- domain wrapper 内の手動型定義を OpenAPI 生成型のエイリアスに置換

### 3. ~~Stripe checkout/portal の Supabase 依存除去~~ ✅ 完了

**実績:** DAY031 で実施。

- `get_user_context` RPC に `user_id`, `email` を追加 (マイグレーション 2 件)
- checkout/portal ルートから `createClient` (Supabase) を除去
- Worker API (`/v1/user/context`) 経由でユーザー情報取得に統一

### 4. ~~Worker エラーハンドリング改善~~ ✅ 完了

**実績:** DAY031 で実施。

- グローバルエラーハンドラー (`app.onError`) 追加
- `callPostgRESTRpc` のエラーメッセージに PostgREST レスポンスボディを含める

---

## 未完了タスク

### 5. Stripe Webhook の Worker 移行

**優先度:** 高
**計画:** [day032-plan-stripe-webhook-to-worker.md](day032-plan-stripe-webhook-to-worker.md)

Console の `POST /api/stripe/webhook` を Worker の `POST /v1/stripe/webhook` に移行。
Webhook にはユーザーセッションがなく `rpcDirect` で PostgREST 直接呼び出しをしている。
Worker で完結させることで `rpcDirect` と旧 `worker-client.ts` を廃止できる。

| タスク | 状態 |
|--------|------|
| Worker に `POST /v1/stripe/webhook` 追加 | 未着手 |
| `STRIPE_WEBHOOK_SECRET` を Worker 環境変数に追加 | 未着手 |
| OpenAPI spec 更新 | 未着手 |
| Console の webhook ルート削除 | 未着手 |
| `rpcDirect` / 旧 `worker-client.ts` のクリーンアップ | 未着手 |
| Stripe ダッシュボードで Webhook URL 変更 | 未着手 |

### 6. RPC 再設計: `get_user_context` の責務分離

**優先度:** 高

`get_user_context` は現在 Console と Go Server の両方が利用しているが、
必要なフィールドが異なる。今回 `user_id`/`email` を追加したことで混在が深まった。

#### 現状の問題

| フィールド | Console | Go Server |
|-----------|---------|-----------|
| user_id | ✅ (Stripe 用) | ❌ 不要 (リクエストで既知) |
| email | ✅ (Stripe 用) | ❌ 不要 |
| account_status | ✅ | ✅ |
| plan_id | ✅ | ✅ |
| daily_used / daily_limit | ✅ | ✅ |
| enabled_modules | ✅ | ✅ |
| enabled_tools | ✅ | ✅ |
| module_descriptions | ✅ | ❌ 不要 |
| role | ✅ (admin 判定) | ❌ 不要 |
| settings | ✅ | ❌ 不要 |
| display_name | ✅ | ❌ 不要 |
| connected_count | ✅ | ❌ 不要 |
| language | ✅ | ✅ |

#### 分離案

**案 A: 2 関数に分割**

```sql
-- Go Server 用 (軽量・高頻度呼び出し)
get_server_context(p_user_id)
  → account_status, plan_id, daily_used, daily_limit,
    enabled_modules, enabled_tools, language

-- Console 用 (フル情報)
get_user_context(p_user_id)
  → 上記 + user_id, email, role, settings, display_name,
    connected_count, module_descriptions
```

**案 B: 現状維持 + Go Server 側で不要フィールド無視**

Go Server は `get_user_context` をそのまま呼び、不要フィールドを JSON デコード時に無視する。
DB 側の変更なし。ただしクエリコストは最適化されない。

| タスク | 状態 |
|--------|------|
| Go Server の利用フィールドを正確に棚卸し | 未着手 |
| 分離の設計判断 (案 A or B) | 未着手 |
| マイグレーション作成 (案 A の場合) | 未着手 |
| Go Server の broker/user.go 更新 (案 A の場合) | 未着手 |

### 7. OAuth authorize ルートの Supabase 依存

**優先度:** 中

11 個の OAuth authorize ルートが `createClient` (Supabase) を使用:
`asana`, `github`, `microsoft`, `trello`, `google`, `ticktick`, `dropbox`,
`todoist`, `atlassian`, `airtable`, `notion`

これらは OAuth フロー開始時のユーザー認証に Supabase Auth を使っている。
Worker API 経由に統一するか、Sprint 010 Phase 1 (自前 OAuth Server) の
完了後に対応するかの判断が必要。

### 8. Sprint 010 Phase 1: OAuth 2.1 Server

**優先度:** 中 (Sprint 010 計画の最優先だが、RESTful 移行を優先した)

| ID | タスク | 状態 |
|----|--------|------|
| S10-001 | `@cloudflare/workers-oauth-provider` 導入 | 未着手 |
| S10-002 | OAuth 2.1 Server エンドポイント実装 | 未着手 |
| S10-003〜005 | Token storage, User identity, Consent redirect | 未着手 |
| S10-006〜008 | メタデータ + JWT 検証更新 | 未着手 |
| S10-010〜011 | Consent page 改修 | 未着手 |
| S10-015〜016 | 互換性確認 + E2E テスト | 未着手 |

### 9. DB 移行: PostgREST 廃止 + Worker アプリケーションサーバー化

**優先度:** 中
**設計判断:** PostgREST 依存を残すと PG ホスティング選択肢が制限される。
Worker に Drizzle ORM を導入し、全 DB アクセスを Worker REST API に一本化する。

#### 方針

```
移行前: Console → Worker → PostgREST → PG (Supabase)
        Go      →           PostgREST → PG (Supabase)

移行後: Console → Worker (Drizzle) → PG (どこでも)
        Go      → Worker (REST API) → PG (どこでも)
```

- PostgREST を廃止、SQL RPC 関数を廃止
- Worker が Drizzle でクエリを実行するアプリケーションサーバーになる
- Go Server は Worker REST API 経由で DB アクセス（`oapi-codegen` で型生成）
- Go Server のキャッシュ TTL を 30秒 → 5分 に延長しレイテンシ影響を最小化
- PG さえあればどのホスティングにも移れる（Neon, Supabase, セルフホスト）

#### 影響整理

| 項目 | 影響 |
|------|------|
| Worker REST ルーティング (完了済み) | そのまま活きる |
| OpenAPI spec (完了済み) | そのまま活きる |
| Console openapi-fetch (完了済み) | 変更なし |
| Worker 内部実装 | `forwardToPostgREST` → Drizzle クエリに置換 |
| SQL RPC 関数 (34件) | Drizzle クエリに移植後、廃止 |
| Go Server broker/ | PostgREST 直接呼び出し → Worker REST API + `oapi-codegen` 生成型 |
| マイグレーション管理 | Supabase CLI → `drizzle-kit` |
| #6 RPC 分離 | SQL RPC の分離ではなく Worker エンドポイント設計の問題になる (`GET /v1/server/context` vs `GET /v1/user/context`) |
| #12 spec 同期 | SQL→spec の手動同期問題が消える (Drizzle スキーマ→Worker 実装→spec が同一コードベース) |
| #13 Go 型安全性 | `oapi-codegen` で Worker OpenAPI spec から Go 型自動生成。完全に型安全 |

#### タスク

| タスク | 状態 |
|--------|------|
| Drizzle スキーマ定義 (既存テーブルから移植) | 未着手 |
| Worker ルートハンドラーを Drizzle クエリに置換 | 未着手 |
| Go Server broker/ を Worker REST API クライアントに置換 | 未着手 |
| Go Server キャッシュ TTL 延長 (30秒 → 5分) | 未着手 |
| `oapi-codegen` で Go クライアント型生成 | 未着手 |
| PG 移行先の選定 (Neon / その他) | 未着手 |
| データ移行 (credential 再暗号化含む) | 未着手 |
| PostgREST + SQL RPC 関数の廃止 | 未着手 |
| `drizzle-kit` によるマイグレーション管理移行 | 未着手 |

### 10. テスト基盤

**優先度:** 中

| ID | タスク | 状態 |
|----|--------|------|
| S10-040 | authz middleware ユニットテスト | 未着手 |
| S10-041 | broker/user.go ユニットテスト | 未着手 |
| S10-042 | broker/retry.go ユニットテスト | 未着手 |
| S10-043 | CI トリガーを push/PR に変更 | 未着手 |

### 11. 旧 `worker-client.ts` の完全除去

**優先度:** 高 (#5 完了後)

`worker-client.ts` のエクスポートは `rpcDirect` と `PostgRESTError` のみ。
利用箇所は `stripe/webhook/route.ts` の 1 ファイルだけ。
#5 (Webhook Worker 移行) が完了すればファイルごと削除できる。

Console の PostgREST 直接接続に必要な環境変数 (`POSTGREST_URL`, `POSTGREST_API_KEY`) も
Console から除去できる可能性がある。

### 12. OpenAPI spec と生成型の同期保証

**優先度:** 中 → **#9 完了で解消**

現在の問題: SQL RPC 定義 → openapi.yaml が手動同期で、spec が間違っていてもビルドは通る。

**#9 (Worker Drizzle 化) が完了すれば解消する:**
Drizzle スキーマ → Worker 実装 → openapi.yaml が同一コードベース内で完結し、
SQL RPC との手動同期が不要になる。それまでは CI で `generate:api` → `git diff --exit-code`
で `types.ts` の同期漏れを検出する案 A が現実的。

### 13. Go Server の型安全性

**優先度:** 中 → **#9 完了で解消**

現在の問題: Go Server が PostgREST を直接呼び、手動 struct でデコード。
スキーマ変更時に silent failure になる。

**#9 で Go Server を Worker REST API 経由に統一すれば解消する:**
Worker の OpenAPI spec から `oapi-codegen` で Go クライアント型を自動生成。
spec とコードの不整合はコンパイル時に検出される。

### 14. 仕様書の残課題 (旧 #11)

**優先度:** 低

| タスク | 由来 |
|--------|------|
| spc-dsn.md Rate Limit 記述更新 | S7-020 |
| spc-itf.md JWT `aud` チェック要件整理 | S7-021 |
| spc-itf.md MCP 拡張エラーコード整理 | S7-022 |
| spc-itf.md Console API 設計更新 | S7-023 |
| spc-itf.md PSP Webhook 仕様整理 | S7-024 |
| credentials JSON 構造の整理 | S7-025 |
| credit model → subscription model に更新 | S007 |
| dsn-modules.md の 3 層アーキテクチャ整合 | S006 |

---

## 優先度

| 優先度 | タスク | 備考 |
|--------|--------|------|
| 高 | #5 Stripe Webhook の Worker 移行 | |
| 高 | #6 `get_user_context` の責務分離 | |
| 高 | #11 旧 `worker-client.ts` の完全除去 | #5 完了後 |
| 中 | #7 OAuth authorize ルートの Supabase 依存 | |
| 中 | #8 OAuth 2.1 Server | |
| 中 | #9 PostgREST 廃止 + Worker Drizzle 化 + DB 移行 | 最大タスク |
| 中 | #10 テスト基盤 | |
| 中→解消 | #12 OpenAPI spec 同期 | #9 完了で解消 |
| 中→解消 | #13 Go Server 型安全性 | #9 完了で解消 |
| 低 | #14 仕様書の残課題 | |

---

## 参考

- [day032-plan-stripe-webhook-to-worker.md](day032-plan-stripe-webhook-to-worker.md) - Stripe Webhook 移行計画
- [sprint010-backlog.md](../sprint/sprint010-backlog.md) - Sprint 010 バックログ
- [sprint010-plan.md](sprint010-plan.md) - Sprint 010 計画
