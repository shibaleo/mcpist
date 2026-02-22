# MCPist セキュリティ仕様書（spc-sec）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v3.0 (Sprint-012) |
| Note | Security Specification — 現行実装に基づく全面改訂 |

---

## 概要

本ドキュメントは、MCPist のセキュリティ要件と実装を定義する。

---

## 認証

### 認証方式

| 方式 | 用途 | 実装 |
|---|---|---|
| Clerk JWT | Console ユーザー認証 | Worker が Clerk JWKS で検証 |
| API Key JWT | MCP クライアント認証 | `mpt_` プレフィックス + Ed25519 署名 JWT |
| Gateway JWT | Worker → Server 間認証 | Ed25519 署名、30 秒有効 |

### Clerk JWT 認証フロー

1. Console / MCP クライアントが Clerk JWT を Bearer トークンとして送信
2. Worker が Clerk JWKS (`jose.createRemoteJWKSet`) で署名検証
3. `payload.sub` (Clerk User ID) を抽出

### API Key JWT 認証フロー

1. MCP クライアントが `mpt_` プレフィックス付き JWT を Bearer トークンとして送信
2. Worker が Server の JWKS エンドポイント (`/.well-known/jwks.json`) で Ed25519 署名検証
3. JWT claims から `sub` (内部 UUID)、`kid` (Key ID) を抽出
4. Server の `/v1/internal/apikeys/{keyId}/status` で有効性確認 (5 分キャッシュ)
5. 削除時はキャッシュを即時無効化

**API Key 仕様:**

| 項目 | 値 |
|---|---|
| 署名アルゴリズム | Ed25519 |
| 鍵サイズ | 32 バイト (seed) |
| Key ID | `mcpist-api-key-v1` |
| デフォルト有効期限 | 90 日 |
| プレフィックス | `mpt_` |
| 失効方式 | DB から物理削除 |

### Gateway JWT

Worker が認証成功後、Server への内部通信用に署名する短命 JWT。

| 項目 | 値 |
|---|---|
| 署名アルゴリズム | EdDSA (Ed25519) |
| 有効期限 | 30 秒 |
| Issuer | `mcpist-gateway` |
| Key ID | `mcpist-gateway-v1` |
| ヘッダー | `X-Gateway-Token` |
| 検証側 leeway | 5 秒 |

**Claims:**

| Claim | 説明 |
|---|---|
| user_id | 内部 UUID (API Key 認証時) |
| clerk_id | Clerk User ID (JWT 認証時) |
| email | メールアドレス (任意) |

---

## 認可

### Server 認可ミドルウェア

Gateway JWT 検証後、以下のコンテキストを構築して認可判定を行う。

| 項目 | 説明 |
|---|---|
| AccountStatus | `active` でなければ拒否 |
| EnabledModules | ユーザーが有効化したモジュールのホワイトリスト |
| EnabledTools | モジュール × ツール単位の有効/無効 |
| DailyLimit | プラン別日次使用量上限 |
| DailyUsed | 当日の使用量 |

**認可チェック順序:**

1. Gateway JWT 署名検証
2. ユーザー解決 (clerk_id → 内部 UUID、または user_id 直接)
3. アカウント状態確認 (`active` のみ許可)
4. モジュール有効チェック
5. ツール有効チェック
6. 日次使用量上限チェック

**エラーコード:**

| コード | HTTP | 説明 |
|---|---|---|
| MISSING_GATEWAY_TOKEN | 401 | X-Gateway-Token ヘッダーなし |
| INVALID_GATEWAY_TOKEN | 401 | JWT 検証失敗 |
| USER_RESOLUTION_ERROR | 500 | ユーザー解決失敗 |
| ACCOUNT_NOT_ACTIVE | 403 | アカウント無効 |
| MODULE_NOT_ENABLED | 403 | モジュール未有効化 |
| TOOL_DISABLED | 403 | ツール無効 |
| USAGE_LIMIT_EXCEEDED | 429 | 日次上限超過 |

### 管理者認可

| 項目 | 値 |
|---|---|
| 判定方式 | `ADMIN_EMAILS` 環境変数 (カンマ区切り) |
| 比較 | 大文字小文字を区別しない |
| 対象操作 | OAuth アプリ CRUD、全 OAuth 同意の一覧 |

### プロンプトインジェクション対策

| レイヤー | 対策 |
|---|---|
| Layer 1 (見せない) | 有効化されていないモジュールのスキーマを返却しない |
| Layer 2 (実行させない) | ツール実行時にモジュール・ツールの有効チェック |
| Layer 3 (検知する) | 権限外アクセス試行をセキュリティログに記録 |

---

## レート制限

### Server 側 (MCP エンドポイント)

| 項目 | 値 |
|---|---|
| 方式 | インメモリ sliding window |
| ウィンドウ | 1 秒 |
| 単位 | ユーザーごと |
| レスポンス | HTTP 429 + `Retry-After: 1` |

**クリーンアップ:** 60 秒ごとにバックグラウンドで 5 分以上アイドルのエントリを削除。

---

## 暗号化

### 保存時暗号化

