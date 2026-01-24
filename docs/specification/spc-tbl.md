# MCPist テーブル仕様書（spc-tbl）

## ドキュメント管理情報

| 項目      | 値                   |
| ------- | ------------------- |
| Status  | `draft`             |
| Version | v2.0                |
| Note    | Table Specification |

---

## 概要

本ドキュメントは、MCPistで使用するデータベーステーブルの配置と役割を定義する。

詳細なテーブル設計は以下で規定する：
- dsn-tbl.md: テーブル設計書（ER図、リレーション）
- dtl-dsn-tbl.md: テーブル詳細設計書（列定義、制約、インデックス）

---

## スキーマ構成

MCPistのテーブルは `mcpist` スキーマに配置する。

| スキーマ | 用途 | 管理 |
|---------|------|------|
| auth | 共通認証基盤（auth.users等） | Supabase管理 |
| vault | 暗号化ストア（vault.secrets） | Supabase管理 |
| mcpist | MCPist関連テーブル | MCPist |

**備考:**
- 同一Supabaseプロジェクトで複数サービス運用を想定
- auth.usersを共通ユーザー基盤として共有
- テーブル名はスキーマ修飾（`mcpist.users`）で一意に識別

---

## インタラクション別データ要件

Data Store（DST）が関わるインタラクションから導出されるテーブル要件。

### ITR-REL-008: HDL → DST（ユーザー設定取得）

MCP Handlerがリクエスト処理時に取得する情報。

| データ項目 | 説明 | 関連テーブル |
|-----------|------|-------------|
| account_status | active/suspended/disabled | users |
| credit_balance | クレジット残高（free + paid） | credits |
| enabled_modules | 有効なモジュール一覧 | module_settings |
| tool_settings | ツール単位の有効/無効 | tool_settings |
| prompts | ユーザー定義プロンプト | prompts |

### ITR-REL-011: MOD → DST（クレジット消費）

ツール実行成功時のクレジット消費記録。

| データ項目 | 説明 | 関連テーブル |
|-----------|------|-------------|
| user_id | ユーザーID | credits |
| module | モジュール名 | credit_transactions |
| tool | ツール名 | credit_transactions |
| amount | 消費量 | credit_transactions |
| request_id | リクエスト追跡用ID | credit_transactions |
| task_id | batch内タスクID（runの場合null） | credit_transactions |

### ITR-REL-016: CON → DST（ツール設定登録）

User Consoleからのユーザー設定管理。

| データ項目 | 説明 | 操作 | 関連テーブル |
|-----------|------|------|-------------|
| enabled_modules | 有効化モジュール一覧 | 登録/更新 | module_settings |
| tool_settings | ツール有効/無効設定 | 登録/更新 | tool_settings |
| prompts | ユーザー定義プロンプト | CRUD | prompts |
| credit_balance | クレジット残高 | 参照のみ | credits |
| account_status | アカウント状態 | 参照のみ | users |
| usage_stats | 利用統計 | 参照のみ | credit_transactions |

### ITR-REL-021: PSP → DST（有料クレジット情報）

決済完了Webhookによるクレジット加算。

| データ項目 | 説明 | 関連テーブル |
|-----------|------|-------------|
| event_id | Webhookイベント識別子（冪等性） | processed_webhook_events |
| paid_credits | 有料クレジット加算額 | credits |
| payment_record | 決済記録 | credit_transactions |

---

## テーブル一覧

### ユーザー・アカウント

| テーブル | 役割 | 参照インタラクション |
|---------|------|---------------------|
| users | ユーザー情報、account_status | ITR-REL-008, 016 |
| api_keys | MCP接続用APIキー（ハッシュ保存） | ITR-REL-005 |

### クレジット・課金

| テーブル | 役割 | 参照インタラクション |
|---------|------|---------------------|
| credits | free_credits, paid_credits残高 | ITR-REL-008, 011, 016, 021 |
| credit_transactions | クレジット増減履歴 | ITR-REL-011, 016, 021 |
| processed_webhook_events | PSP Webhook冪等性 | ITR-REL-021 |

### モジュール・ツール設定

| テーブル | 役割 | 参照インタラクション |
|---------|------|---------------------|
| modules | モジュール定義（マスタ） | ITR-REL-008 |
| module_settings | ユーザー×モジュール有効/無効 | ITR-REL-008, 016 |
| tool_settings | ユーザー×ツール有効/無効 | ITR-REL-008, 016 |

### プロンプト

