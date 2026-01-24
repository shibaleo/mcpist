---
title: MCPist 設計仕様サブコア定義
aliases:
  - dtl-dsn-cor
  - design-sub-core
tags:
  - MCPist
  - architecture
  - sub-core
  - DTL
document-type: detail
document-class: DTL
created: 2026-01-14T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# MCPist 設計仕様サブコア定義

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `current` |
| Version | v1.0 (DAY5) |
| Note | spec-dsn.mdからサブコア要件を抽出 |

---

## 概要

本ドキュメントは、spec-dsn.md（設計仕様書）の中で、コア機能（COR-xxx）を前提とした場合に複数の独立した根拠を持つ要件を「サブコア」として定義する。

**評価基準:**
- コア機能を前提とした場合に、2つ以上の独立した根拠を持つ
- そのコア機能が変わらない限り、変更されない

---

## サブコア要件

### DSN-COR-001: TOON形式統一レスポンス

**前提コア:**
- COR-008 (TOON形式)
- COR-001 (メタツール方式)
- COR-004 (決定論的オーケストレーター)

**独立した根拠（3つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | TOON形式 → JSON比30-40%トークン削減、Context Rot対策 | COR-008 |
| 2 | メタツール方式 → outputSchemaでフィールド宣言、LLMが形式を理解 | COR-001 |
| 3 | 決定論的オーケストレーター → 外部APIレスポンスを正規化、必要フィールドのみ抽出 | COR-004 |

**定義:**
```
TOON形式: items[件数]{フィールド,...}: 値,値,...（1行1レコード）

例:
items[3]{id,title,status}:
  task1,買い物,notStarted
  task2,掃除,completed
  task3,料理,inProgress

outputSchemaによるフィールド宣言:
{
  "name": "github_list_issues",
  "outputSchema": {
    "format": "toon",
    "fields": ["number", "title", "state", "user", "html_url"]
  }
}

変換例（Notion search）:
外部API: {"results": [{"id": "2e62cd76...", "properties": {"Name": ...}}]}
MCPist:  items[1]{id,title,url,created_time}: 2e62cd76...,タスク,...
```

**spec-dsn.mdでの位置:** 2.2 統一レスポンス形式

---

### DSN-COR-002: メタツール設計（get_module_schema / call / batch）

**前提コア:**
- COR-001 (メタツール方式)
- COR-008 (TOON形式)
- COR-007 (Go採用)

**独立した根拠（3つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | メタツール方式 → 3つのメタツールで84ツールを制御 | COR-001 |
| 2 | TOON形式 → 統一入出力形式（JSONL入力、TOON出力） | COR-008 |
| 3 | Go採用 → goroutineによる並列実行、sync.Mapで結果共有 | COR-007 |

**定義:**
```
メタツール:
├─ get_module_schema: モジュールのツール定義を取得（複数モジュール同時可）
├─ call: モジュールのツール実行（単発）
└─ batch: 複数ツールを一括実行（JSONL形式、依存関係指定可能）

JSONL入力フィールド:
├─ module (必須): モジュール名
├─ tool (必須): ツール名
├─ params: パラメータ
├─ id: タスク識別子
├─ after: 依存タスクID配列
└─ output: trueで結果をLLMに返却

実行エンジン（Go）:
├─ afterなし → 即座にgoroutineで並列実行
├─ afterあり → 依存タスク完了を待ってから実行
├─ 変数解決: ${id.items[N].field}形式で前タスク出力参照
└─ エラー時: 循環依存→即失敗、依存タスク失敗→依存先スキップ
```

**spec-dsn.mdでの位置:** 2. メタツール詳細設計

---

### DSN-COR-003: Token Brokerデータモデル（oauth_tokens）

**前提コア:**
- COR-003 (サーバー側認証設計)
- COR-005 (RLS非依存認可)

**独立した根拠（3つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | サーバー側認証 → MCPサーバーがToken Broker経由でトークン取得 | COR-003 |
| 2 | RLS非依存 → role_id + user_idによるアプリケーション層フィルタ | COR-005 |
| 3 | 共有/個人トークンの優先順位解決 → Edge Function内で実装 | COR-003, COR-005 |

**定義:**
```
oauth_tokens テーブル:
├─ id: UUID（PK）
├─ role_id: UUID（FK）必須
├─ user_id: UUID（FK）NULL可
├─ service: VARCHAR（サービス名）
├─ access_token: TEXT（AES-256-GCM暗号化）
├─ refresh_token: TEXT（AES-256-GCM暗号化）
├─ expires_at: TIMESTAMP
└─ scopes: TEXT[]

トークン所有タイプ:
├─ user_id = NULL → ロール共有トークン（adminが設定）
└─ user_id = 値あり → 個人トークン（userが認可）

解決順序:
1. user_id + role_id + service で個人トークンを検索
2. なければ role_id + service で共有トークンを検索
3. どちらもなければエラー（要連携）
```

**spec-dsn.mdでの位置:** 3.3 Token Broker

---

