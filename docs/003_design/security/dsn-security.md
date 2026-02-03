# セキュリティ設計書

| 項目 | 内容 |
|------|------|
| 文書ID | dsn-security |
| ステータス | draft |
| バージョン | v2.0 |
| 作成日 | 2026-02-03 |
| Sprint | SPRINT-007 (S7-010〜S7-015) |
| 前バージョンからの変更 | インフラ実態に合わせ全面改訂（Fly.io→Render/Koyeb、Token Broker廃止、OAuth/PKCE/API Key 実装詳細追加） |

---

## 1. 概要

MCPist のセキュリティ設計書。多層防御の設計思想に基づき、認証・認可・データ保護・入力検証の各レイヤーを定義する。

### 1.1 セキュリティモデル

個人プロジェクトとして現実的なセキュリティレベルを目指す。標的型攻撃への対策は過剰であり、設定ミス・トークン漏洩・不正入力への防御を優先する。

| 脅威カテゴリ | 対策方針 |
|-------------|---------|
| 設定ミス・デプロイ不整合 | Gateway Secret 検証、ヘルスチェック |
| トークン・鍵の漏洩 | 暗号化保存、ハッシュ保存、短命トークン |
| 不正入力 (SSRF, SQLi) | 入力検証、パラメータ化クエリ |
| 権限外アクセス | 多層権限ゲート（Filter → Gate → Detect） |
| 外部サービスの障害 | フェイルオーバー、Best-effort 設計 |

### 1.2 アーキテクチャ概要

```
Client (Claude Desktop / Cursor / API Key)
    │
    ▼
Cloudflare Worker (mcp.mcpist.com)
    ├─ JWT / API Key 認証
    ├─ X-Request-ID 生成
    ├─ X-User-ID / X-Auth-Type ヘッダー付与
    └─ X-Gateway-Secret 付与
    │
    ├─────────────────┐
    ▼                 ▼ (failover)
Render (Primary)    Koyeb (Standby)
    ├─ Gateway Secret 検証
    ├─ 権限ゲート
    ├─ ツール実行
    └─ クレジット消費
    │
    ▼
Supabase (DB, Auth, Vault)
    ├─ RLS ポリシー
    ├─ pgsodium TCE（認証情報暗号化）
    └─ Vault（OAuth アプリ設定暗号化）

Vercel (mcpist.com)
    ├─ Console UI (Next.js)
    └─ OAuth authorize/callback ルート
```

---

## 2. 認証

### 2.1 認証方式一覧

| 方式 | 認証主体 | 検証場所 | 用途 |
|------|---------|---------|------|
| JWT (Bearer) | Supabase Auth | Worker | Console からの MCP アクセス |
| API Key (mpt_*) | MCPist | Worker | Claude Desktop / Cursor 等からの直接アクセス |
| Gateway Secret | 共有シークレット | Go Server | Worker → Origin 間の信頼確認 |

### 2.2 JWT 認証

Worker が Supabase Auth の JWT を検証する。3段階のフォールバック:

| 順序 | 方式 | 説明 |
|------|------|------|
| 1 | OAuth userinfo | `/auth/v1/oauth/userinfo` エンドポイント呼び出し |
| 2 | Supabase Auth API | `/auth/v1/user` エンドポイント呼び出し |
| 3 | JWKS 署名検証 | Supabase 公開鍵で署名を検証（オフライン） |

検証項目:
- **署名**: JWKS (RS256) で検証
- **issuer**: Supabase プロジェクト URL と一致すること
- **有効期限**: exp クレーム（通常 1 時間）

**許容事項**: JWT 漏洩時は有効期限まで悪用される可能性あり。OAuth 2.0 仕様上の制約として許容する。

### 2.3 API Key 認証

| 項目 | 内容 |
|------|------|
| フォーマット | `mpt_` + 32 文字 hex（16 バイトランダム） |
| 生成 | Supabase RPC `generate_my_api_key` (`gen_random_bytes(16)`) |
| 保存 | SHA-256 ハッシュのみ DB に保存。平文は生成時に一度だけ表示 |
| キャッシュ | Cloudflare KV（hard TTL: 24h、soft TTL: 1h） |
| 無効化 | ソフトデリート（`revoked_at` タイムスタンプ） |
| 有効期限 | 任意設定可能（`expires_at`） |

