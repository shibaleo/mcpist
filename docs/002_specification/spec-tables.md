# MCPist テーブル仕様書（spc-tbl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v3.0 (Sprint-012) |
| Note | Table Specification — 現行実装に基づく全面改訂 |

---

## 概要

本ドキュメントは、MCPist のデータベーステーブル構成と列定義を規定する。
すべてのテーブルは PostgreSQL の `mcpist` スキーマに配置される。

---

## テーブル一覧

| # | テーブル | 役割 |
|---|---|---|
| 1 | plans | プランマスタ (free / plus / team) |
| 2 | users | ユーザーアカウント |
| 3 | modules | モジュールマスタ (Server から同期) |
| 4 | module_settings | ユーザー × モジュール 有効/無効 |
| 5 | tool_settings | ユーザー × モジュール × ツール 有効/無効 |
| 6 | prompts | ユーザー定義プロンプト |
| 7 | api_keys | JWT ベース API キー |
| 8 | user_credentials | 外部サービス資格情報 (AES-256-GCM 暗号化) |
| 9 | oauth_apps | OAuth プロバイダ設定 (client_secret 暗号化) |
| 10 | usage_log | ツール実行ログ |
| 11 | processed_webhook_events | Stripe Webhook 冪等性管理 |

---

## テーブル定義

### plans

プランマスタ。

| 列 | 型 | 制約 | 説明 |
|---|---|---|---|
| id | TEXT | PK | プラン ID (free / plus / team) |
| name | TEXT | NOT NULL | 表示名 |
| daily_limit | INTEGER | NOT NULL | 日次使用量上限 |
| price_monthly | INTEGER | DEFAULT 0 | 月額料金 |
| stripe_price_id | TEXT | | Stripe Price ID |
| features | JSONB | NOT NULL, DEFAULT '{}' | 機能フラグ |

---

### users

ユーザーアカウント。Clerk の認証情報と紐付く。

| 列 | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| clerk_id | TEXT | UNIQUE | Clerk ユーザー ID |
| account_status | TEXT | NOT NULL, DEFAULT 'active' | active / suspended / disabled |
| plan_id | TEXT | NOT NULL, DEFAULT 'free', FK → plans(id) | |
| display_name | TEXT | | 表示名 |
| avatar_url | TEXT | | アバター URL |
| email | TEXT | | メールアドレス |
| role | TEXT | NOT NULL, DEFAULT 'user' | user / admin |
| stripe_customer_id | TEXT | UNIQUE | Stripe Customer ID |
| settings | JSONB | NOT NULL, DEFAULT '{}' | ユーザー設定 |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |

**インデックス:**

| インデックス | 列 | 備考 |
|---|---|---|
| idx_users_clerk_id | clerk_id | WHERE clerk_id IS NOT NULL |
| idx_users_stripe_customer_id | stripe_customer_id | WHERE stripe_customer_id IS NOT NULL |
| idx_users_account_status | account_status | |

---

### modules

モジュールマスタ。Server 起動時に Go の modules/ 実装から同期される。

| 列 | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| name | TEXT | NOT NULL, UNIQUE | モジュール名 (notion, github 等) |
| status | TEXT | NOT NULL, DEFAULT 'active' | active / coming_soon / maintenance / beta / deprecated / disabled |
| tools | JSONB | NOT NULL, DEFAULT '[]' | ツール定義 |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |

**インデックス:**

| インデックス | 列 |
|---|---|
| idx_modules_status | status |

---

### module_settings

ユーザーごとのモジュール有効/無効設定。

| 列 | 型 | 制約 | 説明 |
|---|---|---|---|
| user_id | UUID | PK (複合), FK → users(id) ON DELETE CASCADE | |
| module_id | UUID | PK (複合), FK → modules(id) ON DELETE CASCADE | |
| enabled | BOOLEAN | NOT NULL, DEFAULT true | |
| description | TEXT | NOT NULL, DEFAULT '' | ユーザー定義の説明文 |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |

---

### tool_settings

ユーザーごとのツール単位の有効/無効設定。

| 列 | 型 | 制約 | 説明 |
|---|---|---|---|
| user_id | UUID | PK (複合), FK → users(id) ON DELETE CASCADE | |
| module_id | UUID | PK (複合), FK → modules(id) ON DELETE CASCADE | |
| tool_id | TEXT | PK (複合) | ツール ID |
| enabled | BOOLEAN | NOT NULL, DEFAULT true | |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |

---

### prompts

ユーザー定義の MCP プロンプト。