| 対象 | 方式 | 形式 |
|---|---|---|
| user_credentials.encrypted_credentials | AES-256-GCM | `v1:base64(nonce\|\|ciphertext\|\|tag)` |
| oauth_apps.encrypted_client_secret | AES-256-GCM | 同上 |

**暗号化仕様:**

| 項目 | 値 |
|---|---|
| アルゴリズム | AES-256-GCM |
| 鍵サイズ | 256 ビット (32 バイト) |
| Nonce | 12 バイト (`crypto/rand`) |
| 鍵ソース | `CREDENTIAL_ENCRYPTION_KEY` 環境変数 (base64) |
| バージョニング | `key_version` 列 (現在 v1) |

### 通信暗号化

| 区間 | 暗号化 |
|---|---|
| Client ↔ Worker | TLS (Cloudflare) |
| Worker ↔ Server | TLS (Render) |
| Server ↔ Neon | TLS |
| Server ↔ 外部 API | TLS |

---

## Webhook セキュリティ

### Stripe Webhook 検証

| 項目 | 値 |
|---|---|
| 署名アルゴリズム | HMAC-SHA256 |
| シークレット | `STRIPE_WEBHOOK_SECRET` 環境変数 |
| タイムスタンプ許容 | 5 分 |
| 比較方式 | 定数時間比較 (`hmac.Equal`) |
| 冪等性 | `processed_webhook_events` テーブルで重複排除 |

**処理対象イベント:**

| イベント | 処理 |
|---|---|
| invoice.paid | サブスクリプション有効化 |
| customer.subscription.deleted | free プランへダウングレード |

---

## SSRF 対策

### 資格情報バリデータ

外部サービスの資格情報検証時、接続先を制限する。

| 対策 | 説明 |
|---|---|
| 許可ホスト | サービスごとに公式 API ホストのみ許可 |
| ローカルホスト禁止 | `localhost`, `127.0.0.1`, `::1` をブロック |
| タイムアウト | HTTP クライアント 10 秒 |

### PostgreSQL モジュール

| 対策 | 説明 |
|---|---|
| スキーム制限 | `postgresql://` または `postgres://` のみ |
| ローカルホスト禁止 | localhost 接続をブロック |
| SSL 強制 | `sslmode=require` をデフォルト付与 |
| 接続タイムアウト | 10 秒 |
| クエリタイムアウト | 30 秒 |
| 最大行数 | デフォルト 1,000、上限 10,000 |
| DDL 制限 | DROP, TRUNCATE, ALTER 等は `execute_ddl` ツールのみ |
| パラメータ化 | `$1, $2, ...` プレースホルダによる SQL インジェクション防止 |

---

## 入力検証

| 対策 | 説明 |
|---|---|
| SQL インジェクション | GORM ORM 使用 (生 SQL なし) |
| XSS | React 標準エスケープ |
| リクエストサイズ | Cloudflare Workers による制限 |
| JSON スキーマ | ogen 生成コードによるバリデーション |

---

## セキュリティイベントログ

### ログ送信先

Grafana Cloud (Loki)。Worker・Server 両方からセキュリティイベントを送信。

### Server 側ログ

```go
LogSecurityEvent(requestID, userID, event, details)
```

| ラベル | 値 |
|---|---|
| type | `security` |
| level | `warn` |

**記録対象:** Gateway トークン欠如/無効、権限外アクセス試行

### Worker 側ログ

| ラベル | 値 |
|---|---|
| app | `mcpist-worker` |
| type | `security` / `request` |

**記録対象:** 認証失敗、不正 API キー使用

---

## OAuth トークン管理

### トークンリフレッシュ

| 項目 | 値 |
|---|---|
| リフレッシュバッファ | 有効期限の 5 分前 |
| HTTP タイムアウト | 10 秒 |
| フォールバック | リフレッシュ失敗時は既存トークンを返却 |

### プロバイダ別設定

| プロバイダ | 認証方式 | リフレッシュトークンローテーション |
|---|---|---|
| Google (全サービス) | form | なし |
| Asana | form | あり |
| Dropbox | form | なし |
| Microsoft | form | あり |
| Notion | Basic 認証 | あり |
| Airtable | Basic 認証 | あり |
| Jira / Confluence | form | あり |

---

## セキュリティ環境変数

| 変数 | 用途 |
|---|---|
| CREDENTIAL_ENCRYPTION_KEY | AES-256-GCM 暗号化キー (base64, 32 バイト) |
| API_KEY_PRIVATE_KEY | Ed25519 秘密鍵 (base64) |
| GATEWAY_SIGNING_KEY | Gateway JWT 署名鍵 (base64, 32 バイト) |
| STRIPE_WEBHOOK_SECRET | Stripe Webhook HMAC シークレット |
| ADMIN_EMAILS | 管理者メールアドレス (カンマ区切り) |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|---|---|
| [spec-systems.md](./spec-systems.md) | システム仕様書 |
| [spec-design.md](./spec-design.md) | 設計仕様書 |
| [spec-infrastructure.md](./spec-infrastructure.md) | インフラ仕様書 |
| [spc-tbl.md](spec-tables.md) | テーブル仕様書 |
