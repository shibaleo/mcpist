# MCPist Week 1 設計サマリー

## 概要

**期間**: DAY1〜DAY7
**成果**: 設計骨格完成、個別開発可能な状態に到達

---

## 設計ドキュメント完成状況

### 仕様書（DAY5）

| ドキュメント | 内容 | 状態 |
|-------------|------|------|
| spec-req.md | 要件仕様（49機能要件、37非機能要件） | ✅ 完成 |
| spec-sys.md | システム仕様（コア機能、コンポーネント構成） | ✅ 完成 |
| spec-dsn.md | 設計仕様（API、データモデル、管理UI） | ✅ 完成 |
| spec-inf.md | インフラ仕様（Koyeb/Supabase/Vercel、CI/CD） | ✅ 完成 |

### インターフェース・認証（DAY6）

| ドキュメント | 内容 | 状態 |
|-------------|------|------|
| spec-ifc.md | インターフェース仕様（全43 IFC定義） | ✅ 完成 |
| ARD-008 | 認証基盤決定（Supabase一極集中） | ✅ 決定済 |
| dsn-usr-ifc.md | 管理UIデザイン仕様（v0プロンプト、画面設計） | ✅ 完成 |

### 詳細設計（DAY7）

| ドキュメント | 内容 | 状態 |
|-------------|------|------|
| dsn-module-registry.md | Module Registry設計（2層抽象化、DAGバッチ、TOON） | ✅ 完成 |
| dsn-permission-system.md | 権限システム設計（Tool Sieve多層防御） | ✅ 完成 |
| dsn-billing.md | 課金システム設計（Stripe Checkout） | ✅ 完成 |
| dsn-infrastructure.md | インフラ詳細（Worker + Koyeb + Fly.io） | ✅ 完成 |
| dsn-load-management.md | 負荷管理設計（2層Rate Limit、監視） | ✅ 完成 |
| dsn-security.md | セキュリティ設計（リスク分析、対策） | ✅ 完成 |

---

## アーキテクチャ概要

### 全体構成

```
┌─────────────────────────────────────────────────────────────────────┐
│                         MCP Host (Claude Code等)                     │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ MCP Protocol (SSE + JSON-RPC)
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Cloudflare Worker                             │
│                    (JWT検証、Rate Limit、課金チェック)                │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                         ┌──────────┴──────────┐
                         ▼                     ▼
                    ┌─────────┐           ┌─────────┐
                    │  Koyeb  │           │ Fly.io  │
                    │(Primary)│           │(Standby)│
                    └────┬────┘           └────┬────┘
                         │                     │
                         └──────────┬──────────┘
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         MCPist Server (Go)                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  │
│  │   Auth MW   │→ │ Tool Sieve  │→ │  Registry   │                  │
│  │ (JWT検証)   │  │ (権限Filter)│  │ (2層抽象化) │                  │
│  └─────────────┘  └─────────────┘  └──────┬──────┘                  │
│                                           │                          │
│        ┌──────────────────────────────────┼──────────────────┐      │
│        ▼                                  ▼                  ▼      │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐        │
│  │  Notion   │  │  GitHub   │  │   Jira    │  │    ...    │        │
│  │  Module   │  │  Module   │  │  Module   │  │  Module   │        │
│  └─────┬─────┘  └─────┬─────┘  └─────┬─────┘  └─────┬─────┘        │
│        │              │              │              │                │
│        └──────────────┴──────────────┴──────────────┘                │
│                              │                                       │
└──────────────────────────────┼───────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Token Vault (Supabase)                          │
│            (トークン暗号化保存、自動リフレッシュ)                      │
└─────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        External APIs                                 │
│         (Notion, GitHub, Jira, Google Calendar, etc.)                │
└─────────────────────────────────────────────────────────────────────┘
```

### 2層抽象化アーキテクチャ

