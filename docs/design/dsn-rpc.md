# RPC関数設計書（dsn-rpc）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| Note | RPC Function Design |

---

## 概要

本ドキュメントは、インタラクション仕様とテーブル設計から導出されるSupabase RPC関数の設計を定義する。

### 設計方針

1. **インタラクション駆動**: 各RPC関数はインタラクション仕様の要件を満たす
2. **単一責務**: 1つのRPC関数は1つの処理を行う
3. **最小権限**: 必要な情報のみを返却する
4. **命名規則**: `動詞_対象` 形式（例: `lookup_user_by_key_hash`, `get_user_context`）

---

## RPC関数カテゴリ

### MCP Server向け（service_role）

| RPC関数 | インタラクション | 説明 |
|---------|-----------------|------|
| lookup_user_by_key_hash | ITR-REL-005 | APIキーハッシュからuser_idを取得 |
| get_user_context | ITR-REL-008 | ツール実行に必要なユーザー情報取得 |
| consume_credit | ITR-REL-011 | クレジット消費・履歴記録 |
| get_module_token | ITR-REL-010 | モジュール用トークン取得 |
| update_module_token | ITR-REL-010 | リフレッシュ後のトークン保存 |

### Console Frontend向け（authenticated）

| RPC関数                    | 説明                    |
| ------------------------ | --------------------- |
| generate_api_key         | APIキー生成               |
| list_api_keys            | APIキー一覧取得             |
| revoke_api_key           | APIキー削除（論理削除）         |
| list_service_connections | サービス接続一覧（OAuth/PAT両方） |
| upsert_service_token     | サービストークン登録/更新         |
| delete_service_token     | サービストークン削除            |

### Console API Routes向け（service_role）

| RPC関数 | インタラクション | 説明 |
|---------|-----------------|------|
| add_paid_credits | ITR-REL-021 | 有料クレジット加算（Webhook処理） |

### Cron向け

| RPC関数 | 説明 |
|---------|------|
| reset_free_credits | 月初の無料クレジット補充 |

### 用語の使い分け

| 用語 | 視点 | 使用箇所 |
|------|------|----------|
| module | mcpist内部 | MCP Server向けRPC（get_user_context, consume_credit, get_module_token, update_module_token） |
| service | ユーザー向け | Console Frontend向けRPC（list_service_connections等） |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [dtl-dsn-rpc.md](./dtl-dsn-rpc.md) | RPC関数詳細設計書 |
| [dtl-dsn-tbl.md](./dtl-dsn-tbl.md) | テーブル詳細設計書 |
| [spc-tbl.md](../specification/spc-tbl.md) | テーブル仕様書 |
| [itr-dst.md](../specification/interaction/itr-dst.md) | Data Store インタラクション仕様 |
| [dtl-spc-credit-model.md](../specification/dtl-spc/dtl-spc-credit-model.md) | クレジットモデル詳細仕様 |