| テーブル | 役割 | 参照インタラクション |
|---------|------|---------------------|
| prompts | ユーザー定義プロンプト | ITR-REL-008, 016 |

### Token Vault

| テーブル | 役割 | 参照インタラクション |
|---------|------|---------------------|
| vault.secrets | 暗号化トークン本体（Supabase Vault） | ITR-REL-010, 013, 015 |

**命名規則:** `{user_id}:{service}`（例: `e9467a51-a385-488f-86d0-16b68385ed04:notion`）

### Auth（Supabase管理）

| テーブル | 役割 | 備考 |
|---------|------|------|
| auth.users | ユーザー認証情報 | Supabase Auth管理 |

---

## テーブル概要設計

詳細な列定義・制約はdtl-dsn-tbl.mdで規定する。

### users

| 列（概要） | 説明 |
|-----------|------|
| id | PK、auth.usersへのFK |
| account_status | active / suspended / disabled |
| preferences | jsonb: UI設定（theme, background, accent） |
| created_at, updated_at | タイムスタンプ |

### credits

| 列（概要） | 説明 |
|-----------|------|
| user_id | PK、usersへのFK |
| free_credits | 無料クレジット（上限1000、毎月補充） |
| paid_credits | 有料クレジット（上限なし） |
| updated_at | 最終更新日時 |

### credit_transactions

| 列（概要） | 説明 |
|-----------|------|
| id | PK |
| user_id | usersへのFK |
| type | consume / purchase / monthly_reset |
| amount | 増減量（消費は負、加算は正） |
| module, tool | 消費時のモジュール・ツール名 |
| request_id, task_id | リクエスト追跡用 |
| created_at | タイムスタンプ |

### module_settings

| 列（概要） | 説明 |
|-----------|------|
| user_id | PK（複合）、usersへのFK |
| module_id | PK（複合）、modulesへのFK |
| enabled | 有効/無効 |

### tool_settings

| 列（概要） | 説明 |
|-----------|------|
| user_id | PK（複合）、usersへのFK |
| module_id | PK（複合）、modulesへのFK |
| tool_name | PK（複合）、ツール名 |
| enabled | 有効/無効 |

### prompts

| 列（概要） | 説明 |
|-----------|------|
| id | PK |
| user_id | usersへのFK |
| module_id | modulesへのFK |
| name | プロンプト名 |
| content | プロンプト内容 |

### modules

| 列（概要） | 説明 |
|-----------|------|
| id | PK |
| name | モジュール名（notion, github等） |
| status | active / coming_soon / maintenance / beta / deprecated / disabled |

### api_keys

| 列（概要） | 説明 |
|-----------|------|
| id | PK |
| user_id | usersへのFK |
| key_hash | SHA-256ハッシュ |
| name | キー名（ユーザー管理用） |
| expires_at | 有効期限 |

### processed_webhook_events

| 列（概要） | 説明 |
|-----------|------|
| event_id | PK（PSPのevent.id） |
| user_id | usersへのFK（参照用） |
| processed_at | 処理日時 |

---

## テーブル間の関係

```
auth.users (Supabase Auth)
    │
    │ user_id (FK)
    ▼
┌─────────────────────────────────────────────────────────────┐
│                        mcpist                                │
│                                                              │
│  users ──────┬──── credits                                  │
│     │        │         │                                     │
│     │        │         └──── credit_transactions            │
│     │        │                                               │
│     │        ├──── module_settings ──── modules             │
│     │        │                                               │
│     │        ├──── tool_settings ─────────┘                 │
│     │        │                                               │
│     │        ├──── prompts ───────────────┘                 │
│     │        │                                               │
│     │        ├──── api_keys                                 │
│     │        │                                               │
│     │        └──── processed_webhook_events                 │
│                                                              │
│  vault.secrets（{user_id}:{service} 命名規則で識別）          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](./spc-sys.md) | システム仕様書 |
| [spc-itr.md](./spc-itr.md) | インタラクション仕様書 |
| [itr-dst.md](./interaction/itr-dst.md) | Data Store インタラクション仕様 |
| [dtl-spc-credit-model.md](./dtl-spc/dtl-spc-credit-model.md) | クレジットモデル詳細仕様 |
| [itf-tvl.md](./dtl-spc/itf-tvl.md) | Token Vault API仕様 |
| dsn-tbl.md | テーブル設計書（未作成） |
| dtl-dsn-tbl.md | テーブル詳細設計書（未作成） |
