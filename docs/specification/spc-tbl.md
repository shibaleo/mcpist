# MCPist テーブル仕様書（spc-tbl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | Table Specification |

---

## 概要

本ドキュメントは、MCPistで使用するデータベーステーブルの配置と役割を定義する。

---

## スキーマ構成

MCPistのテーブルは `mcpist` スキーマに配置する。

| スキーマ | 用途 | 管理 |
|---------|------|------|
| auth | 共通認証基盤（auth.users等） | Supabase管理 |
| vault | 暗号化ストア（vault.secrets） | Supabase管理 |
| mcpist | MCPist関連テーブル（ENT + TVL） | MCPist |

**備考:**
- 同一Supabaseプロジェクトで複数サービス運用を想定（例：将来のpkmistスキーマ等）
- auth.usersを共通ユーザー基盤として共有
- テーブル名はスキーマ修飾（`mcpist.users`）で一意に識別

---

## テーブル配置

### Entitlement Store（ENT）

ユーザーの権限・課金・設定情報を管理するテーブル群。

| テーブル（暫定名） | 役割 | 主な参照元 |
|------------------|------|-----------|
| users | ユーザー情報、アカウント状態 | AUS, SRV, CON |
| subscriptions | 課金状態、PSP連携情報 | SRV, CON, PSP |
| plans | プラン定義（Rate Limit上限、Quota上限、Credit有効フラグ） | SRV, CON |
| user_module_preferences | ユーザーごとのモジュール有効/無効 | SRV, CON |
| usage | Quota使用量追跡（月次リセット） | SRV, CON |
| credits | Credit残高 | SRV, CON |
| credit_transactions | Credit増減履歴（ビジネスデータ） | CON |
| tool_costs | ツールごとのCredit消費量定義 | SRV |
| modules | モジュール定義（動的有効/無効） | SRV, REG |
| mcp_tokens | MCP接続用Long-lived Token（SHA-256ハッシュ保存） | SRV, CON |
| processed_webhook_events | PSP Webhook冪等性（event_id重複防止） | ENT |

**備考:**
- ロール管理テーブルは運用仕様検討後に再評価
- 管理者の権限範囲（非プログラマ想定）を明確化した後、必要に応じて追加

---

### Token Vault（TVL）

外部サービスのOAuthトークンを管理するテーブル群。

| テーブル（暫定名） | 役割 | 主な参照元 |
|------------------|------|-----------|
| oauth_tokens | ユーザー×サービスの紐づけ、vault.secretsへの参照 | MOD, CON |
| vault.secrets | 暗号化トークン本体（Supabase Vault組み込み） | oauth_tokens経由 |

---

### Auth Server（AUS）

Supabase Authが管理するテーブル（実装範囲外）。

| テーブル | 役割 | 備考 |
|----------|------|------|
| auth.users | ユーザー認証情報 | Supabase Auth管理、ENT.usersから参照 |

---

## テーブル間の関係

```
auth.users (Supabase Auth)
    │
    │ user_id (FK)
    ▼
┌─────────────────────────────────────────────────────────────┐
│                     Entitlement Store                        │
│                                                              │
│  users ──────┬──────── subscriptions ──── plans             │
│     │        │                              ▲                │
│     │        │                              │                │
│     │        ├──────── user_module_preferences              │
│     │        │                              │                │
│     │        ├──────── usage                │                │
│     │        │                              │                │
│     │        ├──────── credits              │                │
│     │        │              │               │                │
│     │        │              ▼               │                │
│     │        │         credit_transactions  │                │
│     │        │                              │                │
│     │        └──────────────────────────────┘                │
│     │                                                        │
│     │  tool_costs    modules    processed_webhook_events     │
│     │                                                        │
└─────┼────────────────────────────────────────────────────────┘
      │
      │ user_id (FK)
      ▼
┌─────────────────────────────────────────────────────────────┐
│                       Token Vault                            │
│                                                              │
│  oauth_tokens ──────────▶ vault.secrets                     │
│  (user_id, service)       (暗号化トークン本体)                │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](./spc-sys.md) | システム仕様書 |
| [spc-itr.md](./spc-itr.md) | インタラクション仕様書 |
| [itr-dst.md](./interaction/itr-dst.md) | Data Store詳細仕様 |
| [itf-tvl.md](./dtl-spc/itf-tvl.md) | Token Vault API仕様 |