| 列 | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| user_id | UUID | NOT NULL, FK → users(id) ON DELETE CASCADE | |
| module_id | UUID | FK → modules(id) ON DELETE SET NULL | 紐付きモジュール (任意) |
| name | TEXT | NOT NULL | プロンプト名 |
| description | TEXT | | 説明 |
| content | TEXT | NOT NULL | プロンプト本文 |
| enabled | BOOLEAN | NOT NULL, DEFAULT true | |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |

**インデックス:**

| インデックス | 列 |
|---|---|
| idx_prompts_user_id | user_id |

---

### api_keys

MCP クライアント用 JWT ベース API キー。

| 列 | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| user_id | UUID | NOT NULL, FK → users(id) ON DELETE CASCADE | |
| jwt_kid | TEXT | | JWT Key ID |
| key_prefix | TEXT | NOT NULL | mpt_* プレフィックス (表示用) |
| name | TEXT | NOT NULL | キー名 |
| expires_at | TIMESTAMPTZ | | 有効期限 |
| last_used_at | TIMESTAMPTZ | | 最終使用日時 |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |

**インデックス:**

| インデックス | 列 |
|---|---|
| idx_api_keys_user_id | user_id |

---

### user_credentials

外部サービスの資格情報。AES-256-GCM で暗号化して保存。

| 列 | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| user_id | UUID | NOT NULL, FK → users(id) ON DELETE CASCADE | |
| module | TEXT | NOT NULL | モジュール名 |
| encrypted_credentials | TEXT | | 暗号化された資格情報 |
| key_version | INTEGER | NOT NULL, DEFAULT 1 | 暗号化キーバージョン |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |

**制約:** UNIQUE (user_id, module)

**インデックス:**

| インデックス | 列 |
|---|---|
| idx_user_credentials_user_module | (user_id, module) |

**暗号化形式:** `v1:base64(nonce||ciphertext||tag)`

---

### oauth_apps

管理者が登録する OAuth プロバイダ設定。client_secret は AES-256-GCM で暗号化。

| 列 | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| provider | TEXT | NOT NULL, UNIQUE | プロバイダ名 |
| client_id | TEXT | NOT NULL | OAuth Client ID |
| encrypted_client_secret | TEXT | | 暗号化された Client Secret |
| redirect_uri | TEXT | NOT NULL | リダイレクト URI |
| enabled | BOOLEAN | NOT NULL, DEFAULT true | |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |

---

### usage_log

ツール実行の使用量記録。非同期で記録される。

| 列 | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| user_id | UUID | NOT NULL, FK → users(id) ON DELETE CASCADE | |
| meta_tool | TEXT | NOT NULL | メタツール名 (run / batch) |
| request_id | TEXT | | リクエスト追跡 ID |
| details | JSONB | NOT NULL | 実行詳細 |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |

**インデックス:**

| インデックス | 列 | 備考 |
|---|---|---|
| idx_usage_log_user_id | user_id | |
| idx_usage_log_created_at | created_at | DESC |
| idx_usage_log_request_id | request_id | WHERE request_id IS NOT NULL |

---

### processed_webhook_events

Stripe Webhook の冪等性管理。

| 列 | 型 | 制約 | 説明 |
|---|---|---|---|
| event_id | TEXT | PK | Stripe イベント ID |
| user_id | UUID | NOT NULL, FK → users(id) ON DELETE CASCADE | |
| processed_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |

---

## テーブル間の関係

```
plans
  │
  │ plan_id (FK)
  ▼
users ──────┬──── module_settings ──── modules
            │                           ↑
            ├──── tool_settings ────────┘
            │                           ↑
            ├──── prompts ─────────────┘ (SET NULL)
            │
            ├──── api_keys
            │
            ├──── user_credentials
            │
            ├──── usage_log
            │
            └──── processed_webhook_events

oauth_apps (独立)
```

すべての users への FK は ON DELETE CASCADE。
prompts.module_id のみ ON DELETE SET NULL。

---

## マイグレーション

マイグレーションファイルは `database/migrations/` に配置。

| # | ファイル | 内容 |
|---|---|---|
| 1 | 00000000000001_baseline.sql | 全テーブル作成 |
| 2 | 00000000000002_add_encrypted_client_secret.sql | oauth_apps に encrypted_client_secret 追加 |
| 3 | 00000000000003_drop_plaintext_columns.sql | user_credentials.credentials, oauth_apps.client_secret を削除 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|---|---|
| [spec-systems.md](./spec-systems.md) | システム仕様書 |
| [spec-design.md](./spec-design.md) | 設計仕様書 |
| [spec-infrastructure.md](./spec-infrastructure.md) | インフラ仕様書 |
| [spc-sec.md](spec-security.md) | セキュリティ仕様書 |