検証フロー:
```
Worker: SHA-256(apiKey)
  → KV キャッシュ検索（1-5ms）
  → miss → Supabase RPC lookup_user_by_key_hash
  → revoked/expired チェック
  → last_used_at 更新（監査証跡）
  → キャッシュに保存
```

### 2.4 Gateway Secret

Worker と Go Server 間の信頼を確保する共有シークレット。

| 項目 | 内容 |
|------|------|
| 生成 | `openssl rand -hex 32`（256 ビット） |
| 保存 | GitHub Secrets |
| 配布 | Worker: `wrangler secret put`、Render/Koyeb: 環境変数 |
| 検証 | Go Server の Authorize ミドルウェアで `X-Gateway-Secret` を検証 |

**`invalid_gateway_secret` の主な原因**: デプロイ時の環境変数不整合、または直接オリジンへのアクセス。

ヘッダー一覧（Worker → Go Server）:

| ヘッダー | 値 | 用途 |
|----------|-----|------|
| `X-Gateway-Secret` | 共有シークレット | リクエスト元の信頼確認 |
| `X-User-ID` | UUID | 認証済みユーザー ID |
| `X-Auth-Type` | `jwt` / `api_key` | 監査用 |
| `X-Request-ID` | UUID v4 | リクエスト追跡 |

---

## 3. 認可（権限ゲート）

### 3.1 多層防御

```
Layer 1: 見せない (Filter)
  get_module_schema → 権限のないツールはスキーマに含めない
  LLM は存在を知らないツールを呼べない

Layer 2: 実行させない (Gate)
  call_module_tool → 権限チェック
  スキーマに無いツールの呼び出しを拒否

Layer 3: 記録する (Detect)
  権限外ツールの呼び出し試行をログに記録
  正常ユーザーは見えないツールを呼ばない = 異常
```

### 3.2 権限キャッシュ

| 項目 | 内容 |
|------|------|
| TTL | 5 分 |
| 保存 | Go Server インメモリ（sync.Map） |
| サイズ | ユーザーあたり約 1.5KB |
| 無効化 | 課金変更時に即時 `InvalidateUser(userID)` |
| マルチインスタンス | DB 同期（10 分間隔）。最大 100% 誤差を許容 |

### 3.3 アカウント状態

| ステータス | 挙動 |
|-----------|------|
| `active` | 正常アクセス |
| `suspended` | HTTP 403（全ツール拒否） |
| `disabled` | HTTP 403（全ツール拒否） |

### 3.4 クレジット認可

ツール実行にはクレジット消費が必要。不足時は HTTP 402 Payment Required。

```
TotalCredits = FreeCredits + PaidCredits
if TotalCredits < creditCost → 402
```

詳細は [dsn-permission-system.md](../details/dsn-permission-system.md) / [dsn-subscription.md](../details/dsn-subscription.md) を参照。

---

## 4. OAuth セキュリティ

### 4.1 プロバイダー一覧

| プロバイダー | OAuth | PKCE | トークン交換 | リフレッシュ |
|-------------|-------|------|-------------|------------|
| Google | 2.0 | - | POST body | あり（offline access） |
| Notion | 2.0 | - | Basic Auth | なし（長命トークン） |
| Airtable | 2.0 | **S256** | Basic Auth | あり（ローテーション） |
| GitHub | 2.0 | - | POST body | なし（長命トークン） |
| Jira | 2.0 | - | POST body | あり |
| Asana | 2.0 | - | POST body | あり |
| Trello | 2.0 | - | POST body | なし |
| Todoist | 2.0 | - | POST body | なし |
| TickTick | 2.0 | - | Basic Auth | なし |
| Microsoft | 2.0 | - | POST body | あり |

### 4.2 CSRF 対策（state パラメータ）

全 OAuth プロバイダーで `state` パラメータを使用:

```typescript
const stateData = {
  nonce: crypto.randomUUID(),   // 一意識別子
  returnTo,                      // リダイレクト先
  module                         // 接続対象モジュール
}
const state = Buffer.from(JSON.stringify(stateData)).toString("base64url")
```

callback 時に `state` を検証し、`returnTo` のパスを sanitize してからリダイレクトする。

### 4.3 PKCE 実装（Airtable）

Airtable は PKCE (S256) が必須:

