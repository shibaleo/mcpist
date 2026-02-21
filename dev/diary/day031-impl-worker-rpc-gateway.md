# Console RPC 移行 v2: Worker Gateway + openapi-fetch

## 日付
2026-02-17

---

## 背景

前回の移行計画 (day031-impl-console-rpc-migration.md) で Phase 0-1 は完了済み:
- PostgREST クライアント (`lib/postgrest.ts`) 作成済み
- SQL リネーム + `auth.uid()` → `p_user_id` 完了 (34 RPC)
- Console lib ファイル 8本が `getUserId()` + `rpc(name, {p_user_id, ...})` パターンに移行済み

**新方針**: Console → Worker → PostgREST に変更。Console は `p_user_id` を知らない。

---

## アーキテクチャ

```
Console (Browser/Server)
  ↓  fetch("/rpc/list_api_keys", { params }) + Authorization: Bearer JWT
Worker (Cloudflare Edge)
  ↓  JWT 検証 → user_id 抽出 → { p_user_id: user_id, ...params }
PostgREST (service_role)
  ↓  SQL 実行
DB
```

### RPC 分類

| 種別 | 認証 | p_user_id | 例 |
|------|------|-----------|---|
| User-scoped (21) | JWT 必須 | Worker が注入 | `list_api_keys`, `upsert_prompt` |
| Public (3) | 不要 | なし | `list_modules_with_tools`, `get_oauth_app_credentials` |
| Admin (4) | JWT + role=admin | Worker が注入 + role チェック | `list_oauth_apps`, `upsert_oauth_app` |
| Webhook (3) | なし (Stripe 署名) | Next.js が直接渡す | `activate_subscription` |

---

## 実施フェーズ

### Phase A: Worker に `/rpc/*` ルート追加
### Phase B: Console クライアント差し替え (postgrest.ts → Worker 向き)
### Phase C: API Route Handler 移行 (OAuth 22本 + API 5本)
### Phase D: module-data.ts 移行
### Phase E: クリーンアップ

---

## openapi-fetch 適用

| 経路 | spec ソース | 生成物 |
|------|------------|--------|
| Worker → PostgREST | PostgREST OpenAPI spec (`/`) | openapi-typescript 型 + openapi-fetch |
| Console → Worker | 手書き `rpc()` | 将来対応 |
