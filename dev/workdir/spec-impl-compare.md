# 仕様書・設計書と実装の比較（調査結果）

作成日: 2026-01-26

## 対象範囲
- 仕様/設計: `docs/specification/*`, `docs/design/dsn-tbl.md`
- 実装: `apps/server`, `apps/worker`, `apps/console`, `supabase/migrations`

## 主要な差分（仕様 ↔ 実装）

### 1) Token Vault API の形が仕様と異なる
- 仕様: Token Vault は **Edge Functions の HTTP API** として `POST /token-vault` を提供し、`Authorization: Bearer <publishable key>` でアクセスする想定。
  - `docs/specification/dtl-spc/itf-tvl.md`
- 実装: Go サーバは **Supabase RPC `get_module_token` を直接呼び出す**方式。
  - `apps/server/internal/store/token.go`
  - `supabase/migrations/00000000000010_fix_get_module_token.sql`
  - `supabase/migrations/00000000000011_fix_get_module_token_metadata.sql`
- さらに Console 側に **内部用 `/api/token-vault`** があるが、これは Next.js API であり仕様の `POST /token-vault` とは別物。
  - `apps/console/src/app/api/token-vault/route.ts`

### 2) MCP メタツール名の不一致
- 仕様: `get_module_schema`, `call`, `batch`
  - `docs/specification/spc-itf.md`
- 実装: `get_module_schema`, `run`, `batch`
  - `apps/server/internal/modules/modules.go`
  - `apps/server/internal/mcp/handler.go`

### 3) Long-lived Token のプレフィックス不一致
- 仕様: `mcpist_...`
  - `docs/specification/spc-itf.md`
- 実装: `mpt_...` を生成・検証
  - `supabase/migrations/00000000000006_rpc_console.sql`
  - `apps/worker/src/index.ts`

### 4) Auth Server の独立実装がない
- 仕様: `auth.mcpist.app` に OAuth/OIDC エンドポイントがある前提
  - `docs/specification/dtl-spc/idx-ept.md`
- 実装: Supabase Auth を直接使い、Worker が `/.well-known/*` のメタデータを返す構成
  - `apps/worker/src/index.ts`

### 5) API Gateway の Rate Limit 未実装
- 仕様: API Gateway がレート制限を担う
  - `docs/specification/spc-dsn.md`
- 実装: コメントで「削除済み」と明記
  - `apps/worker/src/index.ts`

### 6) JWT 検証項目の差
- 仕様: `aud`, `iss`, `exp` を含む厳密検証
  - `docs/specification/spc-itf.md`
- 実装: Supabase API(userinfo/user)確認＋JWKS検証の 3段構えだが、`aud` を明示チェックしていない
  - `apps/worker/src/index.ts`

### 7) MCP エラーコードの差
- 仕様: 2001–2005 の拡張エラーコード
  - `docs/specification/spc-itf.md`
- 実装: JSON-RPC 標準コードのみ
  - `apps/server/internal/mcp/types.go`

### 8) Console API 設計の差
- 仕様: `/api/dashboard` など REST API 前提
  - `docs/specification/spc-itf.md`
- 実装: Supabase RPC を直接呼び出す設計。該当 API ルートは未実装
  - `apps/console/src/lib/api-keys.ts`
  - `supabase/migrations/00000000000006_rpc_console.sql`

### 9) PSP Webhook 未実装
- 仕様: `/webhooks/stripe` を公開
  - `docs/specification/spc-itf.md`
- 実装: ルート／ハンドラ未確認
  - `apps/server`, `apps/worker`, `apps/console` 内に該当なし

### 10) Token Refresh の担当が仕様と異なる
- 仕様: Token Vault がトークンリフレッシュを担う前提
  - `docs/specification/spc-itf.md`
- 実装: 各モジュールが refresh を実装し、更新は RPC `update_module_token` で保存
  - `apps/server/internal/modules/google_calendar/module.go`
  - `apps/server/internal/modules/microsoft_todo/module.go`
  - `apps/server/internal/store/token.go`

## 仕様と一致している点
- **アーキテクチャ構成**（Go server / Worker / Next.js / Supabase）
  - `docs/specification/spc-dsn.md` ↔ `apps/server`, `apps/worker`, `apps/console`, `supabase/`
- **MCP Protocol バージョン 2025-03-26** を返す
  - `docs/specification/spc-itf.md` ↔ `apps/server/internal/mcp/handler.go`
- **/mcp の SSE + POST** を実装
  - `docs/specification/spc-itf.md` ↔ `apps/worker/src/index.ts`, `apps/server/internal/mcp/handler.go`
- **クレジット消費と冪等性**（credit_transactions + request_id）
  - `docs/specification/spc-tbl.md` ↔ `supabase/migrations/00000000000005_rpc_mcp_server.sql`

## DB 設計差分（dsn-tbl.md ↔ migrations）

### 追加されているテーブル / 概念
- `mcpist.service_tokens`（Vault との紐付け）
  - `supabase/migrations/00000000000002_tables.sql`
- `mcpist.oauth_apps`（OAuth app 管理）
  - `supabase/migrations/00000000000014_oauth_apps.sql`

### dsn-tbl には無い列が追加
- `api_keys.key_prefix`, `last_used_at`, `revoked_at`
- `credit_transactions.credit_type`
  - `supabase/migrations/00000000000002_tables.sql`

### dsn-tbl にあるが実装に差分がある点
- dsn-tbl の api_keys は `key_hash` と `name` のみ想定だが、実装はプレフィックス/失効/無効化などの運用列が追加。
  - `docs/design/dsn-tbl.md` ↔ `supabase/migrations/00000000000002_tables.sql`

## 仕様外の追加実装（例）
- モジュール追加: Airtable など（仕様に記載なし）
  - `apps/server/cmd/server/main.go`, `apps/server/internal/modules/*`
- Console API: `validate-token`, OAuth Google/Microsoft の API 群
  - `apps/console/src/app/api/validate-token/route.ts`
  - `apps/console/src/app/api/oauth/*/route.ts`

## 参照ファイル
- `docs/specification/spc-sys.md`
- `docs/specification/spc-itr.md`
- `docs/specification/spc-itf.md`
- `docs/specification/spc-tbl.md`
- `docs/specification/dtl-spc/idx-ept.md`
- `docs/specification/dtl-spc/itf-tvl.md`
- `docs/design/dsn-tbl.md`
- `apps/server/cmd/server/main.go`
- `apps/server/internal/mcp/handler.go`
- `apps/server/internal/mcp/types.go`
- `apps/server/internal/modules/modules.go`
- `apps/server/internal/store/token.go`
- `apps/worker/src/index.ts`
- `apps/console/src/app/api/token-vault/route.ts`
- `apps/console/src/app/api/validate-token/route.ts`
- `supabase/migrations/00000000000002_tables.sql`
- `supabase/migrations/00000000000005_rpc_mcp_server.sql`
- `supabase/migrations/00000000000006_rpc_console.sql`
- `supabase/migrations/00000000000010_fix_get_module_token.sql`
- `supabase/migrations/00000000000011_fix_get_module_token_metadata.sql`
- `supabase/migrations/00000000000014_oauth_apps.sql`
