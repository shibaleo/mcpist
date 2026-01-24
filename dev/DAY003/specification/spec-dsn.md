---
title: MCPist 設計仕様書（spec-dsn）
aliases:
  - spec-dsn
  - MCPist-design-specification
tags:
  - MCPist
  - specification
  - design
document-type:
  - specification
document-class: specification
created: 2026-01-12T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# MCPist 設計仕様書（spec-dsn）

本ドキュメントは、MCPistの詳細設計を定義する。

---

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `archived` |
| Version | v1.0 (MEGA版) |
| Superseded by | [go-mcp-dev/dev/DAY3/specification/spec-dsn.md](../../../go-mcp-dev/dev/DAY3/specification/spec-dsn.md) |
| 差分概要 | go-mcp-dev版はMCPプロトコル2025-11-25対応、TOON形式採用、Tasks/Elicitation対応、ロールベースデータモデル（tenants, roles, user_roles等）、詳細な管理UI設計 |

---

## 1. API設計

### 1.1 MCPサーバーエンドポイント

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/health` | ヘルスチェック |
| POST | `/mcp` | MCP Protocol (JSON-RPC 2.0 over SSE) |

### 1.2 MCPプロトコル

#### 1.2.1 サポートメソッド

| メソッド | 説明 |
|---------|------|
| `initialize` | 接続初期化、サーバー情報返却 |
| `tools/list` | メタツール一覧を返却 |
| `tools/call` | メタツール実行 |

#### 1.2.2 initialize レスポンス

- `protocolVersion`: "2024-11-05"
- `capabilities`: `{tools: {}}`
- `serverInfo`: name/version

#### 1.2.3 tools/list レスポンス

3つのメタツール:
- `get_module_schema`: モジュールのツール定義を取得（複数モジュール同時取得可能、1セッション1回推奨）
- `call`: モジュールのツール実行
- `batch`: 複数ツールを一括実行（JSONL形式、依存関係指定可能）

---

## 2. メタツール詳細設計

### 2.1 get_module_schema

**入力:** `{"modules": ["notion", "github"]}` （配列で複数指定可能）

**出力:** モジュールごとのスキーマ配列。各要素は: モジュール名、description、apiVersion、tools配列（name/description/inputSchema/dangerousフラグ）

### 2.2 call

**入力:** `module`, `tool_name`, `params`

**出力:** `{"success": true/false, "result": {...}}`

### 2.3 batch

**入力形式:** JSONL（1行1タスク）

**フィールド:**
- `id` (必須): タスク識別子
- `module`, `tool` (必須): 実行対象
- `params`: パラメータ
- `after`: 依存タスクID
- `output`: `true`で結果をLLMに返却

**変数参照:** `${id.path}` 形式で前タスクの出力参照（例: `${search.results[0].id}`）

**出力:** `{"success": true, "results": {...}, "errors": {...}}`

---

## 3. データモデル

### 3.1 Vault データ構造

#### users テーブル

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | ユーザーID（PK） |
| email | VARCHAR | メールアドレス |
| created_at | TIMESTAMP | 作成日時 |
| updated_at | TIMESTAMP | 更新日時 |

#### accounts テーブル

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | アカウントID（PK） |
| user_id | UUID | ユーザーID（FK） |
| name | VARCHAR | アカウント名（個人用、仕事用等） |
| created_at | TIMESTAMP | 作成日時 |

#### oauth_tokens テーブル

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | トークンID（PK） |
| account_id | UUID | アカウントID（FK） |
| service | VARCHAR | サービス名（notion, github等） |
| access_token | TEXT | アクセストークン（暗号化） |
| refresh_token | TEXT | リフレッシュトークン（暗号化） |
| expires_at | TIMESTAMP | 有効期限 |
| scopes | TEXT[] | 許可スコープ |
| created_at | TIMESTAMP | 作成日時 |
| updated_at | TIMESTAMP | 更新日時 |

#### user_profiles テーブル

| カラム | 型 | 説明 |
|--------|-----|------|
| id | UUID | プロフィールID（PK） |
| account_id | UUID | アカウントID（FK） |
| enabled_modules | TEXT[] | 有効モジュール一覧 |
| tool_masks | JSONB | ツール別の有効/無効設定 |
| created_at | TIMESTAMP | 作成日時 |
| updated_at | TIMESTAMP | 更新日時 |

### 3.2 ER図

```
users
  │
  │ 1:N
  ▼
accounts
  │
  │ 1:N              1:1
  ├────────────▶ user_profiles
  │
  │ 1:N
  ▼