| 項目 | 内容 |
|------|------|
| `code_verifier` | `crypto.randomBytes(64).toString("base64url")` |
| `code_challenge` | SHA-256(`code_verifier`).toString("base64url") |
| 保存 | httpOnly, secure, sameSite=lax Cookie |
| Cookie path | `/api/oauth/airtable`（狭いスコープ） |
| Cookie TTL | 600 秒（10 分） |

### 4.4 リフレッシュトークンローテーション

Airtable はリフレッシュ時に新しいリフレッシュトークンを発行し、旧トークンを無効化する。Go Server 側で新しいペアを Supabase に保存する。

### 4.5 OAuth アプリ設定の保存

OAuth クライアント ID / Secret は Supabase Vault に暗号化保存:

```sql
-- mcpist.oauth_apps テーブル
secret_id UUID REFERENCES vault.secrets(id)  -- Vault で暗号化

-- 復号は SECURITY DEFINER RPC 経由
get_oauth_app_credentials(p_provider TEXT)
  → client_id, client_secret, redirect_uri を返す
```

Console の OAuth ルートのみが RPC を呼び出し、client_secret を取得する。

---

## 5. データ保護

### 5.1 認証情報の暗号化

ユーザーの OAuth トークン・API キーは `mcpist.user_credentials` テーブルに保存:

| 項目 | 内容 |
|------|------|
| テーブル | `mcpist.user_credentials` |
| 暗号化 | pgsodium Transparent Column Encryption (TCE) |
| 暗号鍵 | `user_credentials_key`（aead-det タイプ） |
| ASSOCIATED | `(id, user_id)` — 行の入れ替え攻撃を防止 |

格納される JSON:
```json
{
  "auth_type": "oauth2",
  "access_token": "...",
  "refresh_token": "...",
  "token_type": "Bearer",
  "scope": "...",
  "expires_at": 1234567890
}
```

### 5.2 API Key のハッシュ保存

| 項目 | 内容 |
|------|------|
| テーブル | `mcpist.api_keys` |
| 保存形式 | SHA-256 ハッシュのみ（`key_hash` カラム） |
| 平文 | DB に保存しない。生成時にユーザーに一度だけ表示 |
| 表示用 | `key_prefix`（先頭 8 文字 + `...` + 末尾 4 文字） |

### 5.3 RLS ポリシー

Row Level Security でユーザー間のデータ分離を保証:

| テーブル | SELECT | INSERT/UPDATE/DELETE | 備考 |
|----------|--------|---------------------|------|
| `users` | `auth.uid() = id` | UPDATE のみ self | 自分の情報のみ |
| `credits` | `auth.uid() = user_id` | RPC 経由 (service_role) | クレジット操作は RPC のみ |
| `user_credentials` | `auth.uid() = user_id` | RPC 経由 (service_role) | トークン操作は RPC のみ |
| `api_keys` | `auth.uid() = user_id` | self | ユーザー自身が管理 |
| `tool_settings` | `auth.uid() = user_id` | self | ツール有効/無効の切替 |
| `oauth_apps` | なし | service_role のみ | 管理者用 |

---

## 6. 入力検証

### 6.1 SSRF 対策（PostgreSQL モジュール）

`postgresql` モジュールはユーザーが接続先を指定するため、SSRF リスクがある。

| 検証項目 | 内容 |
|---------|------|
| スキーム | `postgresql://` または `postgres://` のみ |
| ホスト | 必須。`localhost`, `127.0.0.1`, `::1` を禁止 |
| データベース名 | 必須 |
| SSL | 未指定時は `sslmode=require` を自動付与 |

```go
// apps/server/internal/modules/postgresql/module.go
if host == "localhost" || host == "127.0.0.1" || host == "::1" {
    return fmt.Errorf("localhost connections are not allowed for security reasons")
}
```

### 6.2 SQL インジェクション対策（PostgreSQL モジュール）

| 対策 | 内容 |
|------|------|
| パラメータ化クエリ | `conn.Query(ctx, sql, queryParams...)` |
| DDL パターン検出 | `DROP`, `TRUNCATE`, `ALTER`, `CREATE` を `execute()` で禁止 |
| ツール分離 | SELECT は `query()`、DML は `execute()`、DDL は `execute_ddl()` |

