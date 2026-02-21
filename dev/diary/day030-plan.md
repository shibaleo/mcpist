# DAY030 作業計画

## 日付

2026-02-15

## 対応スプリント

Sprint 010 — Phase 1a: Cloudflare Workers OAuth 2.1 Server 本体

---

## 目標

`@cloudflare/workers-oauth-provider` を使い、LLM → MCPist API の認可フローを自前 OAuth 2.1 Server で動かす。

---

## タスク

| # | ID | タスク | 成果物 |
|---|-----|--------|--------|
| 1 | S10-001 | `@cloudflare/workers-oauth-provider` 導入 | package.json, wrangler.toml (KV namespace `TOKEN_STORE`) |
| 2 | S10-002 | OAuth 2.1 Server エンドポイント実装 | apps/worker/src/oauth-server.ts |
| 3 | S10-003 | Token storage (Workers KV) | auth code / access token / refresh token の保存 |
| 4 | S10-004 | User identity resolution | DB から user_id lookup |
| 5 | S10-005 | Consent redirect | Console 同意画面へのリダイレクト |

---

## 作業順序

```
1. @cloudflare/workers-oauth-provider のドキュメント + examples 読み込み
2. wrangler.toml に KV namespace 追加、パッケージインストール
3. oauth-server.ts 新規作成 (authorize, token, userinfo, revoke)
4. index.ts にルーティング追加 (/oauth/* → oauth-server)
5. ローカル wrangler dev で動作確認
```

## 参考

- [sprint010-plan.md](sprint010-plan.md) - Sprint 010 計画
- [@cloudflare/workers-oauth-provider](https://github.com/cloudflare/workers-oauth-provider) - ライブラリ