oauth_tokens
```

---

## 4. エラーハンドリング

### 4.1 エラーコード体系

| コード範囲 | カテゴリ |
|-----------|---------|
| 1000-1999 | 認証エラー |
| 2000-2999 | 入力検証エラー |
| 3000-3999 | 外部APIエラー |
| 4000-4999 | 内部エラー |

### 4.2 エラーコード一覧

| コード | 名前 | 説明 |
|--------|------|------|
| 1001 | INVALID_JWT | JWTが無効 |
| 1002 | JWT_EXPIRED | JWTが期限切れ |
| 1003 | UNAUTHORIZED | 認証が必要 |
| 2001 | INVALID_MODULE | 存在しないモジュール |
| 2002 | INVALID_TOOL | 存在しないツール |
| 2003 | INVALID_PARAMS | パラメータ不正 |
| 3001 | EXTERNAL_API_ERROR | 外部APIエラー |
| 3002 | TOKEN_REFRESH_FAILED | トークンリフレッシュ失敗 |
| 3003 | RATE_LIMITED | レート制限 |
| 4001 | INTERNAL_ERROR | 内部エラー |
| 4002 | TIMEOUT | タイムアウト |

### 4.3 エラーレスポンス形式

JSON-RPC 2.0形式: `{"jsonrpc": "2.0", "id": N, "error": {"code": N, "message": "...", "data": {...}}}`

### 4.4 リトライ戦略

| エラー種別 | リトライ | 戦略 |
|-----------|---------|------|
| 認証エラー | × | 即座にエラー返却 |
| レート制限 | ○ | Exponential backoff |
| 一時的エラー | ○ | 最大3回、1秒間隔 |
| 永続的エラー | × | 即座にエラー返却 |

---

## 5. セキュリティ設計

### 5.1 暗号化仕様

| 項目 | 仕様 |
|------|------|
| アルゴリズム | AES-256-GCM |
| キー長 | 256bit |
| IV長 | 96bit |
| 認証タグ長 | 128bit |

### 5.2 トークン暗号化

Supabase Vaultが暗号化・復号化を内部で処理。アプリケーション側での実装は不要。

- 保存時: plaintext → Vault（AES-256-GCM暗号化） → DB
- 取得時: DB → Vault（復号化） → plaintext

### 5.3 危険操作フラグ

以下の操作は`dangerous: true`としてマーク:

| モジュール | ツール | 理由 |
|-----------|--------|------|
| Notion | delete_page | データ削除 |
| GitHub | delete_repository | データ削除 |
| GitHub | merge_pull_request | 変更の確定 |
| Jira | delete_issue | データ削除 |
| Supabase | execute_sql | 任意SQL実行 |

---

## 6. 管理UI設計

### 6.1 画面遷移

- `/login` → `/dashboard`
- `/dashboard` → `/services`, `/accounts`, `/logs`
- `/services` → `/services/:service/setup`（各サービスのOAuth設定）
- `/accounts` → `/accounts/new`（アカウント追加）

### 6.2 API エンドポイント（管理UI用）

| メソッド | パス | 説明 |
|---------|------|------|
| POST | `/api/auth/login` | ログイン |
| POST | `/api/auth/logout` | ログアウト |
| GET | `/api/services` | サービス一覧取得 |
| POST | `/api/services/:service/oauth/start` | OAuth開始 |
| GET | `/api/services/:service/oauth/callback` | OAuthコールバック |
| DELETE | `/api/services/:service/token` | トークン削除 |
| GET | `/api/accounts` | アカウント一覧 |
| POST | `/api/accounts` | アカウント作成 |
| PUT | `/api/accounts/:id` | アカウント更新 |
| DELETE | `/api/accounts/:id` | アカウント削除 |
| GET | `/api/logs` | ログ取得 |

---

## 7. モジュール設計

### 7.1 モジュールインターフェース

各モジュールは以下を実装:
- `Name()`, `Description()`, `APIVersion()`: メタ情報
- `Tools()`: ツール定義一覧
- `Execute(ctx, tool, params)`: ツール実行

### 7.2 モジュール登録

8モジュールをレジストリに登録: notion, github, jira, confluence, supabase, google_calendar, microsoft_todo, rag

---

## 8. 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [要件仕様書](spec-req.md) | 要件定義 |
| [システム仕様書](spec-sys.md) | システム全体像 |
| [インフラ仕様書](spec-inf.md) | インフラ構成 |
| [運用仕様書](spec-ops.md) | 運用設計 |
