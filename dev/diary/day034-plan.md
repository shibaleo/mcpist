# DAY034 計画

## 日付

2026-02-21

---

## 前提: day033 までの状態

- Go Server: ogen 自動生成サーバーに移行完了 (28 EP)、ビルド通過、未デプロイ
- Worker → Go Server: Ed25519 JWT ゲートウェイ認証に移行済み
- Supabase 完全除去済み、ローカル Docker PG 導入済み
- セキュリティ調査で 5 件のリスクを検出 (issue_report.md)
- Worker 廃止計画を策定済み (worker_to_go_plan.md)

### day032 バックログ消化状況

| ID | 項目 | 状態 |
|----|------|------|
| 1A | Clerk DCR 有効化 | 未着手 |
| 1B | OAuth App Credentials 再登録 | 未着手 |
| 1C | OAuth コールバック URL 本番設定 | 未着手 |
| 2A | Render 環境変数 | ✅ day033 で `GATEWAY_SECRET` → `GATEWAY_SIGNING_KEY` 移行、`POSTGREST_*` 削除 |
| 2B | Workers シークレット | ✅ day033 で `GATEWAY_SIGNING_KEY` 設定済み |
| 2C | Console 環境変数 | 要確認 |
| 2D | Stripe Webhook 設定 | 一部完了 (Go Server 実装済み、Stripe Dashboard 未設定) |
| 3A | expires_at 形式統一 | 未修正 |
| 3B | Clerk セッション無限リダイレクト | ✅ day033 で AuthProvider ref guard 修正 |
| 3C | Console 本番デプロイ設定 | 未着手 |
| 4A | INTERNAL_SECRET 廃止 | 未着手 |
| 4B | SECONDARY_API_URL 廃止 | 未着手 |
| 5A | ユーザークレデンシャル暗号化 | 未着手 |
| 5B | OAuth App Credentials 暗号化 | 未着手 |
| 6A | Go Server Render デプロイ確認 | 未確認 |
| 6B | Worker → Go Server 接続テスト | ✅ day033 でヘルスチェック確認 (b85f0f3) |
| 6D | Stripe Webhook Go Server 実装 | ✅ day033 で ogenserver/stripe.go に実装 |

---

## 本日の方針

day033 で大規模リファクタ (ogen 移行 + Supabase 除去) を行ったが、dev 環境に未デプロイ。
**まず動くものを確認する** ことを最優先とし、その後セキュリティリスクの #1 (API キー失効) を対処する。

Worker 廃止は中期目標として計画のみ。本日は着手しない。

---

## Phase 1: デプロイ + E2E 検証 (最優先)

### 1-1. Go Server デプロイ

- Render に push → 自動デプロイ
- ヘルスチェック確認: `curl https://mcp.dev.mcpist.app/health`
- ログで `Database connected`, `Ed25519 key pair loaded`, `SyncModules` 成功を確認
- Render 環境変数の確認:
  - `WORKER_JWKS_URL` が設定されているか (Ed25519 JWT 検証に必要)
  - `STRIPE_WEBHOOK_SECRET` が Go Server 側にも設定されているか

### 1-2. Worker デプロイ

- `wrangler deploy -e dev`
- `GATEWAY_SIGNING_KEY` が設定済みか確認 (day033 で `GATEWAY_SECRET` から移行)

### 1-3. E2E 動作確認

| テスト | エンドポイント | 期待 |
|--------|---------------|------|
| モジュール一覧 | `GET /v1/modules` | 200 + モジュール配列 |
| プロフィール取得 | `GET /v1/me/profile` | 200 + UserProfile |
| 使用量 | `GET /v1/me/usage?start=2026-02-21&end=2026-02-21` | 200 + UsageData |
| 登録 (冪等) | `POST /v1/me/register` | 200 + RegisterResult |
| MCP tools/list | `/v1/mcp` (JSON-RPC) | 接続済みモジュールのツール一覧 |
| Stripe webhook | `POST /v1/stripe/webhook` (Stripe CLI) | 200 + `{"received": true}` |

---

## Phase 2: API キー失効の実効性確保

**セキュリティ調査 #1 (Critical)** — API キー JWT の署名検証のみで DB 失効状態を参照していない。

### 2-1. JWT に key_id クレームを追加

- `auth/keys.go`: `GenerateAPIKeyJWT` で `key_id` (= DB の api_keys.id) を JWT クレームに含める
- `server-openapi.yaml` の `GenerateApiKey` レスポンスは変更不要 (JWT 文字列に内包)

### 2-2. API キー認証時に DB 照合を追加

- `ogenserver/security.go`: API キー認証パスで以下を追加:
  1. JWT から `key_id` を抽出
  2. `db.GetAPIKeyByID(key_id)` で存在確認
  3. 存在しない (= revoke 済み) なら 401
- Worker 側の `auth.ts` にも同様の照合を追加 (Worker が JWT を検証するフロー)

### 2-3. API キーの有効期限

- `GenerateApiKey` ハンドラで JWT の `exp` クレームを必須化 (デフォルト 90 日)
- Console の API キー生成 UI に有効期限の表示を追加

---

## Phase 3: OAuth state 検証の強化

**セキュリティ調査 #2 (High)** — OAuth `state` の真正性検証不足。

### 3-1. state を署名付きに変更

- authorize ルート: `state = base64url(HMAC-SHA256(nonce, secret) + nonce)`
- callback ルート: `state` から nonce を取り出し HMAC を再計算して照合
- secret は `CLERK_SECRET_KEY` を流用 (新たな環境変数は不要)

### 3-2. 全 OAuth プロバイダに適用

- Google, Microsoft, Notion, Atlassian, Asana, Todoist, Airtable, TickTick, Trello, Dropbox
- 共通ユーティリティ `lib/oauth/state.ts` を作成し、全 authorize/callback で使用

---

## Phase 4 (時間があれば): 検証 API の認証必須化

**セキュリティ調査 #4 (Medium)** — `/api/credentials/validate` が未認証。

- Clerk 認証ミドルウェアを追加
- レート制限 (1 req/sec/user) を追加
- 許可ドメインのホワイトリストは不要 (認証で十分)

---

## 着手しないもの (中期バックログ)

| 項目 | 理由 |
|------|------|
| Worker 廃止 → Go Server 集約 | 大規模。計画は策定済み (worker_to_go_plan.md)。デプロイ安定後に着手 |
| クレデンシャル暗号化 (issue #3) | 設計済み (day032-backlog 5A-5D)。デプロイ安定 + API キー修正の後 |
| `.env.dev` の git 除外 (issue #5) | 影響小。次回の環境変数整理時にまとめて対応 |
| Clerk DCR 有効化 | MCP OAuth 2.1 フローは次フェーズ |
| OAuth App Credentials 再登録 | dev 環境の E2E 確認後に admin API 経由で実施 |

---

## 優先順位まとめ

```
1. デプロイ + E2E 検証 (Phase 1)     ← ブロッカー: 全ての前提
2. API キー失効修正 (Phase 2)         ← Critical セキュリティ修正
3. OAuth state 検証 (Phase 3)         ← High セキュリティ修正
4. 検証 API 認証 (Phase 4)            ← Medium、時間があれば
```

---

## 成功基準

- [ ] dev 環境で Go Server + Worker が正常稼働
- [ ] Console からログイン → プロフィール表示 → サービス一覧が動作
- [ ] API キーを revoke した後、そのキーでのアクセスが 401 になる
- [ ] OAuth state が改ざん/リプレイされた場合に callback が拒否される