```
従来のMCP:
  MCP Host → [tool1, tool2, ..., tool100]  ← フラットな100ツール

MCPist:
  MCP Host → Module Registry → Module → [Tools, Resources, Prompts]
                    │
                    ├── notion     → [search, create, update, ...]
                    ├── github     → [list_issues, create_pr, ...]
                    ├── jira       → [get_issue, create_issue, ...]
                    └── ...
```

### 認証3層構造

```
Layer 1: Worker（インフラ保護）
  - JWT署名検証
  - Burst制限（10 req/sec）
  - 未登録ユーザー遮断

Layer 2: Origin（ビジネスロジック）
  - Rate Limit（60 req/min）
  - Quota（月間上限）
  - Credit（従量課金）

Layer 3: Handler（ツール権限）
  - Permission Gate
  - Tool Sieve Filter
  - 権限外ツール非表示
```

---

## 主要な設計決定

### ARD-008: 認証基盤

**決定**: Supabase Auth + Supabase Vault + Edge Function

**理由**:
- 任意のOAuthサービス対応（PKMist等の自作サービス含む）
- user_id統一（同一プラットフォーム）
- Design Liability最小化（セキュリティ核心はSupabaseに委譲）
- コスト$0（Free tier）

### Module Registry設計

**決定**: 2層抽象化 + DAGバッチ + TOON形式

**メタツール**:
- `get_module_schema`: モジュールスキーマ取得（配列入力対応）
- `call_module_tool`: ツール実行
- `batch`: DAG並列実行（変数参照対応）

**TOON形式**: トークン90%削減
```
items[3]{id,title,status}:
  task1,買い物,notStarted
  task2,掃除,completed
  task3,料理,inProgress
```

### Tool Sieve（多層防御）

```
Layer 1: 見せない → 権限外ツールはスキーマに含めない
Layer 2: 実行させない → callメタツール内で権限チェック
Layer 3: 検知する → 権限外呼び出し試行をログ記録
```

---

## プロトタイプ実装状況

| コンポーネント | 状態 | 詳細 |
|---------------|------|------|
| MCPサーバー | ✅ 稼働中 | Go実装、Docker化済み |
| モジュール | ✅ 6種実装 | Notion, Jira, Confluence, GitHub, Supabase, Airtable |
| 管理UI | ✅ 19画面 | Next.js 16 + shadcn/ui |
| 認証 | ⚠️ 簡易実装 | Bearer Token（OAuth 2.0移行予定） |
| UI-API接続 | ❌ 未実装 | モックデータのみ |

---

## 個別開発可能な領域

以下はインターフェース仕様に従って独立開発可能:

| 領域 | 参照仕様 |
|------|----------|
| MCPサーバー | spec-ifc IFC-002, IFC-013, IFC-014 |
| Module Registry | dsn-module-registry.md |
| 各モジュール | Module interface (ExecuteTool, TOON) |
| Token Broker | spec-ifc IFC-020, IFC-021, IFC-022 |
| Tool Sieve | dsn-permission-system.md, spec-ifc IFC-011, IFC-012 |
| 管理UI | spec-dsn §6, dsn-usr-ifc.md |
| Cloudflare Worker | dsn-infrastructure.md, dsn-load-management.md |
| 課金システム | dsn-billing.md |

---

## Phase 1 残タスク

### 実装

- [ ] Fly.io デプロイ
- [ ] Cloudflare Worker実装
- [ ] Cloudflare Terraform化
- [ ] UI-バックエンド接続
- [ ] OAuth 2.0認証実装

### ドキュメント

- [ ] Gateway Secret配布手順
- [ ] SECURITY.md作成
- [ ] OWASP Top 10対応表
- [ ] gosec CI/CD追加

---

## 結論

**設計骨格は完成し、個別開発可能な状態に到達。**

- 全43インターフェース定義済み（spec-ifc）
- コンポーネント責務と境界が明確
- 技術選定完了（ARD-008）
- プロトタイプで検証済み

次週以降は、インターフェース仕様に従った実装フェーズへ移行可能。
