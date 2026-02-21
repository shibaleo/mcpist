# Sprint 010 計画書

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-010 |
| 期間 | 2026-02-15 〜 2026-02-21 (7日間) |
| マイルストーン | M9: 認証基盤整理 + DB 移行 |
| 前提 | Sprint-009 完了 (サブスク移行、堅牢性改善) |
| 状態 | 計画中 |

---

## Sprint 目標

**Supabase 依存を解消する。認可を自前で制御し、DB を Neon に移す**

Sprint 009 でサブスクリプション移行と堅牢性改善が完了した。次は構造的負債に取り組む:

1. Supabase Auth OAuth がブラックボックス — LLM 認可フロー失敗時にログが出ない
2. DB + Auth + Vault が Supabase に集中 — 単一障害点
3. 大規模移行にテストがない

---

## 設計判断

### Console ログイン (Social OIDC): 移行しない

Supabase Auth に残す。問題が起きていない。LLM 認可と Console 認証は独立した問題。

### pgsodium TCE → pgcrypto

Neon は pgsodium をサポートしない。pgcrypto の `pgp_sym_encrypt` / `pgp_sym_decrypt` で代替。暗号化キーは環境変数 `CREDENTIAL_ENCRYPTION_KEY` で管理。

### Stripe ogen 化: Sprint 010 ではスコープ外

現状の TypeScript SDK で問題なく動作。Phase 1/2 が大きすぎる。Sprint 011 以降のバックログに残す。

### DB 移行: カットオーバー方式

同時稼働しない。Neon セットアップ → データ移行 → 環境変数切替 → Supabase 停止。

---

## タスク一覧

### Phase 1: Cloudflare Workers OAuth 2.1 Server (優先度: 最高)

LLM → MCPist API の認可フローを Supabase Auth → 自前 Worker に移行。

#### 1a. OAuth Server 本体

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S10-001 | `@cloudflare/workers-oauth-provider` 導入 | apps/worker/ | KV namespace `TOKEN_STORE` 追加 |
| S10-002 | OAuth 2.1 Server エンドポイント実装 | apps/worker/src/oauth-server.ts (新規) | authorize, token, userinfo, revoke |
| S10-003 | Token storage (Workers KV) | apps/worker/src/oauth-server.ts | auth code / access token / refresh token |
| S10-004 | User identity resolution | apps/worker/src/oauth-server.ts | user_id を DB から lookup |
| S10-005 | Consent redirect | apps/worker/src/oauth-server.ts | 同意画面は Console に委譲 |

**フロー:**
```
1. LLM → GET /oauth/authorize (PKCE)
   → Worker: consent page redirect
2. User → Console consent page → 同意
   → Console → Worker POST /oauth/approve
3. LLM → POST /oauth/token (code exchange)
   → Worker: access_token + refresh_token 発行
4. LLM → API call (Bearer token)
   → Worker: KV で token 検証 → user_id 取得
```

#### 1b. メタデータ + JWT 検証更新

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S10-006 | RFC 9728 メタデータ更新 | apps/worker/src/index.ts | `authorization_servers` を自前 Worker URL に変更 |
| S10-007 | RFC 8414 メタデータ更新 | apps/worker/src/index.ts | Supabase プロキシ → 自前メタデータ |
| S10-008 | JWT 検証ロジック更新 | apps/worker/src/index.ts | 自前 OAuth token 検証を優先、Supabase を fallback に |

#### 1c. Console 同意画面

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S10-010 | Consent page を自前 OAuth Server 対応に改修 | apps/console/src/app/oauth/consent/page.tsx | 現在 Supabase OAuth API 依存。自前 API に切替 |
| S10-011 | Consent API route 追加 | apps/console/src/app/api/oauth/consent/route.ts (新規) | 同意時に Worker へ callback |

#### 1d. 検証

| ID | タスク | 備考 |
|----|--------|------|
| S10-015 | 既存 Supabase Auth トークンの互換性確認 | fallback で壊れないこと |
| S10-016 | Claude App 認可フロー E2E テスト | 認可 → トークン取得 → API call |

### Phase 2: Neon PG 移行 (優先度: 高)

#### 2a. Neon セットアップ + スキーマ移植

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S10-020 | Neon プロジェクト作成 | Neon Dashboard | Oregon リージョン |
| S10-021 | スキーマ移植 (DDL) | Neon SQL | 24 migration から DDL 抽出。`auth.users` FK 除去 |
| S10-022 | RPC 関数移植 | Neon SQL | get_user_context, record_usage, credential 関連等 |
| S10-023 | pgsodium TCE → pgcrypto 暗号化変更 | Neon SQL | get_user_credential / upsert_user_credential |
| S10-024 | users テーブルの auth.users FK 除去 | Neon SQL | UUID はそのまま維持 |

#### 2b. データ移行

| ID | タスク | 備考 |
|----|--------|------|
| S10-025 | pg_dump (Supabase → ファイル) | mcpist スキーマのデータのみ |
| S10-026 | credential 復号 + 再暗号化 | Supabase pgsodium 復号 → Neon pgcrypto 暗号化 |
| S10-027 | データインポート (→ Neon) | psql or Neon import |

#### 2c. Go Server 接続先変更

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S10-030 | broker/db.go 新設: DB 接続抽象化 | broker/db.go (新規) | Supabase/Neon のヘッダー差異を吸収 |
| S10-031 | broker/user.go: Neon Data API URL に切替 | broker/user.go | PostgREST 互換、URL 変更 |
| S10-032 | broker/token.go: Neon Data API URL に切替 | broker/token.go | 同上 |
| S10-033 | broker/module.go: Neon Data API URL に切替 | broker/module.go | 同上 |
| S10-034 | HealthCheck 更新 | cmd/server/main.go | Neon Data API ヘルスチェック |