```go
var dangerousPatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)^\s*DROP\s+`),
    regexp.MustCompile(`(?i)^\s*TRUNCATE\s+`),
    // ...
}
```

### 6.3 returnTo パラメータの検証

OAuth callback 後のリダイレクト先 (`returnTo`) を検証:

- パスのみ許可（外部 URL へのリダイレクト防止）
- デフォルトは `/services`

---

## 7. シークレット管理

### 7.1 シークレット一覧

| シークレット | 保存場所 | 用途 |
|-------------|---------|------|
| `GATEWAY_SECRET` | GitHub Secrets → 各サービス環境変数 | Worker ↔ Origin 間認証 |
| `SUPABASE_SERVICE_ROLE_KEY` | GitHub Secrets → Go Server 環境変数 | Supabase 管理操作 |
| `SUPABASE_JWT_SECRET` | Supabase 管理画面 | JWT 署名（Supabase 管理） |
| OAuth client_id/secret | Supabase Vault (`oauth_apps`) | 各プロバイダーの OAuth 認証 |
| `GRAFANA_LOKI_API_KEY` | Go Server 環境変数 | Loki ログ送信 |
| `STRIPE_SECRET_KEY` | Go Server 環境変数 | 決済処理 |
| `STRIPE_WEBHOOK_SECRET` | Go Server 環境変数 | Webhook 署名検証 |

### 7.2 ローテーション手順

| シークレット | ローテーション方法 |
|-------------|-----------------|
| Gateway Secret | 1. 新しい値を生成 2. Worker + Render + Koyeb を同時に更新 3. 確認 |
| OAuth client secret | 各プロバイダーの管理画面で再生成 → Vault 更新 |
| API Key | ユーザーが Console で revoke → 新規発行 |
| Supabase service role key | Supabase ダッシュボードで再生成 → 全サービス再デプロイ |

---

## 8. リスクと許容事項

個人プロジェクトとして、コスト対効果の観点から以下を許容する:

| リスク | 深刻度 | 対策 | 判断 |
|--------|--------|------|------|
| JWT 漏洩 | 中 | 短命トークン（1h） | **許容**: OAuth 仕様上の制約 |
| Rate Limit のマルチインスタンス誤差 | 低 | DB 同期（10 分） | **許容**: 最大 100% 誤差。Redis は過剰 |
| pgsodium 鍵管理 | 中 | Supabase 管理 | **委託**: Supabase の責任範囲 |
| Vault 暗号化 | 中 | Supabase Vault | **委託**: Supabase の責任範囲 |
| DDoS | 低 | Cloudflare (Free) | **許容**: Free Tier の WAF + Rate Limiting |

---

## 9. 新規モジュール追加時のチェックリスト

新しいモジュールを追加する際に確認すべきセキュリティ項目:

### 認証

- [ ] OAuth の場合: state パラメータを使用しているか
- [ ] OAuth の場合: PKCE が必要なプロバイダーか確認したか
- [ ] トークン交換: client_secret の送信方法は正しいか（Basic Auth / POST body）
- [ ] リフレッシュトークン: ローテーションが必要か確認したか
- [ ] トークン保存: `upsert_my_credential` RPC 経由で保存しているか

### データ保護

- [ ] ユーザーデータをログに出力していないか
- [ ] API レスポンスボディをログに出力していないか
- [ ] トークン・API Key をログに出力していないか

### 入力検証

- [ ] ユーザー入力を URL に埋め込む場合、パスインジェクションを防いでいるか
- [ ] ユーザー入力で外部 URL を指定できる場合、SSRF を防いでいるか

### 権限

- [ ] `tools-export` に登録し、`tools.json` を再生成したか
- [ ] `readOnlyHint` / `destructiveHint` を適切に設定したか

---

## 参考文書

| 文書 | 内容 |
|------|------|
| [dsn-permission-system.md](../details/dsn-permission-system.md) | 権限システム設計 |
| [dsn-subscription.md](../details/dsn-subscription.md) | サブスクリプション・クレジット設計 |
| [dsn-observability.md](../observability/dsn-observability.md) | Observability 設計（異常イベント検知含む） |
| [dsn-infrastructure.md](../system/dsn-infrastructure.md) | インフラ設計 |
| [adr-permission-naming.md](./adr-permission-naming.md) | 権限命名 ADR |
| [adr-rate-limit-architecture.md](./adr-rate-limit-architecture.md) | Rate Limit ADR |
| [adr-usage-control-architecture.md](./adr-usage-control-architecture.md) | 使用量制御 ADR |