### DSN-COR-004: Tool Sieveデータモデル（権限管理）

**前提コア:**
- COR-001 (メタツール方式)
- COR-003 (サーバー側認証設計)

**独立した根拠（2つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | メタツール方式 → role_permissionsでツール権限を定義、get_module_schema時にフィルタ | COR-001 |
| 2 | サーバー側認証 → user_rolesでユーザーとロールを紐付け | COR-003 |

**定義:**
```
Tool Sieve関連テーブル:
├─ users: ユーザー情報（system_role: admin/user）
├─ roles: ロール定義（権限パターン）
├─ user_roles: ユーザー↔ロール紐付け（UNIQUE: user_id, role_id）
└─ role_permissions: ロール別ツール権限（enabled_modules, tool_masks）

アクセス制御フロー:
1. JWT検証 → user_id抽出
2. user_roles → 割り当てロール取得
3. role_permissions → 有効モジュール/ツール取得
4. get_module_schema時にフィルタリング適用
```

**spec-dsn.mdでの位置:** 3.2 Tool Sieve

---

### DSN-COR-005: 管理UI URL設計（SPA最適化）

**前提コア:**
- COR-006 (Next.js採用)
- COR-003 (サーバー側認証設計)

**独立した根拠（2つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | Next.js採用 → App Router、useSearchParams、middleware.ts | COR-006 |
| 2 | サーバー側認証 → system_role（admin/user）による権限制御 | COR-003 |

**定義:**
```
URL設計方針:
├─ トップレベルは独立したページ（フルページ遷移）
├─ 詳細・編集はモーダル/パネルで開く
├─ URL queryパラメータで選択状態を表現
└─ 深いネストを避ける

基本ページ:
├─ /: ダッシュボード
├─ /profile: プロファイル
├─ /tools: ツール一覧
├─ /users: ユーザー管理（adminのみ）
├─ /roles: ロール管理（adminのみ）
└─ /logs: 監査ログ（adminのみ）

モーダル/パネル（query parameter）:
├─ /tools?connect=notion: 個人アカウント連携
├─ /users?id=xxx: ユーザー詳細
├─ /roles?id=xxx&tab=permissions: 権限設定タブ
└─ /roles?id=xxx&service=notion: サービス設定モーダル
```

**spec-dsn.mdでの位置:** 6. 管理UI設計

---

### DSN-COR-006: モジュールインターフェース統一

**前提コア:**
- COR-009 (モジュールCLI実装)
- COR-007 (Go採用)

**独立した根拠（2つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | モジュールCLI実装 → 全モジュールが同一インターフェースを実装 | COR-009 |
| 2 | Go採用 → interface定義、Token Broker連携のclient.go | COR-007 |

**定義:**
```
モジュールインターフェース:
├─ Name(): モジュール名
├─ Description(): 説明
├─ APIVersion(): 対応APIバージョン
├─ Tools(): ツール定義一覧
└─ Execute(ctx, tool, params): ツール実行

モジュール実装側の責務:
├─ ページネーション: 外部APIのページネーションを隠蔽
├─ レートリミット: サービスごとの制限に応じた制御
├─ 正規化: 外部APIレスポンスをTOON形式に変換
└─ デフォルト上限: 500件（モジュールごとに設定可能）

登録モジュール（84ツール）:
notion, github, jira, confluence, supabase,
google_calendar, microsoft_todo, rag
```

**spec-dsn.mdでの位置:** 7. モジュール設計

---

## 非サブコア要件（単一根拠）

以下は重要な要件だが、コア機能からの導出が単一であるためサブコアとしない。

| 要件 | 根拠数 | 理由 |
|------|--------|------|
| Tasks API対応 | 1 | MCP仕様準拠からの単一導出 |
| URL Elicitation | 1 | MCP仕様準拠からの単一導出 |
| エラーコード体系 | 1 | 運用品質からの単一導出 |
| 危険操作フラグ | 1 | セキュリティ要件からの単一導出 |
| AES-256-GCM暗号化 | 1 | Supabase Vault標準からの単一導出 |
| リトライ戦略 | 1 | 運用品質からの単一導出 |

---

## サブコアマトリックス

| ID | サブコア要件 | 前提コア | 根拠数 |
|----|--------------|----------|--------|
| DSN-COR-001 | TOON形式統一レスポンス | COR-008, COR-001, COR-004 | 3 |
| DSN-COR-002 | メタツール設計 | COR-001, COR-008, COR-007 | 3 |
| DSN-COR-003 | Token Brokerデータモデル | COR-003, COR-005 | 3 |
| DSN-COR-004 | Tool Sieveデータモデル | COR-001, COR-003 | 2 |
| DSN-COR-005 | 管理UI URL設計 | COR-006, COR-003 | 2 |
| DSN-COR-006 | モジュールインターフェース | COR-009, COR-007 | 2 |

---

## 関連ドキュメント

- [dtl-core.md](dtl-core.md) - コア機能定義
- [spec-dsn.md](../DAY4/spec-dsn.md) - 設計仕様書