**broker/db.go 設計:**
```go
type DBConfig struct {
    BaseURL  string
    APIKey   string
    AuthType string // "supabase" or "neon"
}

func (c *DBConfig) SetAuthHeaders(req *http.Request) {
    switch c.AuthType {
    case "neon":
        req.Header.Set("Authorization", "Bearer "+c.APIKey)
    default:
        req.Header.Set("apikey", c.APIKey)
        req.Header.Set("Authorization", "Bearer "+c.APIKey)
    }
}
```

#### 2d. Console + Worker 接続先変更

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S10-035 | Console DB アクセスを Neon に切替 | apps/console/src/lib/supabase/ | supabase.rpc() → fetch (Neon Data API) |
| S10-036 | admin.ts の service role 接続変更 | apps/console/src/lib/supabase/admin.ts | Stripe webhook, onboarding で使用 |
| S10-037 | Worker API Key 検証先変更 | apps/worker/src/index.ts | lookup_user_by_key_hash の呼出先を Neon に |

### Phase 3: テスト基盤 (優先度: 中)

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S10-040 | authz middleware ユニットテスト | middleware/authz_test.go (新規) | CanAccessTool, WithinDailyLimit |
| S10-041 | broker/user.go ユニットテスト | broker/user_test.go (新規) | GetUserContext, RecordUsage, cache |
| S10-042 | broker/retry.go ユニットテスト | broker/retry_test.go (新規) | backoffWithJitter, isRetryable |
| S10-043 | CI トリガーを push/PR に変更 | .github/workflows/ci.yml | workflow_dispatch → push + pull_request |

### Phase 4: 小タスク (優先度: 低)

| ID | タスク | 備考 |
|----|--------|------|
| S10-050 | 仕様書更新: credit → subscription model | Sprint 009 backlog 繰越し |

---

## 作業順序

```
Day 1:   Phase 1a (S10-001〜005) — OAuth Server 本体
Day 2:   Phase 1b (S10-006〜008) + 1c (S10-010〜011) — メタデータ + 同意画面
Day 3:   Phase 1d (S10-015〜016) + Phase 2a (S10-020〜022) — OAuth 検証 + Neon セットアップ
Day 4:   Phase 2a (S10-023〜024) + 2b (S10-025〜027) — 暗号化移行 + データ移行
Day 5:   Phase 2c (S10-030〜034) + 2d (S10-035〜037) — 接続先変更
Day 6:   Phase 3 (S10-040〜043) — テスト + CI
Day 7:   バッファ + Phase 4 + E2E 検証
```

---

## スコープ外 (Sprint 011 以降)

| 項目 | 理由 |
|------|------|
| Console ログイン移行 (Supabase Auth → Neon Auth) | 問題が起きていない |
| Stripe ogen 化 | 現状動作中。Phase 1/2 が大きすぎる |
| 分散 Rate Limiter | マルチインスタンス運用開始時 |

---

## リスク

| リスク | 影響 | 対策 |
|--------|------|------|
| @cloudflare/workers-oauth-provider の学習コスト | Phase 1 遅延 | examples を事前調査 |
| pgsodium → pgcrypto で credential 損失 | 全ユーザーの外部サービス接続切断 | Supabase で復号 → ファイル → Neon で再暗号化。ロールバック手順を用意 |
| Neon Data API の PostgREST 互換性 | broker/ が動かない | ドキュメント事前確認。直接 PG 接続をフォールバック |
| Phase 1 + 2 同時進行でスコープ超過 | Sprint 未達 | Phase 1 最優先。Phase 2 が間に合わなければ Sprint 011 に繰越し |
| 既存 OAuth トークン無効化 | ユーザー体験断絶 | 移行期間中 Supabase Auth 検証を fallback 維持 |

---

## 完了条件

- [ ] LLM が自前 OAuth 2.1 Server 経由で認可フローを完走できる
- [ ] OAuth 認可フローのログが出力される
- [ ] Worker の well-known endpoints が自前 OAuth Server を指す
- [ ] Neon PG にスキーマ + RPC + データが移植されている
- [ ] Go Server が Neon Data API 経由で全 RPC を呼び出せる
- [ ] Console が Neon 経由で DB 操作できる
- [ ] user_credentials の暗号化が pgcrypto で動作している
- [ ] authz middleware のユニットテストが CI で pass
- [ ] CI トリガーが push/PR で自動実行

---

## 主要変更ファイル

- `apps/worker/src/index.ts` — OAuth メタデータ, JWT 検証, API Key 検証
- `apps/worker/src/oauth-server.ts` (新規) — OAuth 2.1 Server
- `apps/console/src/app/oauth/consent/page.tsx` — 同意画面 (Supabase OAuth API → 自前)
- `apps/server/internal/broker/user.go` — GetUserContext, RecordUsage
- `apps/server/internal/broker/token.go` — fetchCredentials
- `apps/server/internal/broker/db.go` (新規) — DB 接続抽象化
- `apps/server/internal/middleware/authz.go` — CanAccessTool — テスト対象
- `supabase/migrations/` — 24 ファイルからスキーマ + RPC 抽出

---

## 参考

- [sprint009-review.md](sprint009-review.md) - Sprint 009 レビュー
- [sprint009-backlog.md](sprint009-backlog.md) - Sprint 009 バックログ (統合版)
